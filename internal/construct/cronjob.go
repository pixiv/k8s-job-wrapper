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

package construct

import (
	"context"
	"fmt"

	pixivnetv1 "github.com/pixiv/k8s-job-wrapper/api/v1"
	"github.com/pixiv/k8s-job-wrapper/internal/kustomize"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	batchCronJobKind    = "CronJob"
	batchCronJobGroup   = "batch"
	batchCronJobVersion = "v1"

	// BatchCronJobLabelSpecHashKey    = "cronjobs.pixiv.net/cronjob-spec-hash"
	// A label to track the generation source.
	BatchCronJobLabelCreatedBy      = "app.kubernetes.io/created-by"
	BatchCronJobLabelCreatedByValue = "pixiv-job-controller"
	// A label that holds the name of the source [pixivnetv1.CronJob].
	BatchCronJobLabelName = "cronjobs.pixiv.net/name"

	// The suffix for the generated batch CronJob.
	// Using just "cronjob" would likely result in many duplicate names.
	// This should be 11 characters or less at most (since Job uses an 11-character suffix).
	batchCronJobNameSuffix = "-pxvcjob"
)

// Create the name of batch CronJob from CronJob.
func BatchCronJobName(cronJob *pixivnetv1.CronJob) string {
	return cronJob.Name + batchCronJobNameSuffix
}

func BatchCronJobLabelsForList(cronJob *pixivnetv1.CronJob) map[string]string {
	return map[string]string{
		BatchCronJobLabelCreatedBy: BatchCronJobLabelCreatedByValue,
		BatchCronJobLabelName:      cronJob.Name,
	}
}

// Create `cronjobs.v1.batch`.
// Also add the metadata specific to resources generated from a [pixivnetv1.CronJob].
func BatchCronJob(ctx context.Context, cronJob *pixivnetv1.CronJob, podProfile *pixivnetv1.PodProfile, patcher kustomize.Patcher, scheme *runtime.Scheme) (*batchv1.CronJob, error) {
	// Create cronjob.spec.jobTemplate
	batchJobSpec, err := BatchJobSpec(ctx, &cronJob.Spec.Profile, podProfile, patcher)
	if err != nil {
		return nil, err
	}

	var batchCronJob batchv1.CronJob

	//
	// Metadata
	//
	batchCronJob.TypeMeta = metav1.TypeMeta{
		APIVersion: batchCronJobGroup + "/" + batchCronJobVersion,
		Kind:       batchCronJobKind,
	}
	batchCronJob.Namespace = cronJob.Namespace
	batchCronJob.Name = BatchCronJobName(cronJob)
	// Set ownerRefernce.
	if err := ctrl.SetControllerReference(cronJob, &batchCronJob, scheme); err != nil {
		return nil, fmt.Errorf("failed to add owner reference to patched cron job: %w", err)
	}

	batchCronJob.Annotations = map[string]string{}
	batchCronJob.Labels = map[string]string{}

	// Apply additional labels and annotations first.
	// This is to avoid overwriting essential metadata required by the controller.
	for k, v := range cronJob.Spec.Profile.Metadata.Annotations {
		batchCronJob.Annotations[k] = v
	}
	for k, v := range cronJob.Spec.Profile.Metadata.Labels {
		batchCronJob.Labels[k] = v
	}
	for k, v := range BatchCronJobLabelsForList(cronJob) {
		batchCronJob.Labels[k] = v
	}
	//
	// Set the top-level parameters for the CronJob.
	//
	var (
		spec   = &batchCronJob.Spec
		params = cronJob.Spec
	)
	spec.Schedule = params.Schedule
	spec.TimeZone = params.TimeZone
	spec.StartingDeadlineSeconds = params.StartingDeadlineSeconds
	spec.ConcurrencyPolicy = params.ConcurrencyPolicy
	spec.Suspend = params.Suspend
	spec.SuccessfulJobsHistoryLimit = params.SuccessfulJobsHistoryLimit
	spec.FailedJobsHistoryLimit = params.FailedJobsHistoryLimit
	// Set the batch JobSpec and the metadata.
	spec.JobTemplate = batchv1.JobTemplateSpec{
		Spec: *batchJobSpec,
	}
	spec.JobTemplate.Annotations = cronJob.Spec.Profile.Metadata.Annotations
	spec.JobTemplate.Labels = cronJob.Spec.Profile.Metadata.Labels

	return &batchCronJob, nil
}
