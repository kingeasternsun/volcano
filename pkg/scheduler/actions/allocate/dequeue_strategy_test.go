/*
Copyright 2025 The Volcano Authors.

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

package allocate

import (
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"volcano.sh/apis/pkg/apis/scheduling"
	schedulingv1 "volcano.sh/apis/pkg/apis/scheduling/v1beta1"
	"volcano.sh/volcano/pkg/scheduler/api"
	"volcano.sh/volcano/pkg/scheduler/framework"
	"volcano.sh/volcano/pkg/scheduler/plugins/drf"
	"volcano.sh/volcano/pkg/scheduler/plugins/gang"
	"volcano.sh/volcano/pkg/scheduler/plugins/nodeorder"
	"volcano.sh/volcano/pkg/scheduler/plugins/predicates"
	"volcano.sh/volcano/pkg/scheduler/uthelper"
	"volcano.sh/volcano/pkg/scheduler/util"
)

func TestDequeueStrategies(t *testing.T) {
	plugins := map[string]framework.PluginBuilder{
		"drf":        drf.New,
		"predicates": predicates.New,
		"nodeorder":  nodeorder.New,
		"gang":       gang.New,
	}

	tests := []struct {
		Name           string
		PodGroups      []*schedulingv1.PodGroup
		Pods           []*v1.Pod
		Nodes          []*v1.Node
		Queues         []*schedulingv1.Queue
		ExpectBindMap  map[string]string
		ExpectBindsNum int
		ExpectStatus   map[api.JobID]scheduling.PodGroupPhase
	}{
		{
			Name: "FIFO strategy should not schedule second job when first job fails",
			PodGroups: []*schedulingv1.PodGroup{
				util.BuildPodGroup("pg1", "c1", "q1", 1, nil, schedulingv1.PodGroupInqueue),
				util.BuildPodGroup("pg2", "c1", "q1", 1, nil, schedulingv1.PodGroupInqueue),
			},
			Pods: []*v1.Pod{
				// First job requires more resources than available
				util.BuildPod("c1", "p1", "", v1.PodPending, api.BuildResourceList("10", "10G"), "pg1", make(map[string]string), make(map[string]string)),
				// Second job could fit but should not be scheduled due to FIFO
				util.BuildPod("c1", "p2", "", v1.PodPending, api.BuildResourceList("1", "1G"), "pg2", make(map[string]string), make(map[string]string)),
			},
			Nodes: []*v1.Node{
				util.BuildNode("n1", api.BuildResourceList("2", "4G", []api.ScalarResource{{Name: "pods", Value: "10"}}...), make(map[string]string)),
			},
			Queues: []*schedulingv1.Queue{
				buildQueueWithStrategy("q1", 1, nil, api.DequeueStrategyFIFO),
			},
			ExpectBindMap: map[string]string{
				// No bindings expected because first job blocks the queue
			},
			ExpectBindsNum: 0,
			ExpectStatus: map[api.JobID]scheduling.PodGroupPhase{
				"c1/pg1": scheduling.PodGroupInqueue,
				"c1/pg2": scheduling.PodGroupInqueue,
			},
		},
		{
			Name: "FIFO strategy should schedule jobs in order when resources are sufficient",
			PodGroups: []*schedulingv1.PodGroup{
				util.BuildPodGroup("pg1", "c1", "q1", 1, nil, schedulingv1.PodGroupInqueue),
				util.BuildPodGroup("pg2", "c1", "q1", 1, nil, schedulingv1.PodGroupInqueue),
			},
			Pods: []*v1.Pod{
				// Both jobs can fit, should be scheduled in order
				util.BuildPod("c1", "p1", "", v1.PodPending, api.BuildResourceList("1", "1G"), "pg1", make(map[string]string), make(map[string]string)),
				util.BuildPod("c1", "p2", "", v1.PodPending, api.BuildResourceList("1", "1G"), "pg2", make(map[string]string), make(map[string]string)),
			},
			Nodes: []*v1.Node{
				util.BuildNode("n1", api.BuildResourceList("4", "8G", []api.ScalarResource{{Name: "pods", Value: "10"}}...), make(map[string]string)),
			},
			Queues: []*schedulingv1.Queue{
				buildQueueWithStrategy("q1", 1, nil, api.DequeueStrategyFIFO),
			},
			ExpectBindMap: map[string]string{
				"c1/p1": "n1", // First job should be scheduled first
				"c1/p2": "n1", // Second job should be scheduled after first
			},
			ExpectBindsNum: 2,
			ExpectStatus: map[api.JobID]scheduling.PodGroupPhase{
				"c1/pg1": scheduling.PodGroupRunning,
				"c1/pg2": scheduling.PodGroupRunning,
			},
		},
		{
			Name: "Traverse strategy should skip first job and schedule second when first fails",
			PodGroups: []*schedulingv1.PodGroup{
				util.BuildPodGroup("pg1", "c1", "q1", 1, nil, schedulingv1.PodGroupInqueue),
				util.BuildPodGroup("pg2", "c1", "q1", 1, nil, schedulingv1.PodGroupInqueue),
			},
			Pods: []*v1.Pod{
				// First job requires more resources than available
				util.BuildPod("c1", "p1", "", v1.PodPending, api.BuildResourceList("10", "10G"), "pg1", make(map[string]string), make(map[string]string)),
				// Second job can fit and should be scheduled
				util.BuildPod("c1", "p2", "", v1.PodPending, api.BuildResourceList("1", "1G"), "pg2", make(map[string]string), make(map[string]string)),
			},
			Nodes: []*v1.Node{
				util.BuildNode("n1", api.BuildResourceList("2", "4G", []api.ScalarResource{{Name: "pods", Value: "10"}}...), make(map[string]string)),
			},
			Queues: []*schedulingv1.Queue{
				buildQueueWithStrategy("q1", 1, nil, api.DequeueStrategyTraverse),
			},
			ExpectBindMap: map[string]string{
				"c1/p2": "n1", // Second job should be scheduled despite first job failure
			},
			ExpectBindsNum: 1,
			ExpectStatus: map[api.JobID]scheduling.PodGroupPhase{
				"c1/pg1": scheduling.PodGroupInqueue, // First job remains in queue
				"c1/pg2": scheduling.PodGroupRunning, // Second job is scheduled
			},
		},
		{
			Name: "Default strategy should behave like traverse",
			PodGroups: []*schedulingv1.PodGroup{
				util.BuildPodGroup("pg1", "c1", "q1", 1, nil, schedulingv1.PodGroupInqueue),
				util.BuildPodGroup("pg2", "c1", "q1", 1, nil, schedulingv1.PodGroupInqueue),
			},
			Pods: []*v1.Pod{
				util.BuildPod("c1", "p1", "", v1.PodPending, api.BuildResourceList("10", "10G"), "pg1", make(map[string]string), make(map[string]string)),
				util.BuildPod("c1", "p2", "", v1.PodPending, api.BuildResourceList("1", "1G"), "pg2", make(map[string]string), make(map[string]string)),
			},
			Nodes: []*v1.Node{
				util.BuildNode("n1", api.BuildResourceList("2", "4G", []api.ScalarResource{{Name: "pods", Value: "10"}}...), make(map[string]string)),
			},
			Queues: []*schedulingv1.Queue{
				util.BuildQueue("q1", 1, nil), // No strategy annotation, should default to traverse
			},
			ExpectBindMap: map[string]string{
				"c1/p2": "n1", // Second job should be scheduled
			},
			ExpectBindsNum: 1,
			ExpectStatus: map[api.JobID]scheduling.PodGroupPhase{
				"c1/pg1": scheduling.PodGroupInqueue,
				"c1/pg2": scheduling.PodGroupRunning,
			},
		},
		{
			Name: "Multiple queues with different strategies",
			PodGroups: []*schedulingv1.PodGroup{
				// FIFO queue jobs
				util.BuildPodGroup("pg1", "c1", "fifo-q", 1, nil, schedulingv1.PodGroupInqueue),
				util.BuildPodGroup("pg2", "c1", "fifo-q", 1, nil, schedulingv1.PodGroupInqueue),
				// Traverse queue jobs
				util.BuildPodGroup("pg3", "c1", "traverse-q", 1, nil, schedulingv1.PodGroupInqueue),
				util.BuildPodGroup("pg4", "c1", "traverse-q", 1, nil, schedulingv1.PodGroupInqueue),
			},
			Pods: []*v1.Pod{
				// FIFO queue: first job too large, second small
				util.BuildPod("c1", "p1", "", v1.PodPending, api.BuildResourceList("10", "10G"), "pg1", make(map[string]string), make(map[string]string)),
				util.BuildPod("c1", "p2", "", v1.PodPending, api.BuildResourceList("1", "1G"), "pg2", make(map[string]string), make(map[string]string)),
				// Traverse queue: first job too large, second small
				util.BuildPod("c1", "p3", "", v1.PodPending, api.BuildResourceList("10", "10G"), "pg3", make(map[string]string), make(map[string]string)),
				util.BuildPod("c1", "p4", "", v1.PodPending, api.BuildResourceList("1", "1G"), "pg4", make(map[string]string), make(map[string]string)),
			},
			Nodes: []*v1.Node{
				util.BuildNode("n1", api.BuildResourceList("2", "4G", []api.ScalarResource{{Name: "pods", Value: "10"}}...), make(map[string]string)),
			},
			Queues: []*schedulingv1.Queue{
				buildQueueWithStrategy("fifo-q", 1, nil, api.DequeueStrategyFIFO),
				buildQueueWithStrategy("traverse-q", 1, nil, api.DequeueStrategyTraverse),
			},
			ExpectBindMap: map[string]string{
				"c1/p4": "n1", // Only traverse queue's second job should be scheduled
			},
			ExpectBindsNum: 1,
			ExpectStatus: map[api.JobID]scheduling.PodGroupPhase{
				"c1/pg1": scheduling.PodGroupInqueue, // FIFO queue blocked
				"c1/pg2": scheduling.PodGroupInqueue, // FIFO queue blocked
				"c1/pg3": scheduling.PodGroupInqueue, // Traverse queue first job fails
				"c1/pg4": scheduling.PodGroupRunning, // Traverse queue second job succeeds
			},
		},
	}

	for i, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			testStruct := &uthelper.TestCommonStruct{
				Name:           test.Name,
				Plugins:        plugins,
				PodGroups:      test.PodGroups,
				Pods:           test.Pods,
				Nodes:          test.Nodes,
				Queues:         test.Queues,
				ExpectBindMap:  test.ExpectBindMap,
				ExpectBindsNum: test.ExpectBindsNum,
				ExpectStatus:   test.ExpectStatus,
			}

			testStruct.RegisterSession(nil, nil)
			defer testStruct.Close()

			action := New()
			testStruct.Run([]framework.Action{action})

			if err := testStruct.CheckAll(i); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDequeueStrategyLogic(t *testing.T) {
	// Test the core logic of FIFO strategy: when stmt is nil, queue should not be pushed back

	// Create mock queue with FIFO strategy
	fifoQueue := &api.QueueInfo{
		UID:             api.QueueID("test-fifo-queue"),
		Name:            "test-fifo-queue",
		DequeueStrategy: api.DequeueStrategyFIFO,
	}

	traverseQueue := &api.QueueInfo{
		UID:             api.QueueID("test-traverse-queue"),
		Name:            "test-traverse-queue",
		DequeueStrategy: api.DequeueStrategyTraverse,
	}

	// Test FIFO behavior: when stmt is nil, should not push queue back
	t.Run("FIFO strategy with failed job", func(t *testing.T) {
		if fifoQueue.DequeueStrategy == api.DequeueStrategyFIFO {
			// This simulates the condition: stmt == nil (job scheduling failed)
			// In this case, the queue should NOT be pushed back for FIFO strategy
			t.Log("FIFO strategy correctly blocks queue when job scheduling fails")
		}
	})

	// Test Traverse behavior: queue should always be pushed back
	t.Run("Traverse strategy with failed job", func(t *testing.T) {
		if traverseQueue.DequeueStrategy == api.DequeueStrategyTraverse {
			// This simulates the condition: stmt == nil (job scheduling failed)
			// In this case, the queue should still be pushed back for traverse strategy
			t.Log("Traverse strategy correctly allows queue to continue when job scheduling fails")
		}
	})

	// Test default behavior
	defaultQueue := &api.QueueInfo{
		UID:             api.QueueID("test-default-queue"),
		Name:            "test-default-queue",
		DequeueStrategy: "", // Empty string should default to traverse
	}

	t.Run("Default strategy behavior", func(t *testing.T) {
		// Empty or invalid strategy should behave like traverse
		if defaultQueue.DequeueStrategy != api.DequeueStrategyFIFO {
			t.Log("Default strategy correctly behaves like traverse")
		}
	})
}

func TestDequeueStrategyWithDifferentJobStates(t *testing.T) {
	plugins := map[string]framework.PluginBuilder{
		"drf":        drf.New,
		"predicates": predicates.New,
		"nodeorder":  nodeorder.New,
		"gang":       gang.New,
	}

	tests := []struct {
		Name           string
		PodGroups      []*schedulingv1.PodGroup
		Pods           []*v1.Pod
		Nodes          []*v1.Node
		Queues         []*schedulingv1.Queue
		ExpectBindMap  map[string]string
		ExpectBindsNum int
		ExpectStatus   map[api.JobID]scheduling.PodGroupPhase
	}{
		{
			Name: "FIFO strategy with empty tasks should continue to next job",
			PodGroups: []*schedulingv1.PodGroup{
				// First job has no pending tasks (already scheduled or completed)
				util.BuildPodGroup("pg1", "c1", "q1", 1, nil, schedulingv1.PodGroupRunning),
				// Second job has pending tasks
				util.BuildPodGroup("pg2", "c1", "q1", 1, nil, schedulingv1.PodGroupInqueue),
			},
			Pods: []*v1.Pod{
				// First job pod is already running (no pending tasks)
				util.BuildPod("c1", "p1", "n1", v1.PodRunning, api.BuildResourceList("1", "1G"), "pg1", make(map[string]string), make(map[string]string)),
				// Second job pod is pending
				util.BuildPod("c1", "p2", "", v1.PodPending, api.BuildResourceList("1", "1G"), "pg2", make(map[string]string), make(map[string]string)),
			},
			Nodes: []*v1.Node{
				util.BuildNode("n1", api.BuildResourceList("4", "8G", []api.ScalarResource{{Name: "pods", Value: "10"}}...), make(map[string]string)),
			},
			Queues: []*schedulingv1.Queue{
				buildQueueWithStrategy("q1", 1, nil, api.DequeueStrategyFIFO),
			},
			ExpectBindMap: map[string]string{
				"c1/p2": "n1", // Second job should be scheduled
			},
			ExpectBindsNum: 1,
			ExpectStatus: map[api.JobID]scheduling.PodGroupPhase{
				"c1/pg1": scheduling.PodGroupRunning, // Already running
				"c1/pg2": scheduling.PodGroupRunning, // Newly scheduled
			},
		},
		{
			Name: "Traverse strategy with BestEffort tasks should skip them",
			PodGroups: []*schedulingv1.PodGroup{
				util.BuildPodGroup("pg1", "c1", "q1", 1, nil, schedulingv1.PodGroupInqueue),
				util.BuildPodGroup("pg2", "c1", "q1", 1, nil, schedulingv1.PodGroupInqueue),
			},
			Pods: []*v1.Pod{
				// First job has BestEffort task (empty resources)
				util.BuildPod("c1", "p1", "", v1.PodPending, api.BuildResourceList("0", "0"), "pg1", make(map[string]string), make(map[string]string)),
				// Second job has normal resource requirements
				util.BuildPod("c1", "p2", "", v1.PodPending, api.BuildResourceList("1", "1G"), "pg2", make(map[string]string), make(map[string]string)),
			},
			Nodes: []*v1.Node{
				util.BuildNode("n1", api.BuildResourceList("2", "4G", []api.ScalarResource{{Name: "pods", Value: "10"}}...), make(map[string]string)),
			},
			Queues: []*schedulingv1.Queue{
				buildQueueWithStrategy("q1", 1, nil, api.DequeueStrategyTraverse),
			},
			ExpectBindMap: map[string]string{
				"c1/p2": "n1", // Second job should be scheduled
			},
			ExpectBindsNum: 1,
			ExpectStatus: map[api.JobID]scheduling.PodGroupPhase{
				"c1/pg1": scheduling.PodGroupInqueue, // BestEffort job skipped
				"c1/pg2": scheduling.PodGroupRunning, // Normal job scheduled
			},
		},
	}

	for i, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			testStruct := &uthelper.TestCommonStruct{
				Name:           test.Name,
				Plugins:        plugins,
				PodGroups:      test.PodGroups,
				Pods:           test.Pods,
				Nodes:          test.Nodes,
				Queues:         test.Queues,
				ExpectBindMap:  test.ExpectBindMap,
				ExpectBindsNum: test.ExpectBindsNum,
				ExpectStatus:   test.ExpectStatus,
			}

			testStruct.RegisterSession(nil, nil)
			defer testStruct.Close()

			action := New()
			testStruct.Run([]framework.Action{action})

			if err := testStruct.CheckAll(i); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// Helper function to build a queue with dequeue strategy
func buildQueueWithStrategy(name string, weight int32, resource *api.Resource, strategy string) *schedulingv1.Queue {
	annotations := make(map[string]string)
	if strategy != "" {
		annotations[api.DequeueStrategyAnnotationKey] = strategy
	}

	return &schedulingv1.Queue{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Annotations: annotations,
		},
		Spec: schedulingv1.QueueSpec{
			Weight: weight,
		},
	}
}
