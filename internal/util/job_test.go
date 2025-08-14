/*
Copyright 2025 pixiv Inc.

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

package util_test

import (
	"testing"
	"time"

	"github.com/pixiv/k8s-job-wrapper/internal/util"
	"github.com/stretchr/testify/assert"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetFinishedBatchJobInfo(t *testing.T) {
	finishedAt := time.Date(2025, time.May, 1, 12, 1, 2, 0, time.UTC)

	for _, tc := range []struct {
		title string
		job   batchv1.Job
		want  util.FinishedBatchJobInfo
	}{
		{
			title: "running",
			job: batchv1.Job{
				Status: batchv1.JobStatus{
					Conditions: []batchv1.JobCondition{},
				},
			},
			want: util.FinishedBatchJobInfo{},
		},
		{
			title: "finished",
			job: batchv1.Job{
				Status: batchv1.JobStatus{
					Conditions: []batchv1.JobCondition{
						{
							Type:   batchv1.JobComplete,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			want: util.FinishedBatchJobInfo{
				Finished: true,
			},
		},
		{
			title: "finished without last transition time",
			job: batchv1.Job{
				Status: batchv1.JobStatus{
					Conditions: []batchv1.JobCondition{
						{
							Type:               batchv1.JobComplete,
							Status:             corev1.ConditionTrue,
							LastTransitionTime: metav1.NewTime(finishedAt),
						},
					},
				},
			},
			want: util.FinishedBatchJobInfo{
				Finished:   true,
				FinishedAt: metav1.NewTime(finishedAt),
			},
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			got := util.GetFinishedBatchJobInfo(&tc.job)
			assert.Equal(t, tc.want, got)
		})
	}
}
