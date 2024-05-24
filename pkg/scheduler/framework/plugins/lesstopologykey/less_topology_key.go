/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package lesstopology

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"

	framework "github.com/kubewharf/godel-scheduler/pkg/framework/api"
	pluginhelper "github.com/kubewharf/godel-scheduler/pkg/plugins/helper"
)

const (
	// Name is the name of the plugin used in the plugin registry and configurations.
	Name             = "LessTopologyKey"
	preScoreStateKey = "PreScore" + Name
)

type LessTopologyKey struct {
	handle framework.SchedulerFrameworkHandle
}

var (
	_ framework.FilterPlugin   = &LessTopologyKey{}
	_ framework.PreScorePlugin = &LessTopologyKey{}
	_ framework.ScorePlugin    = &LessTopologyKey{}
)

// preScoreState computed at PreScore and used at Score.
type preScoreState struct {
	topologyScore map[string]int64
}

// Clone implements the mandatory Clone interface. We don't really copy the data since
// there is no need for that.
func (s *preScoreState) Clone() framework.StateData {
	return s
}

// Name returns name of the plugin. It is used in logs, etc.
func (pl *LessTopologyKey) Name() string {
	return Name
}

func (pl *LessTopologyKey) getLessTopologyKey(pod *v1.Pod) string {
	if pod.Spec.Affinity == nil ||
		pod.Spec.Affinity.PodAffinity == nil ||
		pod.Spec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil ||
		len(pod.Spec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution) == 0 {
		return ""
	}
	return pod.Spec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0].TopologyKey
}

// Filter invoked at the filter extension point.
func (pl *LessTopologyKey) Filter(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeInfo framework.NodeInfo) *framework.Status {
	lessTopologyKey := pl.getLessTopologyKey(pod)
	if lessTopologyKey != "" {
		node := nodeInfo.GetNode()
		if node == nil {
			return framework.NewStatus(framework.Error, "node not found")
		}

		if _, exists := node.Labels[lessTopologyKey]; !exists {
			klog.V(1).InfoS("LessTopologyKey Filter refuse", "node", node.Name, "topologyKey", lessTopologyKey)
			return framework.NewStatus(framework.Unschedulable, "node does not have the required topology key")
		}
		klog.V(1).InfoS("LessTopologyKey Filter accept", "node", node.Name, "topologyKey", lessTopologyKey)
	}

	return nil
}

// PreScore builds and writes cycle state used by Score and NormalizeScore.
func (pl *LessTopologyKey) PreScore(_ context.Context, cycleState *framework.CycleState, pod *v1.Pod, nodes []framework.NodeInfo) *framework.Status {
	if len(nodes) == 0 {
		// No nodes to score.
		return nil
	}

	lessTopologyKey := pl.getLessTopologyKey(pod)
	if lessTopologyKey == "" {
		cycleState.Write(preScoreStateKey, &preScoreState{
			topologyScore: make(map[string]int64),
		})
		return nil
	}

	state := &preScoreState{
		topologyScore: make(map[string]int64),
	}

	for _, nodeInfo := range nodes {
		node := nodeInfo.GetNode()
		if node == nil {
			klog.ErrorS(nil, "node not found", "nodeInfo", nodeInfo)
		}

		if value, exists := node.Labels[lessTopologyKey]; exists {
			state.topologyScore[value] += int64(nodeInfo.NumPods())
		}
	}
	klog.V(1).InfoS("LessTopologyKey preScoreState: ", "state.topologyScore", state.topologyScore)
	cycleState.Write(preScoreStateKey, state)
	return nil
}

func getPreScoreState(cycleState *framework.CycleState) (*preScoreState, error) {
	c, err := cycleState.Read(preScoreStateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to read %q from cycleState: %w", preScoreStateKey, err)
	}

	s, ok := c.(*preScoreState)
	if !ok {
		return nil, fmt.Errorf("%+v  convert to interpodaffinity.preScoreState error", c)
	}
	return s, nil
}

// Score invoked at the Score extension point.
func (pl *LessTopologyKey) Score(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) (int64, *framework.Status) {
	nodeInfo, err := pl.handle.SnapshotSharedLister().NodeInfos().Get(nodeName)
	if err != nil {
		return framework.MaxNodeScore, framework.NewStatus(framework.Error, fmt.Sprintf("getting node %q from Snapshot: %v", nodeName, err))
	}

	node := nodeInfo.GetNode()
	if node == nil {
		return framework.MaxNodeScore, framework.NewStatus(framework.Error, fmt.Sprintf("getting node %q from NodeInfo: %v", nodeName, err))
	}

	lessTopologyKey := pl.getLessTopologyKey(pod)
	if lessTopologyKey == "" {
		return framework.MaxNodeScore, nil
	}

	preScoreState, err := getPreScoreState(state)
	if err != nil {
		return framework.MaxNodeScore, framework.AsStatus(err)
	}
	if value, exist := node.Labels[lessTopologyKey]; exist {
		klog.V(1).InfoS("LessTopologyKey Score For Node: ", "node", node.Name, "lessTopologyKey", lessTopologyKey, "value", value, "score", preScoreState.topologyScore[value])
		return preScoreState.topologyScore[value], nil
	} else {
		return framework.MaxNodeScore, nil
	}
}

// NormalizeScore invoked after scoring all nodes.
func (pl *LessTopologyKey) NormalizeScore(ctx context.Context, state *framework.CycleState, pod *v1.Pod, scores framework.NodeScoreList) *framework.Status {
	status := pluginhelper.DefaultNormalizeScore(framework.MaxNodeScore, true, scores)
	klog.V(1).InfoS("LessTopologyKey Score After Normalize", "scores", scores)
	return status
}

// ScoreExtensions of the Score plugin.
func (pl *LessTopologyKey) ScoreExtensions() framework.ScoreExtensions {
	return pl
}

// New initializes a new plugin and returns it.
func New(_ runtime.Object, h framework.SchedulerFrameworkHandle) (framework.Plugin, error) {
	return &LessTopologyKey{handle: h}, nil
}
