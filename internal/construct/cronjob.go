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
	// 何が起因で生成されたかを保持するラベル.
	BatchCronJobLabelCreatedBy      = "app.kubernetes.io/created-by"
	BatchCronJobLabelCreatedByValue = "pixiv-job-controller"
	// 生成元の [pixivnetv1.CronJob] の名前を保持するラベル.
	BatchCronJobLabelName = "cronjobs.pixiv.net/name"

	// 生成する batch cronjob のサフィックス
	// cronjob だと重複するケースが多そう
	// 長くとも11文字以内にしたい(Job は11文字のサフィックスを使う)
	batchCronJobNameSuffix = "-pxvcjob"
)

// cronjob から batch cronjob の名前を作る.
func BatchCronJobName(cronJob *pixivnetv1.CronJob) string {
	return cronJob.Name + batchCronJobNameSuffix
}

func BatchCronJobLabelsForList(cronJob *pixivnetv1.CronJob) map[string]string {
	return map[string]string{
		BatchCronJobLabelCreatedBy: BatchCronJobLabelCreatedByValue,
		BatchCronJobLabelName:      cronJob.Name,
	}
}

// `cronjobs.v1.batch` を生成する.
// [pixivnetv1.CronJob] から生成された時の特有のメタデータも付与する.
func BatchCronJob(ctx context.Context, cronJob *pixivnetv1.CronJob, podProfile *pixivnetv1.PodProfile, patcher kustomize.Patcher, scheme *runtime.Scheme) (*batchv1.CronJob, error) {
	// cronjob.spec.jobTemplate を生成する
	batchJobSpec, err := BatchJobSpec(ctx, &cronJob.Spec.Profile, podProfile, patcher)
	if err != nil {
		return nil, err
	}

	var batchCronJob batchv1.CronJob

	//
	// metadata
	//
	batchCronJob.TypeMeta = metav1.TypeMeta{
		APIVersion: batchCronJobGroup + "/" + batchCronJobVersion,
		Kind:       batchCronJobKind,
	}
	batchCronJob.Namespace = cronJob.Namespace
	batchCronJob.Name = BatchCronJobName(cronJob)
	// ownerRefernce をつける
	if err := ctrl.SetControllerReference(cronJob, &batchCronJob, scheme); err != nil {
		return nil, fmt.Errorf("failed to add owner reference to patched cron job: %w", err)
	}

	batchCronJob.Annotations = map[string]string{}
	batchCronJob.Labels = map[string]string{}
	// 追加のラベルとアノテーションを先に付与する
	// controller のために必須なメタデータの上書きを回避する
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
	// CronJob トップレベルのパラメータを設定する
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
	// 生成した batch JobSpec とメタデータを設定する
	spec.JobTemplate = batchv1.JobTemplateSpec{
		Spec: *batchJobSpec,
	}
	spec.JobTemplate.Annotations = cronJob.Spec.Profile.Metadata.Annotations
	spec.JobTemplate.Labels = cronJob.Spec.Profile.Metadata.Labels

	return &batchCronJob, nil
}
