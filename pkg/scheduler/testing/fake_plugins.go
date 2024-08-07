/*
Copyright 2023 The Godel Scheduler Authors.

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

package testing

import (
	"context"
	"fmt"
	"sync/atomic"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	framework "github.com/kubewharf/godel-scheduler/pkg/framework/api"
	"github.com/kubewharf/godel-scheduler/pkg/scheduler/framework/handle"
)

// ErrReasonFake is a fake error message denotes the filter function errored.
const ErrReasonFake = "Nodes failed the fake plugin"

// FalseFilterPlugin is a filter plugin which always return Unschedulable when Filter function is called.
type FalseFilterPlugin struct{}

// Name returns name of the plugin.
func (pl *FalseFilterPlugin) Name() string {
	return "FalseFilter"
}

// Filter invoked at the filter extension point.
func (pl *FalseFilterPlugin) Filter(_ context.Context, pod *v1.Pod, nodeInfo framework.NodeInfo) *framework.Status {
	return framework.NewStatus(framework.Unschedulable, ErrReasonFake)
}

// NewFalseFilterPlugin initializes a FalseFilterPlugin and returns it.
func NewFalseFilterPlugin(_ runtime.Object, _ handle.PodFrameworkHandle, _ handle.PodFrameworkHandle) (framework.Plugin, error) {
	return &FalseFilterPlugin{}, nil
}

// TrueFilterPlugin is a filter plugin which always return Success when Filter function is called.
type TrueFilterPlugin struct{}

// Name returns name of the plugin.
func (pl *TrueFilterPlugin) Name() string {
	return "TrueFilter"
}

// Filter invoked at the filter extension point.
func (pl *TrueFilterPlugin) Filter(_ context.Context, pod *v1.Pod, nodeInfo framework.NodeInfo) *framework.Status {
	return nil
}

// NewTrueFilterPlugin initializes a TrueFilterPlugin and returns it.
func NewTrueFilterPlugin(_ runtime.Object, _ handle.PodFrameworkHandle, _ handle.PodFrameworkHandle) (framework.Plugin, error) {
	return &TrueFilterPlugin{}, nil
}

// FakeFilterPlugin is a test filter plugin to record how many times its Filter() function have
// been called, and it returns different 'Code' depending on its internal 'failedNodeReturnCodeMap'.
type FakeFilterPlugin struct {
	NumFilterCalled         int32
	FailedNodeReturnCodeMap map[string]framework.Code
}

// Name returns name of the plugin.
func (pl *FakeFilterPlugin) Name() string {
	return "FakeFilter"
}

// Filter invoked at the filter extension point.
func (pl *FakeFilterPlugin) Filter(_ context.Context, pod *v1.Pod, nodeInfo framework.NodeInfo) *framework.Status {
	atomic.AddInt32(&pl.NumFilterCalled, 1)

	if returnCode, ok := pl.FailedNodeReturnCodeMap[nodeInfo.GetNode().Name]; ok {
		return framework.NewStatus(returnCode, fmt.Sprintf("injecting failure for pod %v", pod.Name))
	}

	return nil
}

// NewFakeFilterPlugin initializes a fakeFilterPlugin and returns it.
func NewFakeFilterPlugin(failedNodeReturnCodeMap map[string]framework.Code) (framework.Plugin, error) {
	return &FakeFilterPlugin{
		FailedNodeReturnCodeMap: failedNodeReturnCodeMap,
	}, nil
}

// MatchFilterPlugin is a filter plugin which return Success when the evaluated pod and node
// have the same name; otherwise return Unschedulable.
type MatchFilterPlugin struct{}

// Name returns name of the plugin.
func (pl *MatchFilterPlugin) Name() string {
	return "MatchFilter"
}

// Filter invoked at the filter extension point.
func (pl *MatchFilterPlugin) Filter(_ context.Context, pod *v1.Pod, nodeInfo framework.NodeInfo) *framework.Status {
	node := nodeInfo.GetNode()
	if node == nil {
		return framework.NewStatus(framework.Error, "node not found")
	}
	if pod.Name == node.Name {
		return nil
	}
	return framework.NewStatus(framework.Unschedulable, ErrReasonFake)
}

// NewMatchFilterPlugin initializes a MatchFilterPlugin and returns it.
func NewMatchFilterPlugin(_ runtime.Object, _ handle.PodFrameworkHandle, _ handle.PodFrameworkHandle) (framework.Plugin, error) {
	return &MatchFilterPlugin{}, nil
}

// FakePreFilterPlugin is a test filter plugin.
type FakePreFilterPlugin struct {
	Status *framework.Status
}

// Name returns name of the plugin.
func (pl *FakePreFilterPlugin) Name() string {
	return "FakePreFilter"
}

// PreFilter invoked at the PreFilter extension point.
func (pl *FakePreFilterPlugin) PreFilter(_ context.Context, pod *v1.Pod) *framework.Status {
	return pl.Status
}

// // NewFakePreFilterPlugin initializes a fakePreFilterPlugin and returns it.
func NewFakePreFilterPlugin(status *framework.Status) (framework.Plugin, error) {
	return &FakePreFilterPlugin{
		Status: status,
	}, nil
}
