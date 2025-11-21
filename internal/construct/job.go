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
	"strconv"

	pixivnetv1 "github.com/pixiv/k8s-job-wrapper/api/v1"
	pixivnetv2 "github.com/pixiv/k8s-job-wrapper/api/v2"
	"github.com/pixiv/k8s-job-wrapper/internal/kustomize"
	"github.com/pixiv/k8s-job-wrapper/internal/util"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubernetes/pkg/controller"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/yaml"
)

const (
	batchJobKind    = "Job"
	batchJobGroup   = "batch"
	batchJobVersion = "v1"

	// A label that holds the hash of jobs.v1.batch.spec.
	BatchJobLabelSpecHashKey = "jobs.pixiv.net/job-spec-hash"
	// A label to track the generation source.
	BatchJobLabelCreatedBy      = "app.kubernetes.io/created-by"
	BatchJobLabelCreatedByValue = "pixiv-job-controller"
	// A label that holds the name of the source [pixivnetv1.Job].
	BatchJobLabelName = "jobs.pixiv.net/name"
	// An annotation that holds TTLSecondsAfterFinished.
	BatchJobAnnotationTTLSecondsAfterFinished = "jobs.pixiv.net/ttl-seconds-after-finished"
)

func BatchJobTTLSecondsAfterFinishedFromAnnotations(batchJob *batchv1.Job) (int32, bool) {
	x, ok := batchJob.GetAnnotations()[BatchJobAnnotationTTLSecondsAfterFinished]
	if !ok {
		return 0, false
	}
	ttl, err := strconv.ParseInt(x, 10, 32)
	if err != nil {
		return 0, false
	}
	return int32(ttl), true
}

// A label used to retrieve a list of the generated batch Jobs.
func BatchJobLabelsForList(job *pixivnetv2.Job) map[string]string {
	return map[string]string{
		BatchJobLabelCreatedBy: BatchJobLabelCreatedByValue,
		BatchJobLabelName:      job.Name,
	}
}

// Create `jobs.v1.batch`.
// Also add the metadata specific to resources generated from a [pixivnetv1.CronJob].
func BatchJob(ctx context.Context, job *pixivnetv2.Job, podProfile *pixivnetv1.PodProfile, jobProfile *pixivnetv2.JobProfile, patcher kustomize.Patcher, scheme *runtime.Scheme) (*batchv1.Job, error) {
	nextJobSpec, err := BatchJobSpec(ctx, &job.Spec.PodProfile, &job.Spec.JobProfile, podProfile, jobProfile, patcher)
	if err != nil {
		return nil, err
	}

	var batchJob batchv1.Job
	batchJob.Spec = *nextJobSpec

	//
	// Metadata
	//
	batchJob.TypeMeta = metav1.TypeMeta{
		APIVersion: batchJobGroup + "/" + batchJobVersion,
		Kind:       batchJobKind,
	}
	batchJob.Namespace = job.Namespace
	// Set ownerRefernce.
	if err := ctrl.SetControllerReference(job, &batchJob, scheme); err != nil {
		return nil, fmt.Errorf("%w: failed to add owner reference to patched job", err)
	}
	batchJob.Annotations = map[string]string{}
	batchJob.Labels = map[string]string{}
	// TODO: Apply additional labels and annotations
	// Apply additional labels and annotations first.
	// This is to avoid overwriting essential metadata required by the controller.
	// for k, v := range job.Spec.Profile.Metadata.Annotations {
	// 	batchJob.Annotations[k] = v
	// }
	// for k, v := range job.Spec.Profile.Metadata.Labels {
	// 	batchJob.Labels[k] = v
	// }
	// Save TTL to the annotation.
	if x := jobProfile.Spec.Template.JobParams.TTLSecondsAfterFinished; x != nil {
		batchJob.Annotations[BatchJobAnnotationTTLSecondsAfterFinished] = fmt.Sprintf("%d", *x)
	}
	for k, v := range BatchJobLabelsForList(job) {
		batchJob.Labels[k] = v
	}
	// Create the name of the job from the hash of spec.
	podTemplateHash := controller.ComputeHash(&batchJob.Spec.Template, job.Status.CollisionCount)
	batchJob.Labels["pod-template-hash"] = podTemplateHash
	batchJob.Name = job.Name + "-" + podTemplateHash
	// Store the hash of the spec to compare for equality.
	batchJob.Labels[BatchJobLabelSpecHashKey] = util.ComputeHash(&batchJob.Spec)

	return &batchJob, nil
}

// Create `jobs.v1.batch.spec`.
func BatchJobSpec(ctx context.Context, podProfileRef *pixivnetv2.PodProfileRef, jobProfileRef *pixivnetv2.JobProfileRef, podProfile *pixivnetv1.PodProfile, jobProfile *pixivnetv2.JobProfile, patcher kustomize.Patcher) (*batchv1.JobSpec, error) {
	var batchJob batchv1.Job

	//
	// Metadata
	//
	batchJob.TypeMeta = metav1.TypeMeta{
		APIVersion: batchJobGroup + "/" + batchJobVersion,
		Kind:       batchJobKind,
	}
	// A placeholder name.
	// This must be set before calling PatchRunner in order to apply the patch.
	batchJob.Name = "next-job"

	//
	// Set the top-level parameters for the Job
	//
	var (
		spec   = &batchJob.Spec
		params = jobProfile.Spec.Template.JobParams
	)
	spec.Parallelism = params.Parallelism
	spec.Completions = params.Completions
	spec.ActiveDeadlineSeconds = params.ActiveDeadlineSeconds
	spec.PodFailurePolicy = params.PodFailurePolicy
	spec.SuccessPolicy = params.SuccessPolicy
	spec.BackoffLimit = params.BackoffLimit
	spec.BackoffLimitPerIndex = params.BackoffLimitPerIndex
	spec.MaxFailedIndexes = params.MaxFailedIndexes
	spec.Selector = params.Selector
	spec.ManualSelector = params.ManualSelector
	// CRD Job manages TTLSecondsAfterFinished.
	// spec.TTLSecondsAfterFinished = params.TTLSecondsAfterFinished
	spec.CompletionMode = params.CompletionMode
	spec.Suspend = params.Suspend
	spec.PodReplacementPolicy = params.PodReplacementPolicy
	spec.ManagedBy = params.ManagedBy
	spec.Template = podProfile.Spec.Template // Set the podprofile directly as the target for patching.

	if len(podProfileRef.Patches) > 0 {
		podPatches := podProfileRef.DeepCopy().Patches
		for i := range podPatches {
			// The user writes the patch assuming the root path is podprofile.spec.template,
			// which is equivalent to batch/v1.job.spec.template.
			// We rewrite the path to adjust for this and make it work as expected.
			podPatches[i].Path = "/spec/template" + podPatches[i].Path
		}
		spec, err := applyPatchesToJob(ctx, &batchJob, podPatches, patcher)
		if err != nil {
			return nil, fmt.Errorf("failed to apply podPatches: %w", err)
		}
		batchJob.Spec = *spec
	}
	if len(jobProfileRef.Patches) > 0 {
		spec, err := applyPatchesToJob(ctx, &batchJob, jobProfileRef.Patches, patcher)
		if err != nil {
			return nil, fmt.Errorf("failed to apply jobPatches: %w", err)
		}
		batchJob.Spec = *spec
	}

	return &batchJob.Spec, nil
}

// Apply the patch to batch Job.
func applyPatchesToJob(ctx context.Context, src *batchv1.Job, patches []pixivnetv2.JobPatch, patcher kustomize.Patcher) (*batchv1.JobSpec, error) {
	jobYaml, err := yaml.Marshal(src)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to marshal batch job seed", err)
	}

	patchesYaml, err := yaml.Marshal(patches)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to marshal job patches", err)
	}

	// kubectl kustomize
	patched, err := patcher.Patch(ctx, &kustomize.PatchRequest{
		Group:    batchJobGroup,
		Version:  batchJobVersion,
		Kind:     batchJobKind,
		Name:     src.Name,
		Resource: string(jobYaml),
		Patches:  string(patchesYaml),
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to apply job patches", err)
	}

	var patchedJob batchv1.Job
	if err := yaml.Unmarshal([]byte(patched.Manifest), &patchedJob); err != nil {
		return nil, fmt.Errorf("%w: failed to unmarshal pacthed job", err)
	}
	return &patchedJob.Spec, nil
}
