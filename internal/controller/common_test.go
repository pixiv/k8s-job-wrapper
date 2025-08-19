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

package controller

import (
	. "github.com/onsi/gomega"
	pixivnetv1 "github.com/pixiv/k8s-job-wrapper/api/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

//
// Common functions for testing.
//

const testNamespace = "default"

// Create a key for k8sClient.Get().
func newKey(resourceName string) types.NamespacedName {
	return types.NamespacedName{
		Name:      resourceName,
		Namespace: testNamespace,
	}
}

func newPodProfile(resourceName string) *pixivnetv1.PodProfile {
	return &pixivnetv1.PodProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourceName,
			Namespace: testNamespace,
		},
		Spec: pixivnetv1.PodProfileSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "pi",
							Image:   "perl:5.34.0",
							Command: []string{"perl", "-Mbignum=bpi", "-wle", "print bpi(2000)"},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}
}

func assertStatus(got []metav1.Condition, key string, status metav1.ConditionStatus, reason string, message ...string) {
	var msg string
	if len(message) > 0 {
		msg = message[0]
	}

	Expect(got).Should(HaveLen(2))
	statusMap := map[string]metav1.Condition{}
	for _, c := range got {
		statusMap[c.Type] = c
	}
	v, ok := statusMap[key]
	if Expect(ok).To(BeTrue()) {
		Expect(v.Status).To(Equal(status))
		Expect(v.Reason).To(Equal(reason))
		Expect(v.Message).To(Equal(msg))
	}
}

func getPodProfile(resourceName string) *pixivnetv1.PodProfile {
	podProfile := &pixivnetv1.PodProfile{}
	Eventually(func() error {
		return k8sClient.Get(ctx, newKey(resourceName), podProfile)
	}).Should(Succeed())
	return podProfile
}

func newBatchJobCompleteStatus(startTime, completionTime metav1.Time) batchv1.JobStatus {
	return batchv1.JobStatus{
		StartTime:      &startTime,
		CompletionTime: &completionTime,
		Conditions: []batchv1.JobCondition{
			{
				Type:               batchv1.JobComplete,
				Status:             corev1.ConditionTrue,
				Reason:             "OK",
				LastTransitionTime: completionTime,
			},
			{
				Type:               batchv1.JobSuccessCriteriaMet,
				Status:             corev1.ConditionTrue,
				Reason:             "OK",
				LastTransitionTime: completionTime,
			},
		},
	}
}

func newBatchJobFailedStatus(startTime metav1.Time) batchv1.JobStatus {
	return batchv1.JobStatus{
		StartTime: &startTime,
		Conditions: []batchv1.JobCondition{
			{
				Type:               batchv1.JobFailed,
				Status:             corev1.ConditionTrue,
				Reason:             "JobFailed",
				LastTransitionTime: startTime,
			},
			{
				Type:               batchv1.JobFailureTarget,
				Status:             corev1.ConditionTrue,
				Reason:             "Failure",
				LastTransitionTime: startTime,
			},
		},
	}
}
