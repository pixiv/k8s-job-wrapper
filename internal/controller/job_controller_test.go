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
	"fmt"
	"os"
	"slices"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pixivnetv1 "github.com/pixiv/k8s-job-wrapper/api/v1"
	"github.com/pixiv/k8s-job-wrapper/internal/kubectl"
	"github.com/pixiv/k8s-job-wrapper/internal/kustomize"
)

var _ = Describe("Job Controller", Serial, func() {
	Context("When reconciling a resource", func() {
		ctx := context.Background()

		reconcile := func(resourceName string) error {
			controllerReconciler := &JobReconciler{
				Client:  k8sClient,
				Scheme:  k8sClient.Scheme(),
				Patcher: kustomize.NewPatchRunner(kubectl.NewCommand(os.Getenv("KUBECTL"))),
			}
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: newKey(resourceName),
			})
			return err
		}

		// List() のための selector
		newBatchJobLabels := func(resourceName string) map[string]string {
			return map[string]string{
				"app.kubernetes.io/created-by": "pixiv-job-controller",
				"jobs.pixiv.net/name":          resourceName,
			}
		}

		// テストケースの初めに用意する Job
		newJob := func(resourceName string) *pixivnetv1.Job {
			return &pixivnetv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: testNamespace,
				},
				Spec: pixivnetv1.JobSpec{
					Profile: pixivnetv1.JobProfileSpec{
						PodProfileRef: resourceName,
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
							Suspend: ptr.To(true),
						},
					},
				},
			}
		}

		// テストケースの初めに用意する TTL 指定ありの Job
		newJobWithTTL := func(resourceName string, ttl int) *pixivnetv1.Job {
			return &pixivnetv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: testNamespace,
				},
				Spec: pixivnetv1.JobSpec{
					Profile: pixivnetv1.JobProfileSpec{
						PodProfileRef: resourceName,
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
							TTLSecondsAfterFinished: ptr.To(int32(ttl)),
						},
					},
				},
			}
		}

		// テストケースの初めに用意する JobHistoryLimit 指定ありの Job
		newJobWithHistoryLimit := func(resourceName string, limit int) *pixivnetv1.Job {
			return &pixivnetv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: testNamespace,
				},
				Spec: pixivnetv1.JobSpec{
					JobsHistoryLimit: ptr.To(limit),
					Profile: pixivnetv1.JobProfileSpec{
						PodProfileRef: resourceName,
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

		// テストケースの初めに用意する複雑なパッチを持つ Job
		newComplexJob := func(resourceName string) *pixivnetv1.Job {
			return &pixivnetv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: testNamespace,
				},
				Spec: pixivnetv1.JobSpec{
					Profile: pixivnetv1.JobProfileSpec{
						PodProfileRef: resourceName,
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

		// テストケースの初めに用意するメタデータつきの Job
		newJobWithMeta := func(resourceName string) *pixivnetv1.Job {
			return &pixivnetv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: testNamespace,
				},
				Spec: pixivnetv1.JobSpec{
					Profile: pixivnetv1.JobProfileSpec{
						PodProfileRef: resourceName,
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
							Suspend: ptr.To(true),
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

		// それぞれのテストケースの先頭で呼ぶ関数
		// テストケースに使うリソース名をテストケースごとにユニークにしつつ生成する
		beforeEach := func(resourceName string) {
			By(fmt.Sprintf("creating the custom resource for the Kind PodProfile: %s", resourceName))
			Expect(k8sClient.Create(ctx, newPodProfile(resourceName))).To(Succeed())
			By(fmt.Sprintf("creating the custom resource for the Kind Job: %s", resourceName))
			Expect(k8sClient.Create(ctx, newJob(resourceName))).To(Succeed())
		}

		listBatchJobs := func(resourceName string) *batchv1.JobList {
			batchJobs := &batchv1.JobList{}
			Eventually(func() error {
				return k8sClient.List(ctx, batchJobs, &client.ListOptions{
					Namespace:     testNamespace,
					LabelSelector: labels.SelectorFromSet(newBatchJobLabels(resourceName)),
				})
			}).Should(Succeed())
			return batchJobs
		}

		assertJobStatus := func(resourceName string, key pixivnetv1.JobConditionType, status metav1.ConditionStatus, reason string, message ...string) {
			job := &pixivnetv1.Job{}
			Expect(k8sClient.Get(ctx, newKey(resourceName), job)).To(Succeed())
			assertStatus(job.Status.Conditions, string(key), status, reason, message...)
		}

		It("should failed to reconcile the resource because the PodProfile is missing", func() {
			const resourceName = "job-missing"
			By(fmt.Sprintf("creating the custom resource for the Kind Job: %s", resourceName))
			Expect(k8sClient.Create(ctx, newJob(resourceName))).To(Succeed())
			By("reconciling")
			Expect(reconcile(resourceName)).Should(HaveOccurred())
			By("making sure the Status")
			assertJobStatus(resourceName, pixivnetv1.JobAvailable, metav1.ConditionFalse, "Reconciling", "PodProfile not found")
			assertJobStatus(resourceName, pixivnetv1.JobDegraded, metav1.ConditionTrue, "Reconciling")
			By("making sure no batch Jobs are generated")
			Expect(listBatchJobs(resourceName).Items).Should(BeEmpty())
		})

		It("should successfully reconile the resource with complex patches", func() {
			const resourceName = "job-complex"
			By("making sure the PodProfile created successfully")
			Expect(k8sClient.Create(ctx, newPodProfile(resourceName))).To(Succeed())
			By("making sure the Job with complex patches created successfully")
			Expect(k8sClient.Create(ctx, newComplexJob(resourceName))).To(Succeed())
			By("reconciling")
			Expect(reconcile(resourceName)).To(Succeed())
			By("making sure the batch Job created successfully")
			batchJobs := listBatchJobs(resourceName)
			Expect(batchJobs.Items).Should(HaveLen(1))
			batchJob := batchJobs.Items[0]
			Expect(batchJob.Spec.Template.Spec.Containers).Should(HaveLen(2))
			Expect(batchJob.Spec.Template.Spec.Containers[1].Name).To(Equal("added"))
			Expect(batchJob.Spec.Template.Spec.Containers[1].Image).To(Equal("debian:bookworm"))
			Expect(batchJob.Spec.Template.Spec.Containers[1].Command).To(Equal([]string{"sleep", "1"}))
		})

		It("should successfully reconcile the resource with metadata patches", func() {
			const resourceName = "job-meta"
			By("making sure the PodProfile created successfully")
			Expect(k8sClient.Create(ctx, newPodProfile(resourceName))).To(Succeed())
			By("making sure the Job with metadata created successfully")
			Expect(k8sClient.Create(ctx, newJobWithMeta(resourceName))).To(Succeed())
			By("reconciling")
			Expect(reconcile(resourceName)).To(Succeed())
			By("making sure the batch Job created successfully")
			batchJobs := listBatchJobs(resourceName)
			Expect(batchJobs.Items).Should(HaveLen(1))
			batchJob := batchJobs.Items[0]
			if Expect(batchJob.Annotations).Should(HaveKey("case")) {
				Expect(batchJob.GetAnnotations()["case"]).To(Equal("withMeta"))
			}
			if Expect(batchJob.Labels).Should(HaveKey("additional")) {
				Expect(batchJob.GetLabels()["additional"]).To(Equal("label"))
			}
		})

		It("should successfully reconcile the resource", func() {
			const resourceName = "job-reconcile"
			beforeEach(resourceName)

			By("Reconciling the created resource")
			Expect(reconcile(resourceName)).To(Succeed())

			By("making sure the batch Job created successfully")
			batchJobs := listBatchJobs(resourceName)
			Expect(batchJobs.Items).Should(HaveLen(1))

			batchJob := batchJobs.Items[0]
			Expect(batchJob.Spec.Suspend).Should(Equal(ptr.To(true)))
			Expect(batchJob.Spec.Template.Spec.RestartPolicy).Should(Equal(corev1.RestartPolicyNever))
			Expect(batchJob.Spec.Template.Spec.Containers).Should(HaveLen(1))
			Expect(batchJob.Spec.Template.Spec.Containers[0].Name).Should(Equal("pi"))
			Expect(batchJob.Spec.Template.Spec.Containers[0].Image).Should(Equal("debian:bookworm-slim"))
			Expect(batchJob.Spec.Template.Spec.Containers[0].Command).Should(Equal([]string{
				"perl", "-Mbignum=bpi", "-wle", "print bpi(2000)",
			}))

			By("making sure the Status updated successfully")
			updated := &pixivnetv1.Job{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, newKey(resourceName), updated)).To(Succeed())
				g.Expect(updated.Status.Conditions).ShouldNot(BeEmpty(), "status should be updated")
			}).Should(Succeed())
		})

		It("should not delete expired without multiple batch jobs", func() {
			const resourceName = "job-ttl-single-batch-job"
			now := time.Now() // expired 判定には現在時刻との差が重要
			By("making sure the PodProfile created successfully")
			Expect(k8sClient.Create(ctx, newPodProfile(resourceName))).To(Succeed())
			By("making sure the Job with ttl created successfully")
			Expect(k8sClient.Create(ctx, newJobWithTTL(resourceName, 1))).To(Succeed())
			By("Reconciling the created resource")
			Expect(reconcile(resourceName)).To(Succeed())
			By("making sure the batch Job created successfully")
			{
				batchJobs := listBatchJobs(resourceName)
				Expect(batchJobs.Items).Should(HaveLen(1))
				batchJob := batchJobs.Items[0]
				Expect(batchJob.Status.Conditions).Should(BeEmpty())
				By("set the batch Job status complete")
				metaNow := metav1.NewTime(now)
				batchJob.Status = newBatchJobCompleteStatus(metaNow, metaNow)
				Expect(k8sClient.Status().Update(ctx, &batchJob)).To(Succeed())
			}
			By("wait a second to expire the batch Job")
			time.Sleep(time.Second)
			Consistently(func(g Gomega) {
				By("Reconciling the created resource")
				g.Expect(reconcile(resourceName)).To(Succeed())
				By("making sure the batch Job is remaining")
				batchJobs := listBatchJobs(resourceName)
				g.Expect(batchJobs.Items).Should(HaveLen(1))
				batchJob := batchJobs.Items[0]
				g.Expect(batchJob.DeletionTimestamp).To(BeNil())                                    // 削除されていない
				g.Expect(batchJob.GetAnnotations()["jobs.pixiv.net/ttl-expired"]).To(Equal("true")) // 削除予定のマークはある
			}).Should(Succeed())
		})

		It("should delete expired batch jobs", func() {
			const resourceName = "job-ttl"
			now := time.Now() // expired 判定には現在時刻との差が重要
			By("making sure the PodProfile created successfully")
			Expect(k8sClient.Create(ctx, newPodProfile(resourceName))).To(Succeed())
			By("making sure the Job with ttl created successfully")
			Expect(k8sClient.Create(ctx, newJobWithTTL(resourceName, 3600))).To(Succeed())
			By("Reconciling the created resource")
			Expect(reconcile(resourceName)).To(Succeed())
			By("making sure the batch Job created successfully")
			{
				batchJobs := listBatchJobs(resourceName)
				Expect(batchJobs.Items).Should(HaveLen(1))
				batchJob := batchJobs.Items[0]
				Expect(batchJob.Status.Conditions).Should(BeEmpty())
				By("set the batch Job status complete")
				metaNow := metav1.NewTime(now)
				batchJob.Status = newBatchJobCompleteStatus(metaNow, metaNow)
				Expect(k8sClient.Status().Update(ctx, &batchJob)).To(Succeed())
			}
			By("Reconciling the created resource")
			Expect(reconcile(resourceName)).To(Succeed())
			By("making sure the batch Job is remaining because TTL is remaining yet")
			var batchJob1UID types.UID
			{
				batchJobs := listBatchJobs(resourceName)
				Expect(batchJobs.Items).Should(HaveLen(1))
				batchJob1UID = batchJobs.Items[0].GetUID()
			}
			By("update the Job: ttl=1")
			{
				job := &pixivnetv1.Job{}
				Expect(k8sClient.Get(ctx, newKey(resourceName), job)).To(Succeed())
				job.Spec.Profile.Patches = nil
				job.Spec.Profile.Params.TTLSecondsAfterFinished = ptr.To(int32(1))
				Expect(k8sClient.Update(ctx, job)).To(Succeed())
			}
			By("wait a second for new batch job creation time")
			time.Sleep(time.Second)
			By("Reconciling the created resource")
			Expect(reconcile(resourceName)).To(Succeed())
			By("making sure the new batch Job is created")
			var batchJob2UID types.UID
			{
				batchJobs := listBatchJobs(resourceName)
				Expect(batchJobs.Items).Should(HaveLen(2))
				idx := slices.IndexFunc(batchJobs.Items, func(x batchv1.Job) bool {
					return x.GetUID() != batchJob1UID
				})
				Expect(idx >= 0).To(BeTrue())
				batchJob := batchJobs.Items[idx]
				batchJob2UID = batchJob.GetUID()
				By("set the new batch Job status complete")
				metaNow := metav1.NewTime(now)
				batchJob.Status = newBatchJobCompleteStatus(metaNow, metaNow)
				Expect(k8sClient.Status().Update(ctx, &batchJob)).To(Succeed())
			}
			By("wait a second to expire the new batch Job")
			time.Sleep(time.Second)
			By("Reconciling the created resource")
			Expect(reconcile(resourceName)).To(Succeed())
			By("making sure the new batch Job is marked to delete")
			{
				batchJobs := listBatchJobs(resourceName)
				Expect(batchJobs.Items).Should(HaveLen(2))
				idx := slices.IndexFunc(batchJobs.Items, func(x batchv1.Job) bool {
					return x.GetUID() == batchJob2UID
				})
				Expect(idx > 0).To(BeTrue())
				batchJob := batchJobs.Items[idx]
				Expect(batchJob.DeletionTimestamp).To(BeNil())                                    // まだ削除はされてない
				Expect(batchJob.GetAnnotations()["jobs.pixiv.net/ttl-expired"]).To(Equal("true")) // 削除予定のマーク
			}
			By("Reconciling the created resource")
			Expect(reconcile(resourceName)).To(Succeed())
			By("making sure the new batch Job is deleted because TTL is expired")
			{
				batchJobs := listBatchJobs(resourceName)
				Expect(batchJobs.Items).Should(HaveLen(2))
				idx1 := slices.IndexFunc(batchJobs.Items, func(x batchv1.Job) bool {
					return x.GetUID() == batchJob1UID
				})
				idx2 := slices.IndexFunc(batchJobs.Items, func(x batchv1.Job) bool {
					return x.GetUID() == batchJob2UID
				})
				Expect(idx1 >= 0).To(BeTrue())
				Expect(idx2 >= 0).To(BeFalse()) // 削除された
				Expect(batchJobs.Items[idx1].DeletionTimestamp).To(BeNil())
			}
		})

		It("should delete old batch jobs", func() {
			const resourceName = "job-delete-olds"
			now := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
			By("making sure the PodProfile created successfully")
			Expect(k8sClient.Create(ctx, newPodProfile(resourceName))).To(Succeed())
			By("making sure the Job with history limit created successfully")
			Expect(k8sClient.Create(ctx, newJobWithHistoryLimit(resourceName, 1))).To(Succeed())
			By("Reconciling the created resource")
			Expect(reconcile(resourceName)).To(Succeed())
			By("making sure the batch Job created successfully")
			var batchJob1UID types.UID
			{
				batchJobs := listBatchJobs(resourceName)
				Expect(batchJobs.Items).Should(HaveLen(1))
				batchJob := batchJobs.Items[0]
				batchJob1UID = batchJob.GetUID()
				Expect(batchJob.Status.Conditions).Should(BeEmpty())
				By("set the batch Job status complete")
				metaNow := metav1.NewTime(now)
				batchJob.Status = newBatchJobCompleteStatus(metaNow, metaNow)
				Expect(k8sClient.Status().Update(ctx, &batchJob)).To(Succeed())
			}
			By("set the Job patches empty")
			{
				job := &pixivnetv1.Job{}
				Expect(k8sClient.Get(ctx, newKey(resourceName), job)).To(Succeed())
				job.Spec.Profile.Patches = nil
				Expect(k8sClient.Update(ctx, job)).To(Succeed())
			}
			By("wait for 1 second to create a gap in job creation time")
			time.Sleep(time.Second)
			By("Reconciling the created resource")
			Expect(reconcile(resourceName)).To(Succeed())
			By("making sure the old batch job and the new batch job")
			var batchJob2UID types.UID
			{
				batchJobs := listBatchJobs(resourceName)
				Expect(batchJobs.Items).Should(HaveLen(2))
				idx := slices.IndexFunc(batchJobs.Items, func(x batchv1.Job) bool {
					return x.GetUID() != batchJob1UID
				})
				Expect(idx >= 0).Should(BeTrue())
				batchJob := batchJobs.Items[idx]
				batchJob2UID = batchJob.GetUID()
				By("set the batch Job status complete")
				metaNow := metav1.NewTime(now)
				batchJob.Status = newBatchJobCompleteStatus(metaNow, metaNow)
				Expect(k8sClient.Status().Update(ctx, &batchJob)).To(Succeed())
			}
			By("update the Job patches")
			{
				job := &pixivnetv1.Job{}
				Expect(k8sClient.Get(ctx, newKey(resourceName), job)).To(Succeed())
				job.Spec.Profile.Patches = newJobWithHistoryLimit(resourceName, 1).Spec.Profile.Patches
				job.Spec.Profile.Patches[0].Value = apiextensionsv1.JSON{
					Raw: []byte(`"debian:bookworm"`),
				}
				Expect(k8sClient.Update(ctx, job)).To(Succeed())
			}
			By("wait for 1 second to create a gap in job creation time")
			time.Sleep(time.Second)
			By("Reconciling the created resource")
			Expect(reconcile(resourceName)).To(Succeed())
			By("making sure the oldest job is deleted")
			Eventually(func(g Gomega) {
				batchJobs := listBatchJobs(resourceName)
				items := []batchv1.Job{}
				for _, x := range batchJobs.Items {
					// controller が delete している batch job を除外する
					if x.DeletionTimestamp == nil {
						items = append(items, x)
					}
				}
				g.Expect(items).Should(HaveLen(2))
				batchJob2Idx := slices.IndexFunc(items, func(x batchv1.Job) bool {
					return x.GetUID() == batchJob2UID
				})
				g.Expect(batchJob2Idx >= 0).Should(BeTrue())
				batchJob3Idx := slices.IndexFunc(items, func(x batchv1.Job) bool {
					return !slices.Contains([]types.UID{batchJob1UID, batchJob2UID}, x.GetUID())
				})
				g.Expect(batchJob3Idx >= 0).Should(BeTrue())
			}).Should(Succeed())
		})

		//
		// batch Job 再生成のテスト
		//
		var (
			now        = metav1.NewTime(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC))
			patchedJob = func(resourceName string) *pixivnetv1.Job {
				x := newJob(resourceName)
				x.Spec.Profile.Params.ActiveDeadlineSeconds = ptr.To(int64(60))
				return x
			}
			assertPatchedJob = func(x *batchv1.Job) {
				v := x.Spec.ActiveDeadlineSeconds
				if Expect(v).ShouldNot(BeNil()) {
					Expect(*v).Should(BeEquivalentTo(60))
				}
			}
			patchedPodProfile = func(resourceName string) *pixivnetv1.PodProfile {
				x := newPodProfile(resourceName)
				x.Spec.Template.Spec.Containers[0].Name = "patched"
				return x
			}
			assertPatchedPodProfile = func(x *batchv1.Job) {
				Expect(x.Spec.Template.Spec.Containers[0].Name).Should(Equal("patched"))
			}
		)
		for caseNo, tc := range []struct {
			title      string
			job        func(string) *pixivnetv1.Job        // job 変更用, nil なら変更なし
			podprofile func(string) *pixivnetv1.PodProfile // profile 変更用, nil なら変更なし
			status     *batchv1.JobStatus                  // status 変更用, nil なら変更なし
			recreated  bool                                // batch job 再生成を期待するなら true
			assertJob  func(*batchv1.Job)                  // 更新後の job のアサーション, nil なら何もしない
		}{
			{
				title:     "batch Job is not recreated because it is running (conditions is empty)",
				recreated: false,
			},
			{
				title:     "batch Job is not recreated because it is running (conditions is empty) even if Job is updated",
				job:       patchedJob,
				recreated: false,
			},
			{
				title: "batch Job is not recreated because it is suspended",
				status: &batchv1.JobStatus{
					Conditions: []batchv1.JobCondition{
						{
							Type:   batchv1.JobSuspended,
							Status: corev1.ConditionTrue,
							Reason: "JobSuspended",
						},
					},
				},
				recreated: false,
			},
			{
				title: "batch Job is not recreated because it is suspended even if Job is updated",
				job:   patchedJob,
				status: &batchv1.JobStatus{
					Conditions: []batchv1.JobCondition{
						{
							Type:   batchv1.JobSuspended,
							Status: corev1.ConditionTrue,
							Reason: "JobSuspended",
						},
					},
				},
				recreated: false,
			},
			{
				title:     "batch Job is recreated because it is failed and Job is updated",
				job:       patchedJob,
				status:    ptr.To(newBatchJobFailedStatus(now)),
				recreated: true,
				assertJob: assertPatchedJob,
			},
			{
				title:     "batch Job is not recreated because it is failed but no update",
				status:    ptr.To(newBatchJobFailedStatus(now)),
				recreated: false,
			},
			{
				title:      "batch Job is recreated because it is completed and PodProfile is updated",
				podprofile: patchedPodProfile,
				status:     ptr.To(newBatchJobCompleteStatus(now, now)),
				recreated:  true,
				assertJob:  assertPatchedPodProfile,
			},
			{
				title:     "batch Job is not recreated because it is completed but no update",
				status:    ptr.To(newBatchJobCompleteStatus(now, now)),
				recreated: false,
			},
			{
				title:     "batch Job is recreated because it is completed and Job is updated",
				job:       patchedJob,
				status:    ptr.To(newBatchJobCompleteStatus(now, now)),
				recreated: true,
				assertJob: assertPatchedJob,
			},
		} {
			It(tc.title, func() {
				resourceName := fmt.Sprintf("job-recreate%d", caseNo)
				beforeEach(resourceName)
				Expect(reconcile(resourceName)).To(Succeed())

				By("making sure the batch Job created")
				var (
					uid                     types.UID // 最初の batch Job を判別するための id
					batchTypeNamespacedName types.NamespacedName
				)
				{
					batchJobs := listBatchJobs(resourceName)
					Expect(batchJobs.Items).Should(HaveLen(1))
					batchJob := batchJobs.Items[0]
					Expect(batchJob.Status.Conditions).Should(BeEmpty())
					uid = batchJob.GetUID()
					batchTypeNamespacedName = types.NamespacedName{
						Namespace: testNamespace,
						Name:      batchJob.Name,
					}
				}

				if job := tc.job; job != nil {
					By("change the Job")
					var current pixivnetv1.Job
					Expect(k8sClient.Get(ctx, newKey(resourceName), &current)).To(Succeed())
					x := job(resourceName)
					x.ResourceVersion = current.ResourceVersion
					Expect(k8sClient.Update(ctx, x)).To(Succeed())
				}
				if podprofile := tc.podprofile; podprofile != nil {
					By("change the PodProfile")
					var current pixivnetv1.PodProfile
					Expect(k8sClient.Get(ctx, newKey(resourceName), &current)).To(Succeed())
					x := podprofile(resourceName)
					x.ResourceVersion = current.ResourceVersion
					Expect(k8sClient.Update(ctx, x)).To(Succeed())
				}
				if status := tc.status; status != nil {
					By("change the batch Job status")
					batchJob := &batchv1.Job{}
					Expect(k8sClient.Get(ctx, batchTypeNamespacedName, batchJob)).To(Succeed())
					batchJob.Status = *tc.status
					Expect(k8sClient.Status().Update(ctx, batchJob)).To(Succeed())
				}

				By("reconcile")
				// batch Job 作る時に名前が衝突することがある
				// Reconcile() 呼び直すことで生成の seed が変わるので回避される
				Eventually(func() error {
					return reconcile(resourceName)
				}).Should(Succeed())

				By("check the batch Job")
				batchJobs := listBatchJobs(resourceName)
				if tc.recreated {
					By("making sure tha batch Job is recreated")
					Expect(batchJobs.Items).Should(HaveLen(2))
					var newJob *batchv1.Job
					for _, x := range batchJobs.Items {
						if x.GetUID() != uid {
							newJob = &x
						}
					}
					Expect(newJob).ShouldNot(BeNil())
					if f := tc.assertJob; f != nil {
						f(newJob)
					}
				} else {
					By("making sure tha batch Job is not recreated")
					Expect(batchJobs.Items).Should(HaveLen(1))
					batchJob := batchJobs.Items[0]
					Expect(batchJob.GetUID()).Should(Equal(uid))
					if f := tc.assertJob; f != nil {
						f(&batchJob)
					}
				}

				By("check the Status")
				assertJobStatus(resourceName, pixivnetv1.JobAvailable, metav1.ConditionTrue, "OK")
				assertJobStatus(resourceName, pixivnetv1.JobDegraded, metav1.ConditionFalse, "OK")
			})
		}
	})
})
