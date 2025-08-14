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

package util

import (
	"slices"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type FinishedBatchJobInfo struct {
	FinishedAt metav1.Time
	Finished   bool
}

func GetFinishedBatchJobInfo(job *batchv1.Job) FinishedBatchJobInfo {
	idx := slices.IndexFunc(job.Status.Conditions, func(cond batchv1.JobCondition) bool {
		if cond.Status != corev1.ConditionTrue {
			return false
		}
		switch cond.Type {
		// Job が終了したとき Complete か Failed のどちらかが含まれる
		case batchv1.JobComplete, batchv1.JobFailed:
			return true
		default:
			return false
		}
	})

	if idx < 0 {
		return FinishedBatchJobInfo{
			Finished: false,
		}
	}

	t := job.Status.Conditions[idx].LastTransitionTime
	if t.IsZero() {
		return FinishedBatchJobInfo{
			Finished: true,
		}
	}

	return FinishedBatchJobInfo{
		Finished:   true,
		FinishedAt: t,
	}
}

func IsBatchJobFinished(job *batchv1.Job) bool {
	return GetFinishedBatchJobInfo(job).Finished
}
