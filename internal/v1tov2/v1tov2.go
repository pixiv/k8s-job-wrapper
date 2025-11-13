package v1tov2

import (
	"fmt"

	pixivnetv1 "github.com/pixiv/k8s-job-wrapper/api/v1"
	pixivnetv2 "github.com/pixiv/k8s-job-wrapper/api/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func ToV2(before runtime.Object) ([]runtime.Object, error) {
	switch obj := before.(type) {
	case *pixivnetv1.CronJob:
		return CronJobToV2(obj)
	case *pixivnetv1.Job:
		return JobToV2(obj)
	default:
		return []runtime.Object{before}, nil
	}
}

func CronJobToV2(before *pixivnetv1.CronJob) (new []runtime.Object, err error) {
	copied := before.DeepCopy()
	newPodPatches := make([]pixivnetv2.JobPatch, len(copied.Spec.Profile.Patches))
	for i, patch := range copied.Spec.Profile.Patches {
		newPatch, err := JobPatchToV2(&patch)
		if err != nil {
			return nil, fmt.Errorf("failed to convert JobPatch: %w", err)
		}
		newPodPatches[i] = *newPatch
	}

	new = []runtime.Object{
		&pixivnetv2.CronJob{
			TypeMeta: metav1.TypeMeta{
				APIVersion: pixivnetv2.GroupVersion.String(),
				Kind:       "CronJob",
			},
			ObjectMeta: *copied.ObjectMeta.DeepCopy(),
			Spec: pixivnetv2.CronJobSpec{
				CronJobProfile: pixivnetv2.CronJobProfileRef{
					Ref:     copied.Name,
					Patches: []pixivnetv2.JobPatch{},
				},
				PodProfile: pixivnetv2.PodProfileRef{
					Ref:     copied.Spec.Profile.PodProfileRef,
					Patches: newPodPatches,
				},
				JobProfile: pixivnetv2.JobProfileRef{
					Ref:     copied.Name,
					Patches: []pixivnetv2.JobPatch{},
				},
			},
		},
		&pixivnetv1.CronJobProfile{
			TypeMeta: metav1.TypeMeta{
				APIVersion: pixivnetv1.GroupVersion.String(),
				Kind:       "CronJobProfile",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      copied.GetName(),
				Namespace: copied.GetNamespace(),
			},
			Spec: pixivnetv1.CronJobProfileSpec{
				Template: pixivnetv1.CronJobTemplateSpec{
					CronJobParams: pixivnetv1.CronJobParams{
						Schedule:                   copied.Spec.Schedule,
						TimeZone:                   copied.Spec.TimeZone,
						StartingDeadlineSeconds:    copied.Spec.StartingDeadlineSeconds,
						ConcurrencyPolicy:          copied.Spec.ConcurrencyPolicy,
						Suspend:                    copied.Spec.Suspend,
						SuccessfulJobsHistoryLimit: copied.Spec.SuccessfulJobsHistoryLimit,
						FailedJobsHistoryLimit:     copied.Spec.FailedJobsHistoryLimit,
					},
				},
			},
		},
		&pixivnetv2.JobProfile{
			TypeMeta: metav1.TypeMeta{
				APIVersion: pixivnetv2.GroupVersion.String(),
				Kind:       "JobProfile",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      copied.GetName(),
				Namespace: copied.GetNamespace(),
			},
			Spec: pixivnetv2.JobProfileSpec{
				Template: pixivnetv2.JobTemplateSpec{
					JobParams: pixivnetv2.JobParams{
						Parallelism:             copied.Spec.Profile.Params.Parallelism,
						Completions:             copied.Spec.Profile.Params.Completions,
						ActiveDeadlineSeconds:   copied.Spec.Profile.Params.ActiveDeadlineSeconds,
						PodFailurePolicy:        copied.Spec.Profile.Params.PodFailurePolicy,
						SuccessPolicy:           copied.Spec.Profile.Params.SuccessPolicy,
						BackoffLimit:            copied.Spec.Profile.Params.BackoffLimit,
						BackoffLimitPerIndex:    copied.Spec.Profile.Params.BackoffLimitPerIndex,
						MaxFailedIndexes:        copied.Spec.Profile.Params.MaxFailedIndexes,
						Selector:                copied.Spec.Profile.Params.Selector,
						ManualSelector:          copied.Spec.Profile.Params.ManualSelector,
						TTLSecondsAfterFinished: copied.Spec.Profile.Params.TTLSecondsAfterFinished,
						CompletionMode:          copied.Spec.Profile.Params.CompletionMode,
						Suspend:                 copied.Spec.Profile.Params.Suspend,
						PodReplacementPolicy:    copied.Spec.Profile.Params.PodReplacementPolicy,
						ManagedBy:               copied.Spec.Profile.Params.ManagedBy,
					},
				},
			},
		},
	}
	return
}

func JobToV2(before *pixivnetv1.Job) (new []runtime.Object, err error) {
	copied := before.DeepCopy()
	newPodPatches := make([]pixivnetv2.JobPatch, len(copied.Spec.Profile.Patches))
	for i, patch := range copied.Spec.Profile.Patches {
		newPatch, err := JobPatchToV2(&patch)
		if err != nil {
			return nil, fmt.Errorf("failed to convert JobPatch: %w", err)
		}
		newPodPatches[i] = *newPatch
	}

	new = []runtime.Object{
		&pixivnetv2.Job{
			TypeMeta: metav1.TypeMeta{
				APIVersion: pixivnetv2.GroupVersion.String(),
				Kind:       "Job",
			},
			ObjectMeta: *copied.ObjectMeta.DeepCopy(),
			Spec: pixivnetv2.JobSpec{
				PodProfile: pixivnetv2.PodProfileRef{
					Ref:     copied.Spec.Profile.PodProfileRef,
					Patches: newPodPatches,
				},
				JobProfile: pixivnetv2.JobProfileRef{
					Ref:     copied.Name,
					Patches: []pixivnetv2.JobPatch{},
				},
			},
		},
		&pixivnetv2.JobProfile{
			TypeMeta: metav1.TypeMeta{
				APIVersion: pixivnetv2.GroupVersion.String(),
				Kind:       "JobProfile",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      copied.GetName(),
				Namespace: copied.GetNamespace(),
			},
			Spec: pixivnetv2.JobProfileSpec{
				Template: pixivnetv2.JobTemplateSpec{
					JobParams: pixivnetv2.JobParams{
						Parallelism:             copied.Spec.Profile.Params.Parallelism,
						Completions:             copied.Spec.Profile.Params.Completions,
						ActiveDeadlineSeconds:   copied.Spec.Profile.Params.ActiveDeadlineSeconds,
						PodFailurePolicy:        copied.Spec.Profile.Params.PodFailurePolicy,
						SuccessPolicy:           copied.Spec.Profile.Params.SuccessPolicy,
						BackoffLimit:            copied.Spec.Profile.Params.BackoffLimit,
						BackoffLimitPerIndex:    copied.Spec.Profile.Params.BackoffLimitPerIndex,
						MaxFailedIndexes:        copied.Spec.Profile.Params.MaxFailedIndexes,
						Selector:                copied.Spec.Profile.Params.Selector,
						ManualSelector:          copied.Spec.Profile.Params.ManualSelector,
						TTLSecondsAfterFinished: copied.Spec.Profile.Params.TTLSecondsAfterFinished,
						CompletionMode:          copied.Spec.Profile.Params.CompletionMode,
						Suspend:                 copied.Spec.Profile.Params.Suspend,
						PodReplacementPolicy:    copied.Spec.Profile.Params.PodReplacementPolicy,
						ManagedBy:               copied.Spec.Profile.Params.ManagedBy,
					},
					JobsHistoryLimit: copied.Spec.JobsHistoryLimit,
				},
			},
		},
	}
	return
}

func JobPatchToV2(before *pixivnetv1.JobPatch) (new *pixivnetv2.JobPatch, err error) {
	copied := before.DeepCopy()
	new = &pixivnetv2.JobPatch{
		Operation: copied.Operation,
		Path:      copied.Path,
		Value:     *copied.Value.DeepCopy(),
		From:      copied.From,
	}
	return
}
