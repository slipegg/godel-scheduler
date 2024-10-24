/*
Copyright 2024 The Godel Scheduler Authors.

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

package helper

import (
	"fmt"

	"github.com/kubewharf/godel-scheduler/pkg/binder/framework/handle"
	framework "github.com/kubewharf/godel-scheduler/pkg/framework/api"
	podutil "github.com/kubewharf/godel-scheduler/pkg/util/pod"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	AllNodeInfoKey = "AllNodeInfo"
)

type AllNodeInfos struct {
	NodeInfos []framework.NodeInfo
}

func (f AllNodeInfos) Clone() framework.StateData {
	clonedNodeInfos := make([]framework.NodeInfo, len(f.NodeInfos))

	copy(clonedNodeInfos, f.NodeInfos)

	return AllNodeInfos{
		NodeInfos: clonedNodeInfos,
	}
}

func GetAllNodeInfos(frameworkHandle handle.BinderFrameworkHandle) (*AllNodeInfos, error) {
	nodeInfoMap := map[string]framework.NodeInfo{}
	for _, podLauncher := range podutil.PodLanucherTypes {
		if podLauncher == podutil.Kubelet && frameworkHandle.SharedInformerFactory() != nil {
			allV1Nodes, err := frameworkHandle.SharedInformerFactory().Core().V1().Nodes().Lister().List(labels.Everything())
			if err != nil {
				return nil, err
			}

			for _, node := range allV1Nodes {
				nodeInfo := frameworkHandle.GetNodeInfo(node.Name)
				nodeInfoMap[nodeInfo.GetNodeName()] = nodeInfo
			}
		} else if podLauncher == podutil.NodeManager && frameworkHandle.CRDSharedInformerFactory() != nil {
			allNMNodes, err := frameworkHandle.CRDSharedInformerFactory().Node().V1alpha1().NMNodes().Lister().List(labels.Everything())
			if err != nil {
				return nil, err
			}

			for _, node := range allNMNodes {
				nodeInfo := frameworkHandle.GetNodeInfo(node.Name)
				nodeInfoMap[nodeInfo.GetNodeName()] = nodeInfo
			}
		}
	}

	nodeInfos := make([]framework.NodeInfo, 0, len(nodeInfoMap))
	for _, nodeInfo := range nodeInfoMap {
		nodeInfos = append(nodeInfos, nodeInfo)
	}

	return &AllNodeInfos{
		NodeInfos: nodeInfos,
	}, nil
}

func WriteAllNodeInfos(cycleState *framework.CycleState, frameworkHandle handle.BinderFrameworkHandle) error {
	nodeInfoMap := map[string]framework.NodeInfo{}
	for _, podLauncher := range podutil.PodLanucherTypes {
		if podLauncher == podutil.Kubelet && frameworkHandle.SharedInformerFactory() != nil {
			allV1Nodes, err := frameworkHandle.SharedInformerFactory().Core().V1().Nodes().Lister().List(labels.Everything())
			if err != nil {
				return err
			}

			for _, node := range allV1Nodes {
				nodeInfo := frameworkHandle.GetNodeInfo(node.Name)
				nodeInfoMap[nodeInfo.GetNodeName()] = nodeInfo
			}
		} else if podLauncher == podutil.NodeManager && frameworkHandle.CRDSharedInformerFactory() != nil {
			allNMNodes, err := frameworkHandle.CRDSharedInformerFactory().Node().V1alpha1().NMNodes().Lister().List(labels.Everything())
			if err != nil {
				return err
			}

			for _, node := range allNMNodes {
				nodeInfo := frameworkHandle.GetNodeInfo(node.Name)
				nodeInfoMap[nodeInfo.GetNodeName()] = nodeInfo
			}
		}
	}

	nodeInfos := make([]framework.NodeInfo, 0, len(nodeInfoMap))
	for _, nodeInfo := range nodeInfoMap {
		nodeInfos = append(nodeInfos, nodeInfo)
	}

	allNodeInfoData := &AllNodeInfos{
		NodeInfos: nodeInfos,
	}
	cycleState.Write(AllNodeInfoKey, allNodeInfoData)

	return nil
}

func ReadAllNodeInfos(cycleState *framework.CycleState) ([]framework.NodeInfo, error) {
	nodeInfoData, err := cycleState.Read(AllNodeInfoKey)
	if err != nil {
		return nil, err
	}

	allNodeInfos, ok := nodeInfoData.(*AllNodeInfos)
	if !ok {
		return nil, fmt.Errorf("%+v convert to helper.AllNodeInfos error", nodeInfoData)
	}
	return allNodeInfos.NodeInfos, nil
}
