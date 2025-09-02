/*
Copyright(C)2025. Huawei Technologies Co.,Ltd. All rights reserved.

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

/*
Package ascend910a3 is using for A3 affinity schedule.
*/
package ascend910a3

import (
	"testing"

	"volcano.sh/volcano/pkg/scheduler/api"
	"volcano.sh/volcano/pkg/scheduler/plugins/ascend-volcano-plugin/common/util"
	"volcano.sh/volcano/pkg/scheduler/plugins/ascend-volcano-plugin/internal/npu/base"
)

func TestBase910A3_CheckReqNPUEqualNodeNPU(t *testing.T) {
	tests := []struct {
		name           string
		base910A3      *Base910A3
		expectedResult *api.ValidateResult
	}{
		{
			name: "01 All tasks have correct NPU number - should pass",
			base910A3: &Base910A3{
				NPUHandler: base.NPUHandler{
					MaxNodeNPUNum: 16,
					SchedulerJobAttr: util.SchedulerJobAttr{
						NPUJob: &util.NPUJob{
							Tasks: map[api.TaskID]util.NPUTask{
								"task1": {
									ReqNPUNum: 16,
								},
								"task2": {
									ReqNPUNum: 16,
								},
							},
						},
					},
				},
			},
			expectedResult: nil, // Should pass, return nil
		},
		{
			name: "02 Task with zero NPU and scheduler spec annotation - should pass",
			base910A3: &Base910A3{
				NPUHandler: base.NPUHandler{
					MaxNodeNPUNum: 16,
					SchedulerJobAttr: util.SchedulerJobAttr{
						NPUJob: &util.NPUJob{
							Tasks: map[api.TaskID]util.NPUTask{
								"task1": {
									ReqNPUNum: 0,
									Annotation: map[string]string{
										taskSpec: schedulerSpec,
									},
								},
							},
						},
					},
				},
			},
			expectedResult: nil, // Should pass, return nil
		},
		{
			name: "03 Task with zero NPU and skip ascend plugin annotation - should pass",
			base910A3: &Base910A3{
				NPUHandler: base.NPUHandler{
					MaxNodeNPUNum: 16,
					SchedulerJobAttr: util.SchedulerJobAttr{
						NPUJob: &util.NPUJob{
							Tasks: map[api.TaskID]util.NPUTask{
								"task1": {
									ReqNPUNum: 0,
									Annotation: map[string]string{
										skipAscendPlugin: skipEnabled,
									},
								},
							},
						},
					},
				},
			},
			expectedResult: nil, // Should pass, return nil
		},
		{
			name: "04 Task with zero NPU but no special annotations - should fail",
			base910A3: &Base910A3{
				NPUHandler: base.NPUHandler{
					MaxNodeNPUNum: 16,
					SchedulerJobAttr: util.SchedulerJobAttr{
						NPUJob: &util.NPUJob{
							Tasks: map[api.TaskID]util.NPUTask{
								"task1": {
									ReqNPUNum:  0,
									Annotation: map[string]string{},
								},
							},
						},
					},
				},
			},
			expectedResult: &api.ValidateResult{
				Pass:    false,
				Reason:  JobCheckFailedReason,
				Message: "distributed job require npu 16, instead of 0",
			},
		},
		{
			name: "05 Task with incorrect NPU number - should fail",
			base910A3: &Base910A3{
				NPUHandler: base.NPUHandler{
					MaxNodeNPUNum: 16,
					SchedulerJobAttr: util.SchedulerJobAttr{
						NPUJob: &util.NPUJob{
							Tasks: map[api.TaskID]util.NPUTask{
								"task1": {
									ReqNPUNum: 8,
								},
							},
						},
					},
				},
			},
			expectedResult: &api.ValidateResult{
				Pass:    false,
				Reason:  JobCheckFailedReason,
				Message: "distributed job require npu 16, instead of 8",
			},
		},
		{
			name: "06 Mixed tasks - some pass, some fail - should fail on first failure",
			base910A3: &Base910A3{
				NPUHandler: base.NPUHandler{
					MaxNodeNPUNum: 16,
					SchedulerJobAttr: util.SchedulerJobAttr{
						NPUJob: &util.NPUJob{
							Tasks: map[api.TaskID]util.NPUTask{
								"task1": {
									ReqNPUNum: 16, // This one passes
								},
								"task2": {
									ReqNPUNum: 8, // This one fails
								},
							},
						},
					},
				},
			},
			expectedResult: &api.ValidateResult{
				Pass:    false,
				Reason:  JobCheckFailedReason,
				Message: "distributed job require npu 16, instead of 8",
			},
		},
		{
			name: "07 Empty tasks list - should pass",
			base910A3: &Base910A3{
				NPUHandler: base.NPUHandler{
					MaxNodeNPUNum: 16,
					SchedulerJobAttr: util.SchedulerJobAttr{
						NPUJob: &util.NPUJob{
							Tasks: map[api.TaskID]util.NPUTask{},
						},
					},
				},
			},
			expectedResult: nil, // Should pass, return nil
		},
		{
			name: "08 Task with zero NPU and both special annotations - should pass",
			base910A3: &Base910A3{
				NPUHandler: base.NPUHandler{
					MaxNodeNPUNum: 16,
					SchedulerJobAttr: util.SchedulerJobAttr{
						NPUJob: &util.NPUJob{
							Tasks: map[api.TaskID]util.NPUTask{
								"task1": {
									ReqNPUNum: 0,
									Annotation: map[string]string{
										taskSpec:         schedulerSpec,
										skipAscendPlugin: skipEnabled,
									},
								},
							},
						},
					},
				},
			},
			expectedResult: nil, // Should pass, return nil
		},
		{
			name: "09 Task with zero NPU and wrong annotation values - should fail",
			base910A3: &Base910A3{
				NPUHandler: base.NPUHandler{
					MaxNodeNPUNum: 16,
					SchedulerJobAttr: util.SchedulerJobAttr{
						NPUJob: &util.NPUJob{
							Tasks: map[api.TaskID]util.NPUTask{
								"task1": {
									ReqNPUNum: 0,
									Annotation: map[string]string{
										taskSpec:         "wrong-value",
										skipAscendPlugin: "wrong-value",
									},
								},
							},
						},
					},
				},
			},
			expectedResult: &api.ValidateResult{
				Pass:    false,
				Reason:  JobCheckFailedReason,
				Message: "distributed job require npu 16, instead of 0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.base910A3.CheckReqNPUEqualNodeNPU()

			if tt.expectedResult == nil {
				if result != nil {
					t.Errorf("CheckReqNPUEqualNodeNPU() = %v, expected nil", result)
				}
			} else {
				if result == nil {
					t.Errorf("CheckReqNPUEqualNodeNPU() = nil, expected %v", tt.expectedResult)
				} else {
					if result.Pass != tt.expectedResult.Pass {
						t.Errorf("CheckReqNPUEqualNodeNPU().Pass = %v, expected %v", result.Pass, tt.expectedResult.Pass)
					}
					if result.Reason != tt.expectedResult.Reason {
						t.Errorf("CheckReqNPUEqualNodeNPU().Reason = %v, expected %v", result.Reason, tt.expectedResult.Reason)
					}
					if result.Message != tt.expectedResult.Message {
						t.Errorf("CheckReqNPUEqualNodeNPU().Message = %v, expected %v", result.Message, tt.expectedResult.Message)
					}
				}
			}
		})
	}
}

func TestBase910A3_CheckReqNPUEqualNodeNPU_EdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		base910A3      *Base910A3
		expectedResult *api.ValidateResult
	}{
		{
			name: "01 MaxNodeNPUNum is zero but task matches - should pass",
			base910A3: &Base910A3{
				NPUHandler: base.NPUHandler{
					MaxNodeNPUNum: 0,
					SchedulerJobAttr: util.SchedulerJobAttr{
						NPUJob: &util.NPUJob{
							Tasks: map[api.TaskID]util.NPUTask{
								"task1": {
									ReqNPUNum: 0,
								},
							},
						},
					},
				},
			},
			expectedResult: nil, // Should pass, both are 0
		},
		{
			name: "02 Task with negative NPU number - should fail",
			base910A3: &Base910A3{
				NPUHandler: base.NPUHandler{
					MaxNodeNPUNum: 16,
					SchedulerJobAttr: util.SchedulerJobAttr{
						NPUJob: &util.NPUJob{
							Tasks: map[api.TaskID]util.NPUTask{
								"task1": {
									ReqNPUNum: -1,
								},
							},
						},
					},
				},
			},
			expectedResult: &api.ValidateResult{
				Pass:    false,
				Reason:  JobCheckFailedReason,
				Message: "distributed job require npu 16, instead of -1",
			},
		},
		{
			name: "03 Task with very large NPU number - should fail",
			base910A3: &Base910A3{
				NPUHandler: base.NPUHandler{
					MaxNodeNPUNum: 16,
					SchedulerJobAttr: util.SchedulerJobAttr{
						NPUJob: &util.NPUJob{
							Tasks: map[api.TaskID]util.NPUTask{
								"task1": {
									ReqNPUNum: 1000,
								},
							},
						},
					},
				},
			},
			expectedResult: &api.ValidateResult{
				Pass:    false,
				Reason:  JobCheckFailedReason,
				Message: "distributed job require npu 16, instead of 1000",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.base910A3.CheckReqNPUEqualNodeNPU()

			if tt.expectedResult == nil {
				if result != nil {
					t.Errorf("CheckReqNPUEqualNodeNPU() = %v, expected nil", result)
				}
			} else {
				if result == nil {
					t.Errorf("CheckReqNPUEqualNodeNPU() = nil, expected %v", tt.expectedResult)
				} else {
					if result.Pass != tt.expectedResult.Pass {
						t.Errorf("CheckReqNPUEqualNodeNPU().Pass = %v, expected %v", result.Pass, tt.expectedResult.Pass)
					}
					if result.Reason != tt.expectedResult.Reason {
						t.Errorf("CheckReqNPUEqualNodeNPU().Reason = %v, expected %v", result.Reason, tt.expectedResult.Reason)
					}
					if result.Message != tt.expectedResult.Message {
						t.Errorf("CheckReqNPUEqualNodeNPU().Message = %v, expected %v", result.Message, tt.expectedResult.Message)
					}
				}
			}
		})
	}
}
