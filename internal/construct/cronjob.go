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
	pixivnetv2 "github.com/pixiv/k8s-job-wrapper/api/v2"
	"github.com/pixiv/k8s-job-wrapper/internal/kustomize"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/yaml"
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
func BatchCronJobName(cronJob *pixivnetv2.CronJob) string {
	return cronJob.Name + batchCronJobNameSuffix
}

func BatchCronJobLabelsForList(cronJob *pixivnetv2.CronJob) map[string]string {
	return map[string]string{
		BatchCronJobLabelCreatedBy: BatchCronJobLabelCreatedByValue,
		BatchCronJobLabelName:      cronJob.Name,
	}
}

// Create `cronjobs.v1.batch`.
// Also add the metadata specific to resources generated from a [pixivnetv2.CronJob].
func BatchCronJob(ctx context.Context, cronJob *pixivnetv2.CronJob, podProfile *pixivnetv1.PodProfile, jobProfile *pixivnetv2.JobProfile, cronJobProfile *pixivnetv1.CronJobProfile, patcher kustomize.Patcher, scheme *runtime.Scheme) (*batchv1.CronJob, error) {
	// Create cronjob.spec.jobTemplate
	batchJobSpec, err := BatchJobSpec(ctx, &cronJob.Spec.PodProfile, &cronJob.Spec.JobProfile, podProfile, jobProfile, patcher)
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
		return nil, fmt.Errorf("%w: failed to add owner reference to patched cron job", err)
	}

	batchCronJob.Annotations = map[string]string{}
	batchCronJob.Labels = map[string]string{}

	// TODO: Apply additional labels and annotations
	// Apply additional labels and annotations first.
	// This is to avoid overwriting essential metadata required by the controller.
	// for k, v := range cronJob.Spec.Profile.Metadata.Annotations {
	// 	batchCronJob.Annotations[k] = v
	// }
	// for k, v := range cronJob.Spec.Profile.Metadata.Labels {
	// 	batchCronJob.Labels[k] = v
	// }
	for k, v := range BatchCronJobLabelsForList(cronJob) {
		batchCronJob.Labels[k] = v
	}
	//
	// Set the top-level parameters for the CronJob.
	//
	var (
		spec   = &batchCronJob.Spec
		params = cronJobProfile.Spec.Template.CronJobParams
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
	// spec.JobTemplate.Annotations = cronJob.Spec.Profile.Metadata.Annotations
	// spec.JobTemplate.Labels = cronJob.Spec.Profile.Metadata.Labels
	if len(cronJob.Spec.CronJobProfile.Patches) > 0 {
		spec, err := applyPatchesToCronJob(ctx, &batchCronJob, cronJob.Spec.CronJobProfile.Patches, patcher)
		if err != nil {
			return nil, fmt.Errorf("failed to apply jobPatches: %w", err)
		}
		batchCronJob.Spec = *spec
	}

	return &batchCronJob, nil
}

// Apply the patch to batch Job.
func applyPatchesToCronJob(ctx context.Context, src *batchv1.CronJob, patches []pixivnetv2.JobPatch, patcher kustomize.Patcher) (*batchv1.CronJobSpec, error) {
	cronJobYaml, err := yaml.Marshal(src)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to marshal batch job seed", err)
	}

	patchesYaml, err := yaml.Marshal(patches)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to marshal job patches", err)
	}

	// kubectl kustomize
	patched, err := patcher.Patch(ctx, &kustomize.PatchRequest{
		Group:    batchCronJobGroup,
		Version:  batchCronJobVersion,
		Kind:     batchCronJobKind,
		Name:     src.Name,
		Resource: string(cronJobYaml),
		Patches:  string(patchesYaml),
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to apply job patches", err)
	}

	var patchedCronJob batchv1.CronJob
	if err := yaml.Unmarshal([]byte(patched.Manifest), &patchedCronJob); err != nil {
		return nil, fmt.Errorf("%w: failed to unmarshal pacthed job", err)
	}
	return &patchedCronJob.Spec, nil
}
