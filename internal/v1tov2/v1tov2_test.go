// Copyright 2[0-9]{3} pixiv Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1tov2

import (
	"testing"

	pixivnetv1 "github.com/pixiv/k8s-job-wrapper/api/v1"
	pixivnetv2 "github.com/pixiv/k8s-job-wrapper/api/v2"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/batch/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
)

func complexCronJobV1(mutator ...func(*pixivnetv1.CronJob)) *pixivnetv1.CronJob {
	c := &pixivnetv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sample-cronjob",
			Namespace: "sample",
		},
		Spec: pixivnetv1.CronJobSpec{
			Profile: pixivnetv1.JobProfileSpec{
				PodProfileRef: "sample-podprofile",
				Patches: []pixivnetv1.JobPatch{
					{Operation: "add", Path: "/hoge", Value: apiextensionsv1.JSON{Raw: []byte("{\"fuga\":\"foo\"}")}},
					{Operation: "replace", Path: "/hogehoge", Value: apiextensionsv1.JSON{Raw: []byte("{\"fuga\":\"foooo\"}")}},
				},
				Params: pixivnetv1.JobParams{
					Parallelism:           ptr.To(int32(1)),
					Completions:           ptr.To(int32(2)),
					ActiveDeadlineSeconds: ptr.To(int64(3)),
					PodFailurePolicy: &v1.PodFailurePolicy{
						Rules: []v1.PodFailurePolicyRule{{Action: v1.PodFailurePolicyActionFailJob}},
					},
					SuccessPolicy: &v1.SuccessPolicy{
						Rules: []v1.SuccessPolicyRule{{SucceededCount: ptr.To(int32(4))}},
					},
					BackoffLimit:            ptr.To(int32(5)),
					BackoffLimitPerIndex:    ptr.To(int32(6)),
					MaxFailedIndexes:        ptr.To(int32(7)),
					Selector:                &metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}},
					ManualSelector:          ptr.To(true),
					TTLSecondsAfterFinished: ptr.To(int32(8)),
					CompletionMode:          ptr.To(v1.IndexedCompletion),
					Suspend:                 ptr.To(true),
					PodReplacementPolicy:    ptr.To(v1.TerminatingOrFailed),
					ManagedBy:               ptr.To("sample-managedby"),
				},
			},
			Metadata: pixivnetv1.CronJobMetadata{
				Labels:      map[string]string{"label": "fuga"},
				Annotations: map[string]string{"annotation": "foo"},
			},
			Schedule:                   "* * * * *",
			TimeZone:                   ptr.To("Asia/Tokyo"),
			StartingDeadlineSeconds:    ptr.To(int64(1)),
			ConcurrencyPolicy:          v1.AllowConcurrent,
			Suspend:                    ptr.To(true),
			SuccessfulJobsHistoryLimit: ptr.To(int32(2)),
			FailedJobsHistoryLimit:     ptr.To(int32(3)),
		},
	}
	c.GetObjectKind().SetGroupVersionKind(pixivnetv1.GroupVersion.WithKind("CronJob"))
	for _, m := range mutator {
		m(c)
	}
	return c
}

func complexJobV1(mutator ...func(*pixivnetv1.Job)) *pixivnetv1.Job {
	j := &pixivnetv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sample-job",
			Namespace: "sample",
		},
		Spec: pixivnetv1.JobSpec{
			Profile: pixivnetv1.JobProfileSpec{
				PodProfileRef: "sample-podprofile",
				Patches: []pixivnetv1.JobPatch{
					{Operation: "add", Path: "/hoge", Value: apiextensionsv1.JSON{Raw: []byte("{\"fuga\":\"foo\"}")}},
					{Operation: "replace", Path: "/hogehoge", Value: apiextensionsv1.JSON{Raw: []byte("{\"fuga\":\"foooo\"}")}},
				},
				Params: pixivnetv1.JobParams{
					Parallelism:           ptr.To(int32(1)),
					Completions:           ptr.To(int32(2)),
					ActiveDeadlineSeconds: ptr.To(int64(3)),
					PodFailurePolicy: &v1.PodFailurePolicy{
						Rules: []v1.PodFailurePolicyRule{{Action: v1.PodFailurePolicyActionFailJob}},
					},
					SuccessPolicy: &v1.SuccessPolicy{
						Rules: []v1.SuccessPolicyRule{{SucceededCount: ptr.To(int32(4))}},
					},
					BackoffLimit:            ptr.To(int32(5)),
					BackoffLimitPerIndex:    ptr.To(int32(6)),
					MaxFailedIndexes:        ptr.To(int32(7)),
					Selector:                &metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}},
					ManualSelector:          ptr.To(true),
					TTLSecondsAfterFinished: ptr.To(int32(8)),
					CompletionMode:          ptr.To(v1.IndexedCompletion),
					Suspend:                 ptr.To(true),
					PodReplacementPolicy:    ptr.To(v1.TerminatingOrFailed),
					ManagedBy:               ptr.To("sample-managedby"),
				},
			},
			JobsHistoryLimit: ptr.To(3),
		},
	}
	j.GetObjectKind().SetGroupVersionKind(pixivnetv1.GroupVersion.WithKind("Job"))
	for _, m := range mutator {
		m(j)
	}
	return j
}

func TestToV2_CronJob(t *testing.T) {
	complexBefore := complexCronJobV1()
	complexAfter := []runtime.Object{
		&pixivnetv2.CronJob{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "pixiv.net/v2",
				Kind:       "CronJob",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample-cronjob",
				Namespace: "sample",
			},
			Spec: pixivnetv2.CronJobSpec{
				CronJobProfile: pixivnetv2.CronJobProfileRef{
					Ref:     "sample-cronjob",
					Patches: []pixivnetv2.JobPatch{},
				},
				PodProfile: pixivnetv2.PodProfileRef{
					Ref: "sample-podprofile",
					Patches: []pixivnetv2.JobPatch{
						{Operation: "add", Path: "/hoge", Value: apiextensionsv1.JSON{Raw: []byte("{\"fuga\":\"foo\"}")}},
						{Operation: "replace", Path: "/hogehoge", Value: apiextensionsv1.JSON{Raw: []byte("{\"fuga\":\"foooo\"}")}},
					},
				},
				JobProfile: pixivnetv2.JobProfileRef{
					Ref:     "sample-cronjob",
					Patches: []pixivnetv2.JobPatch{},
				},
			},
		},
		&pixivnetv1.CronJobProfile{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "pixiv.net/v1",
				Kind:       "CronJobProfile",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample-cronjob",
				Namespace: "sample",
			},
			Spec: pixivnetv1.CronJobProfileSpec{
				Template: pixivnetv1.CronJobTemplateSpec{
					CronJobParams: pixivnetv1.CronJobParams{
						Schedule:                   "* * * * *",
						TimeZone:                   ptr.To("Asia/Tokyo"),
						StartingDeadlineSeconds:    ptr.To(int64(1)),
						ConcurrencyPolicy:          v1.AllowConcurrent,
						Suspend:                    ptr.To(true),
						SuccessfulJobsHistoryLimit: ptr.To(int32(2)),
						FailedJobsHistoryLimit:     ptr.To(int32(3)),
					},
				},
			},
		},
		&pixivnetv2.JobProfile{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "pixiv.net/v2",
				Kind:       "JobProfile",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample-cronjob",
				Namespace: "sample",
			},
			Spec: pixivnetv2.JobProfileSpec{
				Template: pixivnetv2.JobTemplateSpec{
					JobParams: pixivnetv2.JobParams{
						Parallelism:           ptr.To(int32(1)),
						Completions:           ptr.To(int32(2)),
						ActiveDeadlineSeconds: ptr.To(int64(3)),
						PodFailurePolicy: &v1.PodFailurePolicy{
							Rules: []v1.PodFailurePolicyRule{{Action: v1.PodFailurePolicyActionFailJob}},
						},
						SuccessPolicy: &v1.SuccessPolicy{
							Rules: []v1.SuccessPolicyRule{{SucceededCount: ptr.To(int32(4))}},
						},
						BackoffLimit:            ptr.To(int32(5)),
						BackoffLimitPerIndex:    ptr.To(int32(6)),
						MaxFailedIndexes:        ptr.To(int32(7)),
						Selector:                &metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}},
						ManualSelector:          ptr.To(true),
						TTLSecondsAfterFinished: ptr.To(int32(8)),
						CompletionMode:          ptr.To(v1.IndexedCompletion),
						Suspend:                 ptr.To(true),
						PodReplacementPolicy:    ptr.To(v1.TerminatingOrFailed),
						ManagedBy:               ptr.To("sample-managedby"),
					},
				},
			},
		},
	}
	suites := []struct {
		Name   string
		Before *pixivnetv1.CronJob
		After  []runtime.Object
	}{
		{"Complex", complexBefore, complexAfter},
	}
	for _, suit := range suites {
		t.Run(suit.Name, func(t *testing.T) {
			actual, changed, err := ToV2(suit.Before)
			if err != nil {
				t.Errorf("failed to convert CronJob with V1tov2: %v", err)
			}
			assert.Equal(t, suit.After, actual)
			assert.Equal(t, true, changed)
		})
	}
}

func TestToV2_Job(t *testing.T) {
	complexBefore := complexJobV1()
	complexAfter := []runtime.Object{
		&pixivnetv2.Job{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "pixiv.net/v2",
				Kind:       "Job",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample-job",
				Namespace: "sample",
			},
			Spec: pixivnetv2.JobSpec{
				PodProfile: pixivnetv2.PodProfileRef{
					Ref: "sample-podprofile",
					Patches: []pixivnetv2.JobPatch{
						{Operation: "add", Path: "/hoge", Value: apiextensionsv1.JSON{Raw: []byte("{\"fuga\":\"foo\"}")}},
						{Operation: "replace", Path: "/hogehoge", Value: apiextensionsv1.JSON{Raw: []byte("{\"fuga\":\"foooo\"}")}},
					},
				},
				JobProfile: pixivnetv2.JobProfileRef{
					Ref:     "sample-job",
					Patches: []pixivnetv2.JobPatch{},
				},
			},
		},
		&pixivnetv2.JobProfile{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "pixiv.net/v2",
				Kind:       "JobProfile",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sample-job",
				Namespace: "sample",
			},
			Spec: pixivnetv2.JobProfileSpec{
				Template: pixivnetv2.JobTemplateSpec{
					JobParams: pixivnetv2.JobParams{
						Parallelism:           ptr.To(int32(1)),
						Completions:           ptr.To(int32(2)),
						ActiveDeadlineSeconds: ptr.To(int64(3)),
						PodFailurePolicy: &v1.PodFailurePolicy{
							Rules: []v1.PodFailurePolicyRule{{Action: v1.PodFailurePolicyActionFailJob}},
						},
						SuccessPolicy: &v1.SuccessPolicy{
							Rules: []v1.SuccessPolicyRule{{SucceededCount: ptr.To(int32(4))}},
						},
						BackoffLimit:            ptr.To(int32(5)),
						BackoffLimitPerIndex:    ptr.To(int32(6)),
						MaxFailedIndexes:        ptr.To(int32(7)),
						Selector:                &metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}},
						ManualSelector:          ptr.To(true),
						TTLSecondsAfterFinished: ptr.To(int32(8)),
						CompletionMode:          ptr.To(v1.IndexedCompletion),
						Suspend:                 ptr.To(true),
						PodReplacementPolicy:    ptr.To(v1.TerminatingOrFailed),
						ManagedBy:               ptr.To("sample-managedby"),
					},
					JobsHistoryLimit: ptr.To(3),
				},
			},
		},
	}
	suites := []struct {
		Name   string
		Before *pixivnetv1.Job
		After  []runtime.Object
	}{
		{"Complex", complexBefore, complexAfter},
	}
	for _, suit := range suites {
		t.Run(suit.Name, func(t *testing.T) {
			actual, changed, err := ToV2(suit.Before)
			if err != nil {
				t.Errorf("failed to convert Job with V1tov2: %v", err)
			}
			assert.Equal(t, suit.After, actual)
			assert.Equal(t, true, changed)
		})
	}
}

func TestToV2_JobPatch(t *testing.T) {
	suites := []struct {
		Name   string
		Before *pixivnetv1.JobPatch
		After  *pixivnetv2.JobPatch
	}{
		{
			"Complex Add",
			&pixivnetv1.JobPatch{Operation: "add", Path: "/hoge", Value: apiextensionsv1.JSON{Raw: []byte("{\"fuga\":\"foo\"}")}, From: "yeah"},
			&pixivnetv2.JobPatch{Operation: "add", Path: "/hoge", Value: apiextensionsv1.JSON{Raw: []byte("{\"fuga\":\"foo\"}")}, From: "yeah"},
		},
		{
			"Complex Replace",
			&pixivnetv1.JobPatch{Operation: "replace", Path: "/hogehoge", Value: apiextensionsv1.JSON{Raw: []byte("{\"fuga\":\"foooo\"}")}},
			&pixivnetv2.JobPatch{Operation: "replace", Path: "/hogehoge", Value: apiextensionsv1.JSON{Raw: []byte("{\"fuga\":\"foooo\"}")}},
		},
	}
	for _, suit := range suites {
		t.Run(suit.Name, func(t *testing.T) {
			actual, err := JobPatchToV2(suit.Before)
			if err != nil {
				t.Errorf("failed to convert JobPatch: %v", err)
			}
			assert.Equal(t, suit.After, actual)
		})
	}
}
