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
	"reflect"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	framework "github.com/kubewharf/godel-scheduler/pkg/framework/api"
	godelcache "github.com/kubewharf/godel-scheduler/pkg/scheduler/cache"
	"github.com/kubewharf/godel-scheduler/pkg/scheduler/cache/handler"
	testingutil "github.com/kubewharf/godel-scheduler/pkg/scheduler/testing"
	testing_helper "github.com/kubewharf/godel-scheduler/pkg/testing-helper"
	podutil "github.com/kubewharf/godel-scheduler/pkg/util/pod"
)

func TestLessTopologyKeyFilter(t *testing.T) {
	regionKeyAffinity := &v1.Affinity{
		PodAffinity: &v1.PodAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{{
				TopologyKey: "kubernetes.io/region",
			}},
		},
	}

	hostNameKeyAffinity := &v1.Affinity{
		PodAffinity: &v1.PodAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{{
				TopologyKey: "kubernetes.io/hostname",
			}},
		},
	}

	tests := []struct {
		pod        *v1.Pod
		topology   map[string]string
		name       string
		wantStatus *framework.Status
	}{
		{
			pod:  &v1.Pod{},
			name: "no thing",
		},
		{
			pod:      &v1.Pod{},
			topology: map[string]string{"kubernetes.io/hostname": "foo"},
			name:     "no less topology key constrain",
		},
		{
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Affinity: regionKeyAffinity,
				},
			},
			name:       "missing topology",
			wantStatus: framework.NewStatus(framework.Unschedulable, "node does not have the required topology key"),
		},
		{
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Affinity: hostNameKeyAffinity,
				},
			},
			topology:   map[string]string{"kubernetes.io/region": "east"},
			name:       "dismatch topology",
			wantStatus: framework.NewStatus(framework.Unschedulable, "node does not have the required topology key"),
		},
		{
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Affinity: hostNameKeyAffinity,
				},
			},
			topology: map[string]string{"kubernetes.io/hostname": "foo"},
			name:     "same host name topology key",
		},
		{
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Affinity: regionKeyAffinity,
				},
			},
			topology: map[string]string{"kubernetes.io/region": "east"},
			name:     "same region name topology key",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			node := v1.Node{ObjectMeta: metav1.ObjectMeta{
				Labels: test.topology,
			}}
			nodeInfo := framework.NewNodeInfo()
			nodeInfo.SetNode(&node)

			cycleState := framework.NewCycleState()
			framework.SetPodResourceTypeState(podutil.GuaranteedPod, cycleState)
			p, _ := New(nil, nil)
			gotStatus := p.(framework.FilterPlugin).Filter(context.Background(), cycleState, test.pod, nodeInfo)
			if !reflect.DeepEqual(gotStatus, test.wantStatus) {
				t.Errorf("status does not match: %v, want: %v", gotStatus, test.wantStatus)
			}
		})
	}
}

func getNodes(scheduledPodNum map[int]int) []framework.NodeInfo {
	basicNodes := []*v1.Node{
		{ObjectMeta: metav1.ObjectMeta{Name: "machine1", Labels: map[string]string{"kubernetes.io/region": "east", "kubernetes.io/hostname": "machine1"}}},
		{ObjectMeta: metav1.ObjectMeta{Name: "machine2", Labels: map[string]string{"kubernetes.io/region": "east", "kubernetes.io/hostname": "machine2"}}},
		{ObjectMeta: metav1.ObjectMeta{Name: "machine3", Labels: map[string]string{"kubernetes.io/region": "west", "kubernetes.io/hostname": "machine3"}}},
	}

	var nodeInfos []framework.NodeInfo

	for i, basicNode := range basicNodes {
		num := scheduledPodNum[i]
		pods := make([]*v1.Pod, num)
		for j := 0; j < num; j++ {
			pods[j] = testing_helper.MakePod().UID(fmt.Sprintf("%d-%d", i, j)).Obj()
		}
		nodeInfo := framework.NewNodeInfo(pods...)
		nodeInfo.SetNode(basicNode)

		nodeInfos = append(nodeInfos, nodeInfo)
	}
	return nodeInfos
}

func TestLessTopologyKeyScore(t *testing.T) {
	regionKeyAffinity := &v1.Affinity{
		PodAffinity: &v1.PodAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{{
				TopologyKey: "kubernetes.io/region",
			}},
		},
	}

	hostNameKeyAffinity := &v1.Affinity{
		PodAffinity: &v1.PodAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{{
				TopologyKey: "kubernetes.io/hostname",
			}},
		},
	}

	tests := []struct {
		pod          *v1.Pod
		expectedList framework.NodeScoreList
		nodes        []framework.NodeInfo
		name         string
	}{
		{
			pod:          &v1.Pod{},
			nodes:        getNodes(map[int]int{}),
			expectedList: []framework.NodeScore{{Name: "machine1", Score: 0}, {Name: "machine2", Score: 0}, {Name: "machine3", Score: 0}},
			name:         "all machines are same priority as LessTopologyKey is nil",
		},
		{
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Affinity: regionKeyAffinity,
				},
			},
			nodes:        getNodes(map[int]int{}),
			expectedList: []framework.NodeScore{{Name: "machine1", Score: framework.MaxNodeScore}, {Name: "machine2", Score: framework.MaxNodeScore}, {Name: "machine3", Score: framework.MaxNodeScore}},
			name:         "region topology constrain: all machines are max priority as scheduled pod is empty",
		},
		{
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Affinity: hostNameKeyAffinity,
				},
			},
			nodes:        getNodes(map[int]int{}),
			expectedList: []framework.NodeScore{{Name: "machine1", Score: framework.MaxNodeScore}, {Name: "machine2", Score: framework.MaxNodeScore}, {Name: "machine3", Score: framework.MaxNodeScore}},
			name:         "hostname topology constrain: all machines are max priority as scheduled pod is empty",
		},
		{
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Affinity: hostNameKeyAffinity,
				},
			},
			nodes:        getNodes(map[int]int{0: 1}),
			expectedList: []framework.NodeScore{{Name: "machine1", Score: 0}, {Name: "machine2", Score: framework.MaxNodeScore}, {Name: "machine3", Score: framework.MaxNodeScore}},
			name:         "hostname topology constrain: machine1 with one pod",
		},
		{
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Affinity: hostNameKeyAffinity,
				},
			},
			nodes:        getNodes(map[int]int{0: 1, 1: 5, 2: 4}),
			expectedList: []framework.NodeScore{{Name: "machine1", Score: framework.MaxNodeScore - framework.MaxNodeScore/5}, {Name: "machine2", Score: 0}, {Name: "machine3", Score: framework.MaxNodeScore / 5}},
			name:         "hostname topology constrain: machine1 with height score",
		},
		{
			pod: &v1.Pod{
				Spec: v1.PodSpec{
					Affinity: regionKeyAffinity,
				},
			},
			nodes:        getNodes(map[int]int{0: 1, 1: 5, 2: 4}),
			expectedList: []framework.NodeScore{{Name: "machine1", Score: 0}, {Name: "machine2", Score: 0}, {Name: "machine3", Score: framework.MaxNodeScore - framework.MaxNodeScore*2/3}},
			name:         "region topology constrain: machine3 with height score",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cycleState := framework.NewCycleState()
			framework.SetPodResourceTypeState(podutil.GuaranteedPod, cycleState)
			cache := godelcache.New(handler.MakeCacheHandlerWrapper().
				SchedulerName("").SchedulerType("").SubCluster(framework.DefaultSubCluster).
				TTL(time.Second).Period(10 * time.Second).StopCh(make(<-chan struct{})).
				EnableStore("PreemptionStore").
				Obj())
			snapshot := godelcache.NewEmptySnapshot(handler.MakeCacheHandlerWrapper().
				SubCluster(framework.DefaultSubCluster).SwitchType(framework.DefaultSubClusterSwitchType).
				EnableStore("PreemptionStore").
				Obj())

			for _, n := range test.nodes {
				cache.AddNode(n.GetNode())
			}
			cache.UpdateSnapshot(snapshot)

			fh, _ := testingutil.NewSchedulerFrameworkHandle(nil, nil, nil, nil, nil, snapshot, nil, nil, nil, nil)
			p, _ := New(nil, fh)

			p.(framework.PreScorePlugin).PreScore(context.Background(), cycleState, test.pod, test.nodes)
			var gotList framework.NodeScoreList
			for _, n := range test.nodes {
				nodeName := n.GetNodeName()
				score, status := p.(framework.ScorePlugin).Score(context.Background(), cycleState, test.pod, nodeName)
				if !status.IsSuccess() {
					t.Errorf("unexpected error: %v", status)
				}
				gotList = append(gotList, framework.NodeScore{Name: nodeName, Score: score})
			}

			status := p.(framework.ScorePlugin).ScoreExtensions().NormalizeScore(context.Background(), cycleState, test.pod, gotList)
			if !status.IsSuccess() {
				t.Errorf("unexpected error: %v", status)
			}

			if !reflect.DeepEqual(test.expectedList, gotList) {
				t.Errorf("expected:\n\t%+v,\ngot:\n\t%+v", test.expectedList, gotList)
			}
		})
	}
}
