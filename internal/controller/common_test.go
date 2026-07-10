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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	pixivnetv1 "github.com/pixiv/k8s-job-wrapper/api/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type controllerTestBase struct{}

func (controllerTestBase) reconcile(ctx context.Context, r reconcile.Reconciler, req types.NamespacedName) error {
	_, err := r.Reconcile(ctx, reconcile.Request{
		NamespacedName: req,
	})
	return err
}

func (controllerTestBase) newNSName(namespace, resourceName string) types.NamespacedName {
	return types.NamespacedName{
		Name:      resourceName,
		Namespace: namespace,
	}
}

func (controllerTestBase) newPodProfile(namespace, resourceName string) *pixivnetv1.PodProfile {
	return &pixivnetv1.PodProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourceName,
			Namespace: namespace,
		},
		Spec: pixivnetv1.PodProfileSpec{
			Template: pixivnetv1.PodProfileTemplate{
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

func (controllerTestBase) newCronJob(namespace, resourceName, podProfileRef string) *pixivnetv1.CronJob {
	return &pixivnetv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourceName,
			Namespace: namespace,
		},
		Spec: pixivnetv1.CronJobSpec{
			Profile: pixivnetv1.JobProfileSpec{
				PodProfileRef: podProfileRef,
				Patches: []pixivnetv1.JobPatch{
					{
						Operation: "replace",
						Path:      "/spec/containers/0/image",
						Value: apiextensionsv1.JSON{
							Raw: []byte(`"debian:bookworm-slim"`),
						},
					},
				},
				Params: pixivnetv1.JobParams{
					Suspend: new(true),
				},
			},
			Schedule: "* * * * *",
		},
	}
}

func (controllerTestBase) newJob(namespace, resourceName, podProfileRef string) *pixivnetv1.Job {
	return &pixivnetv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourceName,
			Namespace: namespace,
		},
		Spec: pixivnetv1.JobSpec{
			Profile: pixivnetv1.JobProfileSpec{
				PodProfileRef: podProfileRef,
				Patches: []pixivnetv1.JobPatch{
					{
						Operation: "replace",
						Path:      "/spec/containers/0/image",
						Value: apiextensionsv1.JSON{
							Raw: []byte(`"debian:bookworm-slim"`),
						},
					},
				},
				Params: pixivnetv1.JobParams{
					Suspend: new(true),
				},
			},
		},
	}
}

func (controllerTestBase) newJobWithTTL(namespace, resourceName, podProfileRef string, ttl int) *pixivnetv1.Job {
	return &pixivnetv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourceName,
			Namespace: namespace,
		},
		Spec: pixivnetv1.JobSpec{
			Profile: pixivnetv1.JobProfileSpec{
				PodProfileRef: podProfileRef,
				Patches: []pixivnetv1.JobPatch{
					{
						Operation: "replace",
						Path:      "/spec/containers/0/image",
						Value: apiextensionsv1.JSON{
							Raw: []byte(`"debian:bookworm-slim"`),
						},
					},
				},
				Params: pixivnetv1.JobParams{
					TTLSecondsAfterFinished: new(int32(ttl)),
				},
			},
		},
	}
}

func (controllerTestBase) newJobWithHistoryLimit(namespace, resourceName, podProfileRef string, limit int) *pixivnetv1.Job {
	return &pixivnetv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourceName,
			Namespace: namespace,
		},
		Spec: pixivnetv1.JobSpec{
			JobsHistoryLimit: new(limit),
			Profile: pixivnetv1.JobProfileSpec{
				PodProfileRef: podProfileRef,
				Patches: []pixivnetv1.JobPatch{
					{
						Operation: "replace",
						Path:      "/spec/containers/0/image",
						Value: apiextensionsv1.JSON{
							Raw: []byte(`"debian:bookworm-slim"`),
						},
					},
				},
			},
		},
	}
}

func (controllerTestBase) newJobWithComplexPatch(namespace, resourceName, podProfileRef string) *pixivnetv1.Job {
	return &pixivnetv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourceName,
			Namespace: namespace,
		},
		Spec: pixivnetv1.JobSpec{
			Profile: pixivnetv1.JobProfileSpec{
				PodProfileRef: podProfileRef,
				Patches: []pixivnetv1.JobPatch{
					{
						Operation: "add",
						Path:      "/spec/containers/-",
						Value: apiextensionsv1.JSON{
							Raw: []byte(`{
  "name": "added",
  "image": "debian:bookworm",
  "command": ["sleep", "1"]
}`),
						},
					},
				},
			},
		},
	}
}

func (controllerTestBase) newJobWithMeta(namespace, resourceName, podProfileRef string) *pixivnetv1.Job {
	return &pixivnetv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourceName,
			Namespace: namespace,
		},
		Spec: pixivnetv1.JobSpec{
			Profile: pixivnetv1.JobProfileSpec{
				PodProfileRef: podProfileRef,
				Patches: []pixivnetv1.JobPatch{
					{
						Operation: "replace",
						Path:      "/spec/containers/0/image",
						Value: apiextensionsv1.JSON{
							Raw: []byte(`"debian:bookworm-slim"`),
						},
					},
				},
				Params: pixivnetv1.JobParams{
					Suspend: new(true),
				},
				Metadata: pixivnetv1.JobMetadata{
					Annotations: map[string]string{
						"case": "withMeta",
					},
					Labels: map[string]string{
						"additional": "label",
					},
				},
			},
		},
	}
}

func (controllerTestBase) assertStatus(got []metav1.Condition, key string, status metav1.ConditionStatus, reason string, message ...string) {
	GinkgoHelper()
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

func (controllerTestBase) newBatchJobCompleteStatus(startTime, completionTime metav1.Time) batchv1.JobStatus {
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

func (controllerTestBase) newBatchJobFailedStatus(startTime metav1.Time) batchv1.JobStatus {
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

//
// Common functions for testing.
//

const testNamespace = "default"

// Create a key for k8sClient.Get().
func newKey(namespace, resourceName string) types.NamespacedName {
	return types.NamespacedName{
		Name:      resourceName,
		Namespace: namespace,
	}
}

func newPodProfile(namespace, resourceName string) *pixivnetv1.PodProfile {
	return &pixivnetv1.PodProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourceName,
			Namespace: namespace,
		},
		Spec: pixivnetv1.PodProfileSpec{
			Template: pixivnetv1.PodProfileTemplate{
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
