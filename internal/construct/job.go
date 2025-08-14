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

	// `jobs.v1.batch.spec` のハッシュを保持するラベル.
	BatchJobLabelSpecHashKey = "jobs.pixiv.net/job-spec-hash"
	// 何が起因で生成されたかを保持するラベル.
	BatchJobLabelCreatedBy      = "app.kubernetes.io/created-by"
	BatchJobLabelCreatedByValue = "pixiv-job-controller"
	// 生成元の [pixivnetv1.Job] の名前を保持するラベル.
	BatchJobLabelName = "jobs.pixiv.net/name"
	// TTLSecondsAfterFinished を保持するアノテーション
	BatchJobAnnotationTTLSecondsAfterFinished = "jobs.pixiv.net/ttl-seconds-after-finished"
)

func BatchJobTTLSecondsAfterFinishedFromAnnotations(batchJob *batchv1.Job) (int32, bool) {
	x, ok := batchJob.GetAnnotations()[BatchJobAnnotationTTLSecondsAfterFinished]
	if !ok {
		return 0, false
	}
	ttl, err := strconv.Atoi(x)
	if err != nil {
		return 0, false
	}
	return int32(ttl), true
}

// 生成される batch Job に付与するラベルのうちで batch Job 一覧取得するために利用するもの.
func BatchJobLabelsForList(job *pixivnetv1.Job) map[string]string {
	return map[string]string{
		BatchJobLabelCreatedBy: BatchJobLabelCreatedByValue,
		BatchJobLabelName:      job.Name,
	}
}

// `jobs.v1.batch` を生成する.
// [pixivnetv1.Job] から生成する際に特有のメタデータも付与する.
func BatchJob(ctx context.Context, job *pixivnetv1.Job, podProfile *pixivnetv1.PodProfile, patcher kustomize.Patcher, scheme *runtime.Scheme) (*batchv1.Job, error) {
	nextJobSpec, err := BatchJobSpec(ctx, &job.Spec.Profile, podProfile, patcher)
	if err != nil {
		return nil, err
	}

	var batchJob batchv1.Job
	batchJob.Spec = *nextJobSpec

	//
	// metadata
	//
	batchJob.TypeMeta = metav1.TypeMeta{
		APIVersion: batchJobGroup + "/" + batchJobVersion,
		Kind:       batchJobKind,
	}
	batchJob.Namespace = job.Namespace
	// ownerRefernce をつける
	if err := ctrl.SetControllerReference(job, &batchJob, scheme); err != nil {
		return nil, fmt.Errorf("failed to add owner reference to patched job: %w", err)
	}
	batchJob.Annotations = map[string]string{}
	batchJob.Labels = map[string]string{}
	// 追加のラベルとアノテーションを先に付与する
	// controller のために必須なメタデータの上書きを回避する
	for k, v := range job.Spec.Profile.Metadata.Annotations {
		batchJob.Annotations[k] = v
	}
	for k, v := range job.Spec.Profile.Metadata.Labels {
		batchJob.Labels[k] = v
	}
	// TTL をアノテーションに保存する
	if x := job.Spec.Profile.Params.TTLSecondsAfterFinished; x != nil {
		batchJob.Annotations[BatchJobAnnotationTTLSecondsAfterFinished] = fmt.Sprintf("%d", *x)
	}
	for k, v := range BatchJobLabelsForList(job) {
		batchJob.Labels[k] = v
	}
	// テンプレートのハッシュからジョブの名前を決める
	podTemplateHash := controller.ComputeHash(&batchJob.Spec.Template, job.Status.CollisionCount)
	batchJob.Labels["pod-template-hash"] = podTemplateHash
	batchJob.Name = job.Name + "-" + podTemplateHash
	// spec の同一性を比較するためにハッシュを記録しておく
	batchJob.Labels[BatchJobLabelSpecHashKey] = util.ComputeHash(&batchJob.Spec)

	return &batchJob, nil
}

// `jobs.v1.batch.spec` を生成する.
func BatchJobSpec(ctx context.Context, jobProfileSpec *pixivnetv1.JobProfileSpec, podProfile *pixivnetv1.PodProfile, patcher kustomize.Patcher) (*batchv1.JobSpec, error) {
	var batchJob batchv1.Job

	//
	// metadata
	//
	batchJob.TypeMeta = metav1.TypeMeta{
		APIVersion: batchJobGroup + "/" + batchJobVersion,
		Kind:       batchJobKind,
	}
	batchJob.Name = "next-job" // 仮の名前. パッチをあてるために PatchRunner 呼び出しよりも前に設定する必要がある

	//
	// Job トップレベルのパラメータを設定する
	//
	var (
		spec   = &batchJob.Spec
		params = jobProfileSpec.Params
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
	// TTLSecondsAfterFinished は CRD Job 側で管理する
	// spec.TTLSecondsAfterFinished = params.TTLSecondsAfterFinished
	spec.CompletionMode = params.CompletionMode
	spec.Suspend = params.Suspend
	spec.PodReplacementPolicy = params.PodReplacementPolicy
	spec.ManagedBy = params.ManagedBy
	spec.Template = podProfile.Spec.Template // パッチをあてる対象として podprofile をそのままセットする

	if len(jobProfileSpec.Patches) == 0 { // パッチがない場合はあてる必要がない
		return &batchJob.Spec, nil
	}

	//
	// batch Job にパッチをあてる
	//
	jobYaml, err := yaml.Marshal(batchJob)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal batch job seed: %w", err)
	}
	patches := jobProfileSpec.DeepCopy().Patches
	for i := range patches {
		// ユーザーは podprofile.spec.template つまり
		// batch/v1.job.spec.template をパッチの path のルートとしてパッチを書いている
		// path を書き換えて実際にそうなるよう調整する
		patches[i].Path = "/spec/template" + patches[i].Path
	}

	patchesYaml, err := yaml.Marshal(patches)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal job patches: %w", err)
	}

	// kubectl kustomize
	patched, err := patcher.Patch(ctx, &kustomize.PatchRequest{
		Group:    batchJobGroup,
		Version:  batchJobVersion,
		Kind:     batchJobKind,
		Name:     batchJob.Name,
		Resource: string(jobYaml),
		Patches:  string(patchesYaml),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to apply job patches: %w", err)
	}

	var patchedJob batchv1.Job
	if err := yaml.Unmarshal([]byte(patched.Manifest), &patchedJob); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pacthed job: %w", err)
	}

	return &patchedJob.Spec, nil
}
