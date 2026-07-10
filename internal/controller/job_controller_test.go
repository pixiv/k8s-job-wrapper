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
	"fmt"
	"slices"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pixivnetv1 "github.com/pixiv/k8s-job-wrapper/api/v1"
	"github.com/pixiv/k8s-job-wrapper/internal/kubectl"
	"github.com/pixiv/k8s-job-wrapper/internal/kustomize"
	"github.com/pixiv/k8s-job-wrapper/test/utils"
)

var _ = new(jobControllerTest).run()

type jobControllerTest struct {
	controllerTestBase
}

func (j jobControllerTest) run() bool {
	contexts := []controllerTestContext{
		j.reconcilePodProfileMissing(),
		j.reconcileComplexPatch(),
		j.reconcileMeta(),
		j.reconcileNormal(),
		j.reconcileTTLSingleBatchJob(),
		j.reconcileTTL(),
		j.reconcileDeleteOldJobs(),
	}
	contexts = append(contexts, j.reconcileRecreateJobs()...)
	return controllerTest{
		name:            "JobController",
		namespacePrefix: "job",
		contexts:        contexts,
	}.run()
}

func (jobControllerTest) newReconciler() *JobReconciler {
	return &JobReconciler{
		Client:  k8sClient,
		Scheme:  k8sClient.Scheme(),
		Patcher: kustomize.NewPatchRunner(kubectl.NewCommand(utils.Kubectl())),
	}
}

func (j jobControllerTest) assertJobStatus(
	namespace, resourceName string,
	key pixivnetv1.JobConditionType,
	status metav1.ConditionStatus,
	reason string,
	message ...string,
) {
	GinkgoHelper()
	job, err := Get[*pixivnetv1.Job](ctx, k8sClient, j.newNSName(namespace, resourceName))
	Expect(err).To(Succeed())
	j.assertStatus(job.Status.Conditions, string(key), status, reason, message...)
}

func (j jobControllerTest) reconcilePodProfileMissing() controllerTestContext {
	const resourceName = "job-missing"
	return controllerTestContext{
		name: "When reconciling a resource without PodProfile",
		beforeEach: func(a *controllerTestContextArg) {
			By(fmt.Sprintf("creating the custom resource for the Kind Job: %s", resourceName))
			Expect(k8sClient.Create(ctx, j.newJob(a.namespace, resourceName, resourceName))).To(Succeed())
		},
		afterEach: func(a *controllerTestContextArg) {
			job, err := Get[*pixivnetv1.Job](ctx, k8sClient, j.newNSName(a.namespace, resourceName))
			Expect(err).To(Succeed())
			By("cleanup the specific resource instance Job")
			Expect(k8sClient.Delete(ctx, job)).To(Succeed())
		},
		test: func(a *controllerTestContextArg) {
			typeNamespacedName := j.newNSName(a.namespace, resourceName)
			It("should failed to reconcile the resource because the PodProfile is missing", func() {
				reconciler := j.newReconciler()
				By("reconciling")
				Expect(j.reconcile(ctx, reconciler, typeNamespacedName)).Should(HaveOccurred())
				By("making sure the Status")
				j.assertJobStatus(a.namespace, resourceName, pixivnetv1.JobAvailable, metav1.ConditionFalse, "Reconciling", "PodProfile not found")
				j.assertJobStatus(a.namespace, resourceName, pixivnetv1.JobDegraded, metav1.ConditionTrue, "Reconciling")
				By("making sure no batch Jobs are generated")
				job, err := Get[*pixivnetv1.Job](ctx, k8sClient, typeNamespacedName)
				Expect(err).To(Succeed())
				batchJobs, err := ListBatchJobsFromPixivNetJob(ctx, k8sClient, job)
				Expect(err).To(Succeed())
				Expect(batchJobs.Items).Should(BeEmpty())
			})
		},
	}
}

func (j jobControllerTest) reconcileComplexPatch() controllerTestContext {
	const resourceName = "job-complex"
	return controllerTestContext{
		name: "When reconciling a resource with complex patch",
		beforeEach: func(a *controllerTestContextArg) {
			By(fmt.Sprintf("creating the custom resource for the Kind PodProfile: %s", resourceName))
			Expect(k8sClient.Create(ctx, j.newPodProfile(a.namespace, resourceName))).To(Succeed())
			By(fmt.Sprintf("creating the custom resource for the Kind Job: %s", resourceName))
			Expect(k8sClient.Create(ctx, j.newJobWithComplexPatch(a.namespace, resourceName, resourceName))).To(Succeed())
		},
		afterEach: func(a *controllerTestContextArg) {
			job, err := Get[*pixivnetv1.Job](ctx, k8sClient, j.newNSName(a.namespace, resourceName))
			Expect(err).To(Succeed())
			By("cleanup the specific resource instance Job")
			Expect(k8sClient.Delete(ctx, job)).To(Succeed())

			podProfile, err := Get[*pixivnetv1.PodProfile](ctx, k8sClient, j.newNSName(a.namespace, resourceName))
			Expect(err).To(Succeed())
			By("cleanup the specific resource instance PodProfile")
			Expect(k8sClient.Delete(ctx, podProfile)).To(Succeed())
		},
		test: func(a *controllerTestContextArg) {
			typeNamespacedName := j.newNSName(a.namespace, resourceName)
			It("should successfully reconcile the resource", func() {
				reconciler := j.newReconciler()
				By("reconciling")
				Expect(j.reconcile(ctx, reconciler, typeNamespacedName)).To(Succeed())
				By("making sure the batch Job created successfully")
				job, err := Get[*pixivnetv1.Job](ctx, k8sClient, typeNamespacedName)
				Expect(err).To(Succeed())
				batchJobs, err := ListBatchJobsFromPixivNetJob(ctx, k8sClient, job)
				Expect(err).To(Succeed())
				Expect(batchJobs.Items).Should(HaveLen(1))
				batchJob := batchJobs.Items[0]
				Expect(batchJob.Spec.Template.Spec.Containers).Should(HaveLen(2))
				Expect(batchJob.Spec.Template.Spec.Containers[1].Name).To(Equal("added"))
				Expect(batchJob.Spec.Template.Spec.Containers[1].Image).To(Equal("debian:bookworm"))
				Expect(batchJob.Spec.Template.Spec.Containers[1].Command).To(Equal([]string{"sleep", "1"}))
			})
		},
	}
}

func (j jobControllerTest) reconcileMeta() controllerTestContext {
	const resourceName = "job-meta"
	return controllerTestContext{
		name: "When reconciling a resource with complex patch",
		beforeEach: func(a *controllerTestContextArg) {
			By(fmt.Sprintf("creating the custom resource for the Kind PodProfile: %s", resourceName))
			Expect(k8sClient.Create(ctx, j.newPodProfile(a.namespace, resourceName))).To(Succeed())
			By(fmt.Sprintf("creating the custom resource for the Kind Job: %s", resourceName))
			Expect(k8sClient.Create(ctx, j.newJobWithMeta(a.namespace, resourceName, resourceName))).To(Succeed())
		},
		afterEach: func(a *controllerTestContextArg) {
			job, err := Get[*pixivnetv1.Job](ctx, k8sClient, j.newNSName(a.namespace, resourceName))
			Expect(err).To(Succeed())
			By("cleanup the specific resource instance Job")
			Expect(k8sClient.Delete(ctx, job)).To(Succeed())

			podProfile, err := Get[*pixivnetv1.PodProfile](ctx, k8sClient, j.newNSName(a.namespace, resourceName))
			Expect(err).To(Succeed())
			By("cleanup the specific resource instance PodProfile")
			Expect(k8sClient.Delete(ctx, podProfile)).To(Succeed())
		},
		test: func(a *controllerTestContextArg) {
			typeNamespacedName := j.newNSName(a.namespace, resourceName)
			It("should successfully reconcile the resource", func() {
				reconciler := j.newReconciler()
				By("reconciling")
				Expect(j.reconcile(ctx, reconciler, typeNamespacedName)).To(Succeed())
				By("making sure the batch Job created successfully")
				job, err := Get[*pixivnetv1.Job](ctx, k8sClient, typeNamespacedName)
				Expect(err).To(Succeed())
				batchJobs, err := ListBatchJobsFromPixivNetJob(ctx, k8sClient, job)
				Expect(err).To(Succeed())
				Expect(batchJobs.Items).Should(HaveLen(1))
				batchJob := batchJobs.Items[0]
				if Expect(batchJob.Annotations).Should(HaveKey("case")) {
					Expect(batchJob.GetAnnotations()["case"]).To(Equal("withMeta"))
				}
				if Expect(batchJob.Labels).Should(HaveKey("additional")) {
					Expect(batchJob.GetLabels()["additional"]).To(Equal("label"))
				}
			})
		},
	}
}

func (j jobControllerTest) reconcileNormal() controllerTestContext {
	const resourceName = "job-reconcile"
	return controllerTestContext{
		name: "When reconciling a resource",
		beforeEach: func(a *controllerTestContextArg) {
			By(fmt.Sprintf("creating the custom resource for the Kind PodProfile: %s", resourceName))
			Expect(k8sClient.Create(ctx, j.newPodProfile(a.namespace, resourceName))).To(Succeed())
			By(fmt.Sprintf("creating the custom resource for the Kind Job: %s", resourceName))
			Expect(k8sClient.Create(ctx, j.newJob(a.namespace, resourceName, resourceName))).To(Succeed())
		},
		afterEach: func(a *controllerTestContextArg) {
			job, err := Get[*pixivnetv1.Job](ctx, k8sClient, j.newNSName(a.namespace, resourceName))
			Expect(err).To(Succeed())
			By("cleanup the specific resource instance Job")
			Expect(k8sClient.Delete(ctx, job)).To(Succeed())

			podProfile, err := Get[*pixivnetv1.PodProfile](ctx, k8sClient, j.newNSName(a.namespace, resourceName))
			Expect(err).To(Succeed())
			By("cleanup the specific resource instance PodProfile")
			Expect(k8sClient.Delete(ctx, podProfile)).To(Succeed())
		},
		test: func(a *controllerTestContextArg) {
			typeNamespacedName := j.newNSName(a.namespace, resourceName)
			It("should successfully reconcile the resource", func() {
				reconciler := j.newReconciler()
				By("reconciling")
				Expect(j.reconcile(ctx, reconciler, typeNamespacedName)).To(Succeed())
				By("making sure the batch Job created successfully")
				job, err := Get[*pixivnetv1.Job](ctx, k8sClient, typeNamespacedName)
				Expect(err).To(Succeed())
				batchJobs, err := ListBatchJobsFromPixivNetJob(ctx, k8sClient, job)
				Expect(err).To(Succeed())
				Expect(batchJobs.Items).Should(HaveLen(1))
				batchJob := batchJobs.Items[0]
				Expect(batchJob.Spec.Suspend).Should(Equal(new(true)))
				Expect(batchJob.Spec.Template.Spec.RestartPolicy).Should(Equal(corev1.RestartPolicyNever))
				Expect(batchJob.Spec.Template.Spec.Containers).Should(HaveLen(1))
				Expect(batchJob.Spec.Template.Spec.Containers[0].Name).Should(Equal("pi"))
				Expect(batchJob.Spec.Template.Spec.Containers[0].Image).Should(Equal("debian:bookworm-slim"))
				Expect(batchJob.Spec.Template.Spec.Containers[0].Command).Should(Equal([]string{
					"perl", "-Mbignum=bpi", "-wle", "print bpi(2000)",
				}))
				By("making sure the Status updated successfully")
				Expect(job.Status.Conditions).ShouldNot(BeEmpty(), "status should be updated")
			})
		},
	}
}

func (j jobControllerTest) reconcileTTLSingleBatchJob() controllerTestContext {
	const resourceName = "job-ttl-single-batch-job"
	return controllerTestContext{
		name: "When reconciling a resource with TTL",
		beforeEach: func(a *controllerTestContextArg) {
			By(fmt.Sprintf("creating the custom resource for the Kind PodProfile: %s", resourceName))
			Expect(k8sClient.Create(ctx, j.newPodProfile(a.namespace, resourceName))).To(Succeed())
			By(fmt.Sprintf("creating the custom resource for the Kind Job: %s", resourceName))
			Expect(k8sClient.Create(ctx, j.newJobWithTTL(a.namespace, resourceName, resourceName, 1))).To(Succeed())
		},
		afterEach: func(a *controllerTestContextArg) {
			job, err := Get[*pixivnetv1.Job](ctx, k8sClient, j.newNSName(a.namespace, resourceName))
			Expect(err).To(Succeed())
			By("cleanup the specific resource instance Job")
			Expect(k8sClient.Delete(ctx, job)).To(Succeed())

			podProfile, err := Get[*pixivnetv1.PodProfile](ctx, k8sClient, j.newNSName(a.namespace, resourceName))
			Expect(err).To(Succeed())
			By("cleanup the specific resource instance PodProfile")
			Expect(k8sClient.Delete(ctx, podProfile)).To(Succeed())
		},
		test: func(a *controllerTestContextArg) {
			typeNamespacedName := j.newNSName(a.namespace, resourceName)
			It("should not delete expired job without multiple batch jobs", func() {
				now := time.Now()
				reconciler := j.newReconciler()
				By("reconciling")
				Expect(j.reconcile(ctx, reconciler, typeNamespacedName)).To(Succeed())
				By("making sure the batch Job created successfully")
				{
					job, err := Get[*pixivnetv1.Job](ctx, k8sClient, typeNamespacedName)
					Expect(err).To(Succeed())
					batchJobs, err := ListBatchJobsFromPixivNetJob(ctx, k8sClient, job)
					Expect(err).To(Succeed())
					Expect(batchJobs.Items).Should(HaveLen(1))
					batchJob := batchJobs.Items[0]
					Expect(batchJob.Status.Conditions).Should(BeEmpty())
					By("set the batch Job status complete")
					metaNow := metav1.NewTime(now)
					batchJob.Status = j.newBatchJobCompleteStatus(metaNow, metaNow)
					Expect(k8sClient.Status().Update(ctx, &batchJob)).To(Succeed())
				}
				By("wait a second to expire the batch Job")
				time.Sleep(time.Second)
				Consistently(func(g Gomega) {
					By("Reconciling the created resource")
					g.Expect(j.reconcile(ctx, reconciler, typeNamespacedName)).To(Succeed())
					By("making sure the batch Job is remaining")
					job, err := Get[*pixivnetv1.Job](ctx, k8sClient, typeNamespacedName)
					g.Expect(err).To(Succeed())
					batchJobs, err := ListBatchJobsFromPixivNetJob(ctx, k8sClient, job)
					g.Expect(err).To(Succeed())
					g.Expect(batchJobs.Items).Should(HaveLen(1))
					batchJob := batchJobs.Items[0]
					g.Expect(batchJob.DeletionTimestamp).To(BeNil())
					g.Expect(batchJob.GetAnnotations()["jobs.pixiv.net/ttl-expired"]).To(Equal("true")) // marked to be deleted
				}).Should(Succeed())
			})
		},
	}
}

func (j jobControllerTest) reconcileTTL() controllerTestContext {
	const resourceName = "job-ttl"
	return controllerTestContext{
		name: "When reconciling a resource with TTL",
		beforeEach: func(a *controllerTestContextArg) {
			By(fmt.Sprintf("creating the custom resource for the Kind PodProfile: %s", resourceName))
			Expect(k8sClient.Create(ctx, j.newPodProfile(a.namespace, resourceName))).To(Succeed())
			By(fmt.Sprintf("creating the custom resource for the Kind Job: %s", resourceName))
			Expect(k8sClient.Create(ctx, j.newJobWithTTL(a.namespace, resourceName, resourceName, 3600))).To(Succeed())
		},
		afterEach: func(a *controllerTestContextArg) {
			job, err := Get[*pixivnetv1.Job](ctx, k8sClient, j.newNSName(a.namespace, resourceName))
			Expect(err).To(Succeed())
			By("cleanup the specific resource instance Job")
			Expect(k8sClient.Delete(ctx, job)).To(Succeed())

			podProfile, err := Get[*pixivnetv1.PodProfile](ctx, k8sClient, j.newNSName(a.namespace, resourceName))
			Expect(err).To(Succeed())
			By("cleanup the specific resource instance PodProfile")
			Expect(k8sClient.Delete(ctx, podProfile)).To(Succeed())
		},
		test: func(a *controllerTestContextArg) {
			typeNamespacedName := j.newNSName(a.namespace, resourceName)
			It("should delete expired batch jobs", func() {
				now := time.Now()
				reconciler := j.newReconciler()
				By("Reconciling the created resource")
				Expect(j.reconcile(ctx, reconciler, typeNamespacedName)).To(Succeed())
				By("making sure the batch Job created successfully")
				{
					job, err := Get[*pixivnetv1.Job](ctx, k8sClient, typeNamespacedName)
					Expect(err).To(Succeed())
					batchJobs, err := ListBatchJobsFromPixivNetJob(ctx, k8sClient, job)
					Expect(err).To(Succeed())
					Expect(batchJobs.Items).Should(HaveLen(1))
					batchJob := batchJobs.Items[0]
					Expect(batchJob.Status.Conditions).Should(BeEmpty())
					By("set the batch Job status complete")
					metaNow := metav1.NewTime(now)
					batchJob.Status = j.newBatchJobCompleteStatus(metaNow, metaNow)
					Expect(k8sClient.Status().Update(ctx, &batchJob)).To(Succeed())
				}
				By("Reconciling the created resource")
				Expect(j.reconcile(ctx, reconciler, typeNamespacedName)).To(Succeed())
				By("making sure the batch Job is remaining because TTL is remaining yet")
				var batchJob1UID types.UID
				{
					job, err := Get[*pixivnetv1.Job](ctx, k8sClient, typeNamespacedName)
					Expect(err).To(Succeed())
					batchJobs, err := ListBatchJobsFromPixivNetJob(ctx, k8sClient, job)
					Expect(err).To(Succeed())
					Expect(batchJobs.Items).Should(HaveLen(1))
					batchJob1UID = batchJobs.Items[0].GetUID()
				}
				By("update the Job: ttl=1")
				{
					job, err := Get[*pixivnetv1.Job](ctx, k8sClient, typeNamespacedName)
					Expect(err).To(Succeed())
					job.Spec.Profile.Patches = nil
					job.Spec.Profile.Params.TTLSecondsAfterFinished = new(int32(1))
					Expect(k8sClient.Update(ctx, job)).To(Succeed())
				}
				By("wait a second for new batch job creation time")
				time.Sleep(time.Second)
				By("Reconciling the created resource")
				Expect(j.reconcile(ctx, reconciler, typeNamespacedName)).To(Succeed())
				By("making sure the new batch Job is created")
				var batchJob2UID types.UID
				{
					job, err := Get[*pixivnetv1.Job](ctx, k8sClient, typeNamespacedName)
					Expect(err).To(Succeed())
					batchJobs, err := ListBatchJobsFromPixivNetJob(ctx, k8sClient, job)
					Expect(err).To(Succeed())
					Expect(batchJobs.Items).Should(HaveLen(2))
					idx := slices.IndexFunc(batchJobs.Items, func(x batchv1.Job) bool {
						return x.GetUID() != batchJob1UID
					})
					Expect(idx >= 0).To(BeTrue())
					batchJob := batchJobs.Items[idx]
					batchJob2UID = batchJob.GetUID()
					By("set the new batch Job status complete")
					metaNow := metav1.NewTime(now)
					batchJob.Status = j.newBatchJobCompleteStatus(metaNow, metaNow)
					Expect(k8sClient.Status().Update(ctx, &batchJob)).To(Succeed())
				}
				By("wait a second to expire the new batch Job")
				time.Sleep(time.Second)
				By("Reconciling the created resource")
				Expect(j.reconcile(ctx, reconciler, typeNamespacedName)).To(Succeed())
				By("making sure the new batch Job is marked to delete")
				{
					job, err := Get[*pixivnetv1.Job](ctx, k8sClient, typeNamespacedName)
					Expect(err).To(Succeed())
					batchJobs, err := ListBatchJobsFromPixivNetJob(ctx, k8sClient, job)
					Expect(err).To(Succeed())
					Expect(batchJobs.Items).Should(HaveLen(2))
					idx := slices.IndexFunc(batchJobs.Items, func(x batchv1.Job) bool {
						return x.GetUID() == batchJob2UID
					})
					Expect(idx >= 0).To(BeTrue())
					batchJob := batchJobs.Items[idx]
					Expect(batchJob.DeletionTimestamp).To(BeNil())
					Expect(batchJob.GetAnnotations()["jobs.pixiv.net/ttl-expired"]).To(Equal("true")) // marked to be deleted
				}
				By("Reconciling the created resource")
				Expect(j.reconcile(ctx, reconciler, typeNamespacedName)).To(Succeed())
				By("making sure the new batch Job is deleted because TTL is expired")
				{
					job, err := Get[*pixivnetv1.Job](ctx, k8sClient, typeNamespacedName)
					Expect(err).To(Succeed())
					batchJobs, err := ListBatchJobsFromPixivNetJob(ctx, k8sClient, job)
					Expect(err).To(Succeed())
					Expect(batchJobs.Items).Should(HaveLen(2))
					idx1 := slices.IndexFunc(batchJobs.Items, func(x batchv1.Job) bool {
						return x.GetUID() == batchJob1UID
					})
					idx2 := slices.IndexFunc(batchJobs.Items, func(x batchv1.Job) bool {
						return x.GetUID() == batchJob2UID
					})
					Expect(idx1 >= 0).To(BeTrue())
					Expect(idx2 >= 0).To(BeFalse()) // deleted
					Expect(batchJobs.Items[idx1].DeletionTimestamp).To(BeNil())
				}
			})
		},
	}
}

func (j jobControllerTest) reconcileDeleteOldJobs() controllerTestContext {
	const resourceName = "job-delete-olds"
	now := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	return controllerTestContext{
		name: "When reconciling a resource with TTL",
		beforeEach: func(a *controllerTestContextArg) {
			By(fmt.Sprintf("creating the custom resource for the Kind PodProfile: %s", resourceName))
			Expect(k8sClient.Create(ctx, j.newPodProfile(a.namespace, resourceName))).To(Succeed())
			By(fmt.Sprintf("creating the custom resource for the Kind Job: %s", resourceName))
			Expect(k8sClient.Create(ctx, j.newJobWithHistoryLimit(a.namespace, resourceName, resourceName, 1))).To(Succeed())
		},
		afterEach: func(a *controllerTestContextArg) {
			job, err := Get[*pixivnetv1.Job](ctx, k8sClient, j.newNSName(a.namespace, resourceName))
			Expect(err).To(Succeed())
			By("cleanup the specific resource instance Job")
			Expect(k8sClient.Delete(ctx, job)).To(Succeed())

			podProfile, err := Get[*pixivnetv1.PodProfile](ctx, k8sClient, j.newNSName(a.namespace, resourceName))
			Expect(err).To(Succeed())
			By("cleanup the specific resource instance PodProfile")
			Expect(k8sClient.Delete(ctx, podProfile)).To(Succeed())
		},
		test: func(a *controllerTestContextArg) {
			typeNamespacedName := j.newNSName(a.namespace, resourceName)
			It("should delete old batch jobs", func() {
				reconciler := j.newReconciler()
				By("Reconciling the created resource")
				Expect(j.reconcile(ctx, reconciler, typeNamespacedName)).To(Succeed())
				By("making sure the batch Job created successfully")
				var batchJob1UID types.UID
				{
					job, err := Get[*pixivnetv1.Job](ctx, k8sClient, typeNamespacedName)
					Expect(err).To(Succeed())
					batchJobs, err := ListBatchJobsFromPixivNetJob(ctx, k8sClient, job)
					Expect(err).To(Succeed())
					Expect(batchJobs.Items).Should(HaveLen(1))
					batchJob := batchJobs.Items[0]
					batchJob1UID = batchJob.GetUID()
					Expect(batchJob.Status.Conditions).Should(BeEmpty())
					By("set the batch Job status complete")
					metaNow := metav1.NewTime(now)
					batchJob.Status = j.newBatchJobCompleteStatus(metaNow, metaNow)
					Expect(k8sClient.Status().Update(ctx, &batchJob)).To(Succeed())
				}
				By("set the Job patches empty")
				{
					job, err := Get[*pixivnetv1.Job](ctx, k8sClient, typeNamespacedName)
					Expect(err).To(Succeed())
					job.Spec.Profile.Patches = nil
					Expect(k8sClient.Update(ctx, job)).To(Succeed())
				}
				By("wait for 1 second to create a gap in job creation time")
				time.Sleep(time.Second)
				By("Reconciling the created resource")
				Expect(j.reconcile(ctx, reconciler, typeNamespacedName)).To(Succeed())
				By("making sure the old batch job and the new batch job")
				var batchJob2UID types.UID
				{
					job, err := Get[*pixivnetv1.Job](ctx, k8sClient, typeNamespacedName)
					Expect(err).To(Succeed())
					batchJobs, err := ListBatchJobsFromPixivNetJob(ctx, k8sClient, job)
					Expect(err).To(Succeed())
					Expect(batchJobs.Items).Should(HaveLen(2))
					idx := slices.IndexFunc(batchJobs.Items, func(x batchv1.Job) bool {
						return x.GetUID() != batchJob1UID
					})
					Expect(idx >= 0).Should(BeTrue())
					batchJob := batchJobs.Items[idx]
					batchJob2UID = batchJob.GetUID()
					By("set the batch Job status complete")
					metaNow := metav1.NewTime(now)
					batchJob.Status = j.newBatchJobCompleteStatus(metaNow, metaNow)
					Expect(k8sClient.Status().Update(ctx, &batchJob)).To(Succeed())
				}
				By("update the Job patches")
				{
					job, err := Get[*pixivnetv1.Job](ctx, k8sClient, typeNamespacedName)
					Expect(err).To(Succeed())
					job.Spec.Profile.Patches = j.newJobWithHistoryLimit(a.namespace, resourceName, resourceName, 1).Spec.Profile.Patches
					job.Spec.Profile.Patches[0].Value = apiextensionsv1.JSON{
						Raw: []byte(`"debian:bookworm"`),
					}
					Expect(k8sClient.Update(ctx, job)).To(Succeed())
				}
				By("wait for 1 second to create a gap in job creation time")
				time.Sleep(time.Second)
				By("Reconciling the created resource")
				Expect(j.reconcile(ctx, reconciler, typeNamespacedName)).To(Succeed())
				By("making sure the oldest job is deleted")
				job, err := Get[*pixivnetv1.Job](ctx, k8sClient, typeNamespacedName)
				Expect(err).To(Succeed())
				batchJobs, err := ListBatchJobsFromPixivNetJob(ctx, k8sClient, job)
				Expect(err).To(Succeed())
				Expect(batchJobs.Items).Should(HaveLen(2))
				items := []batchv1.Job{}
				for _, x := range batchJobs.Items {
					// Exclude batch jobs that the controller is deleting.
					if x.DeletionTimestamp == nil {
						items = append(items, x)
					}
				}
				Expect(items).Should(HaveLen(2))
				batchJob2Idx := slices.IndexFunc(items, func(x batchv1.Job) bool {
					return x.GetUID() == batchJob2UID
				})
				Expect(batchJob2Idx >= 0).Should(BeTrue())
				batchJob3Idx := slices.IndexFunc(items, func(x batchv1.Job) bool {
					return !slices.Contains([]types.UID{batchJob1UID, batchJob2UID}, x.GetUID())
				})
				Expect(batchJob3Idx >= 0).Should(BeTrue())
			})
		},
	}
}

type jobControllerTestRecreateTestcase struct {
	title      string
	job        func(namespace, resourceName string) *pixivnetv1.Job        // For changing the job; no change if nil.
	podprofile func(namespace, resourceName string) *pixivnetv1.PodProfile // For changing the profile; no change if nil.
	status     *batchv1.JobStatus                                          // For changing the status; no change if nil.
	recreated  bool                                                        // Set to true if regeneration of the batch job is expected.
	assertJob  func(*batchv1.Job)                                          // Assertion for the updated job; does nothing if nil.
}

func (j jobControllerTest) reconcileRecreateJobs() []controllerTestContext {
	var (
		now        = metav1.NewTime(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC))
		patchedJob = func(namespace, resourceName string) *pixivnetv1.Job {
			x := j.newJob(namespace, resourceName, resourceName)
			x.Spec.Profile.Params.ActiveDeadlineSeconds = new(int64(60))
			return x
		}
		assertPatchedJob = func(x *batchv1.Job) {
			v := x.Spec.ActiveDeadlineSeconds
			if Expect(v).ShouldNot(BeNil()) {
				Expect(*v).Should(BeEquivalentTo(60))
			}
		}
		patchedPodProfile = func(namespace, resourceName string) *pixivnetv1.PodProfile {
			x := j.newPodProfile(namespace, resourceName)
			x.Spec.Template.Spec.Containers[0].Name = "patched"
			return x
		}
		assertPatchedPodProfile = func(x *batchv1.Job) {
			Expect(x.Spec.Template.Spec.Containers[0].Name).Should(Equal("patched"))
		}
	)
	testcases := []jobControllerTestRecreateTestcase{
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
			status:    new(j.newBatchJobFailedStatus(now)),
			recreated: true,
			assertJob: assertPatchedJob,
		},
		{
			title:     "batch Job is not recreated because it is failed but no update",
			status:    new(j.newBatchJobFailedStatus(now)),
			recreated: false,
		},
		{
			title:      "batch Job is recreated because it is completed and PodProfile is updated",
			podprofile: patchedPodProfile,
			status:     new(j.newBatchJobCompleteStatus(now, now)),
			recreated:  true,
			assertJob:  assertPatchedPodProfile,
		},
		{
			title:     "batch Job is not recreated because it is completed but no update",
			status:    new(j.newBatchJobCompleteStatus(now, now)),
			recreated: false,
		},
		{
			title:     "batch Job is recreated because it is completed and Job is updated",
			job:       patchedJob,
			status:    new(j.newBatchJobCompleteStatus(now, now)),
			recreated: true,
			assertJob: assertPatchedJob,
		},
	}

	contexts := make([]controllerTestContext, len(testcases))
	for i, tc := range testcases {
		contexts[i] = j.newRecreateTestContext(&tc)
	}
	return contexts
}

func (j *jobControllerTest) newRecreateTestContext(tc *jobControllerTestRecreateTestcase) controllerTestContext {
	const resourceName = "job-recreate"
	return controllerTestContext{
		name: "When reconciling a resource with job recreation/" + tc.title,
		beforeEach: func(a *controllerTestContextArg) {
			By(fmt.Sprintf("creating the custom resource for the Kind PodProfile: %s", resourceName))
			Expect(k8sClient.Create(ctx, j.newPodProfile(a.namespace, resourceName))).To(Succeed())
			By(fmt.Sprintf("creating the custom resource for the Kind Job: %s", resourceName))
			Expect(k8sClient.Create(ctx, j.newJob(a.namespace, resourceName, resourceName))).To(Succeed())
		},
		afterEach: func(a *controllerTestContextArg) {
			job, err := Get[*pixivnetv1.Job](ctx, k8sClient, j.newNSName(a.namespace, resourceName))
			Expect(err).To(Succeed())
			By("cleanup the specific resource instance Job")
			Expect(k8sClient.Delete(ctx, job)).To(Succeed())

			podProfile, err := Get[*pixivnetv1.PodProfile](ctx, k8sClient, j.newNSName(a.namespace, resourceName))
			Expect(err).To(Succeed())
			By("cleanup the specific resource instance PodProfile")
			Expect(k8sClient.Delete(ctx, podProfile)).To(Succeed())
		},
		test: func(a *controllerTestContextArg) {
			typeNamespacedName := j.newNSName(a.namespace, resourceName)
			It(tc.title, func() {
				reconciler := j.newReconciler()
				Expect(j.reconcile(ctx, reconciler, typeNamespacedName)).To(Succeed())

				By("making sure the batch Job created")
				var (
					uid                     types.UID // The id to identify the first batch Job
					batchTypeNamespacedName types.NamespacedName
				)
				{
					job, err := Get[*pixivnetv1.Job](ctx, k8sClient, typeNamespacedName)
					Expect(err).To(Succeed())
					batchJobs, err := ListBatchJobsFromPixivNetJob(ctx, k8sClient, job)
					Expect(err).To(Succeed())
					Expect(batchJobs.Items).Should(HaveLen(1))
					batchJob := batchJobs.Items[0]
					Expect(batchJob.Status.Conditions).Should(BeEmpty())
					uid = batchJob.GetUID()
					batchTypeNamespacedName = types.NamespacedName{
						Namespace: a.namespace,
						Name:      batchJob.Name,
					}
				}

				if job := tc.job; job != nil {
					By("change the Job")
					current, err := Get[*pixivnetv1.Job](ctx, k8sClient, typeNamespacedName)
					Expect(err).To(Succeed())
					x := job(a.namespace, resourceName)
					x.ResourceVersion = current.ResourceVersion
					Expect(k8sClient.Update(ctx, x)).To(Succeed())
				}
				if podprofile := tc.podprofile; podprofile != nil {
					By("change the PodProfile")
					current, err := Get[*pixivnetv1.PodProfile](ctx, k8sClient, typeNamespacedName)
					Expect(err).To(Succeed())
					x := podprofile(a.namespace, resourceName)
					x.ResourceVersion = current.ResourceVersion
					Expect(k8sClient.Update(ctx, x)).To(Succeed())
				}
				if status := tc.status; status != nil {
					By("change the batch Job status")
					batchJob, err := Get[*batchv1.Job](ctx, k8sClient, batchTypeNamespacedName)
					Expect(err).To(Succeed())
					batchJob.Status = *tc.status
					Expect(k8sClient.Status().Update(ctx, batchJob)).To(Succeed())
				}

				By("reconcile")
				// A name collision can sometimes occur when creating a batch Job.
				// This is avoided because re-running Reconcile() changes the seed for generation.
				Eventually(func() error {
					return j.reconcile(ctx, reconciler, typeNamespacedName)
				}).Should(Succeed())

				By("check the batch Job")
				job, err := Get[*pixivnetv1.Job](ctx, k8sClient, typeNamespacedName)
				Expect(err).To(Succeed())
				batchJobs, err := ListBatchJobsFromPixivNetJob(ctx, k8sClient, job)
				Expect(err).To(Succeed())

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
				j.assertJobStatus(a.namespace, resourceName, pixivnetv1.JobAvailable, metav1.ConditionTrue, "OK")
				j.assertJobStatus(a.namespace, resourceName, pixivnetv1.JobDegraded, metav1.ConditionFalse, "OK")
			})
		},
	}
}
