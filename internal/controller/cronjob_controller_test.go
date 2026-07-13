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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pixivnetv1 "github.com/pixiv/k8s-job-wrapper/api/v1"
	"github.com/pixiv/k8s-job-wrapper/internal/kubectl"
	"github.com/pixiv/k8s-job-wrapper/internal/kustomize"
	"github.com/pixiv/k8s-job-wrapper/test/utils"
)

var _ = new(cronJobControllerTest).run()

type cronJobControllerTest struct {
	controllerTestBase
}

func (c cronJobControllerTest) run() bool {
	return controllerTest{
		name:            "CronJobController",
		namespacePrefix: "cronjob",
		contexts: []controllerTestContext{
			c.reconcilePodProfileMissing(),
			c.reconcileNormalTest(),
		},
	}.run()
}

func (cronJobControllerTest) newReconciler() *CronJobReconciler {
	return &CronJobReconciler{
		Client:  k8sClient,
		Scheme:  k8sClient.Scheme(),
		Patcher: kustomize.NewPatchRunner(kubectl.NewCommand(utils.Kubectl())),
	}
}

func (c cronJobControllerTest) assertCronJobStatus(
	namespace, resourceName string,
	key pixivnetv1.CronJobConditionType,
	status metav1.ConditionStatus,
	reason string,
	message ...string,
) {
	GinkgoHelper()
	cronJob, err := Get[*pixivnetv1.CronJob](ctx, k8sClient, c.newNSName(namespace, resourceName))
	Expect(err).To(Succeed())
	c.assertStatus(cronJob.Status.Conditions, string(key), status, reason, message...)
}

func (c cronJobControllerTest) reconcilePodProfileMissing() controllerTestContext {
	const resourceName = "podprofile-missing"
	return controllerTestContext{
		name: "When reconciling a resource without PodProfile",
		beforeEach: func(a *controllerTestContextArg) {
			By(fmt.Sprintf("creating the custom resource for the Kind CronJob: %s", resourceName))
			Expect(k8sClient.Create(ctx, c.newCronJob(a.namespace, resourceName, resourceName))).To(Succeed())
		},
		afterEach: func(a *controllerTestContextArg) {
			cronJob, err := Get[*pixivnetv1.CronJob](ctx, k8sClient, c.newNSName(a.namespace, resourceName))
			Expect(err).To(Succeed())
			By("cleanup the specific resource instance CronJob")
			Expect(k8sClient.Delete(ctx, cronJob)).To(Succeed())
		},
		test: func(a *controllerTestContextArg) {
			typeNamespacedName := c.newNSName(a.namespace, resourceName)
			It("should failed to reconcile the resource because the PodProfile is missing", func() {
				reconciler := c.newReconciler()
				By("reconciling")
				Expect(c.reconcile(ctx, reconciler, typeNamespacedName)).Should(HaveOccurred())
				By("making sure the Status")
				c.assertCronJobStatus(a.namespace, resourceName, pixivnetv1.CronJobAvailable, metav1.ConditionFalse, "Reconciling", "PodProfile not found")
				c.assertCronJobStatus(a.namespace, resourceName, pixivnetv1.CronJobDegraded, metav1.ConditionTrue, "Reconciling")
				By("making sure no batch CronJobs are generated")
				cronJob, err := Get[*pixivnetv1.CronJob](ctx, k8sClient, c.newNSName(a.namespace, resourceName))
				Expect(err).To(Succeed())
				_, err = GetBatchCronJobFromPixivNetCronJob(ctx, k8sClient, cronJob)
				Expect(apierrors.IsNotFound(err)).To(BeTrue())
			})
		},
	}
}

func (c cronJobControllerTest) reconcileNormalTest() controllerTestContext {
	const resourceName = "cronjob-reconcile"
	return controllerTestContext{
		name: "When reconciling a resource",
		beforeEach: func(a *controllerTestContextArg) {
			By(fmt.Sprintf("creating the custom resource for the Kind PodProfile %s", resourceName))
			Expect(k8sClient.Create(ctx, c.newPodProfile(a.namespace, resourceName))).To(Succeed())
			By(fmt.Sprintf("creating the custom resource for the Kind CronJob: %s", resourceName))
			Expect(k8sClient.Create(ctx, c.newCronJob(a.namespace, resourceName, resourceName))).To(Succeed())
		},
		afterEach: func(a *controllerTestContextArg) {
			cronJob, err := Get[*pixivnetv1.CronJob](ctx, k8sClient, c.newNSName(a.namespace, resourceName))
			Expect(err).To(Succeed())
			By("cleanup the specific resource instance CronJob")
			Expect(k8sClient.Delete(ctx, cronJob)).To(Succeed())

			podProfile, err := Get[*pixivnetv1.PodProfile](ctx, k8sClient, c.newNSName(a.namespace, resourceName))
			Expect(err).To(Succeed())
			By("cleanup the specific resource instance PodProfile")
			Expect(k8sClient.Delete(ctx, podProfile)).To(Succeed())
		},
		test: func(a *controllerTestContextArg) {
			typeNamespacedName := c.newNSName(a.namespace, resourceName)

			It("should successfully reconcile the resource", func() {
				reconciler := c.newReconciler()
				By("Reconciling the created resource")
				Expect(c.reconcile(ctx, reconciler, typeNamespacedName)).To(Succeed())

				By("making sure the batch CronJob created successfully")
				cronJob, err := Get[*pixivnetv1.CronJob](ctx, k8sClient, typeNamespacedName)
				Expect(err).To(Succeed())
				batchCronJob, err := GetBatchCronJobFromPixivNetCronJob(ctx, k8sClient, cronJob)
				Expect(err).To(Succeed())
				Expect(batchCronJob.Spec.Schedule).To(Equal("* * * * *"))
				Expect(batchCronJob.Spec.JobTemplate.Spec.Suspend).Should(Equal(new(true)))
				Expect(batchCronJob.Spec.JobTemplate.Spec.Template.Spec.Containers).Should(HaveLen(1))
				Expect(batchCronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Name).Should(Equal("pi"))
				Expect(batchCronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image).Should(Equal("debian:bookworm-slim"))
				Expect(batchCronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Command).Should(Equal([]string{
					"perl", "-Mbignum=bpi", "-wle", "print bpi(2000)",
				}))

				By("making sure the Status updated successfully")
				c.assertCronJobStatus(a.namespace, resourceName, pixivnetv1.CronJobAvailable, metav1.ConditionTrue, "OK")
				c.assertCronJobStatus(a.namespace, resourceName, pixivnetv1.CronJobDegraded, metav1.ConditionFalse, "OK")

				By("update the CronJob")
				{
					cronJob, err := Get[*pixivnetv1.CronJob](ctx, k8sClient, typeNamespacedName)
					Expect(err).To(Succeed())
					cronJob.Spec.Profile.Patches[0].Value.Raw = []byte(`"nginx:latest"`)
					Expect(k8sClient.Update(ctx, cronJob)).To(Succeed())
				}

				By("reconcling the updated resource")
				Expect(c.reconcile(ctx, reconciler, typeNamespacedName)).To(Succeed())

				By("making sure the batch CronJob updated successfully")
				cronJob, err = Get[*pixivnetv1.CronJob](ctx, k8sClient, typeNamespacedName)
				Expect(err).To(Succeed())
				batchCronJob, err = GetBatchCronJobFromPixivNetCronJob(ctx, k8sClient, cronJob)
				Expect(err).To(Succeed())
				Expect(batchCronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image).Should(Equal("nginx:latest"))

				By("making sure the Status keeps OK")
				c.assertCronJobStatus(a.namespace, resourceName, pixivnetv1.CronJobAvailable, metav1.ConditionTrue, "OK")
				c.assertCronJobStatus(a.namespace, resourceName, pixivnetv1.CronJobDegraded, metav1.ConditionFalse, "OK")

				By("update the PodProfile")
				{
					podProfile, err := Get[*pixivnetv1.PodProfile](ctx, k8sClient, c.newNSName(a.namespace, resourceName))
					Expect(err).To(Succeed())
					podProfile.Spec.Template.Spec.Containers[0].Command = []string{"sleep", "10"}
					Expect(k8sClient.Update(ctx, podProfile)).To(Succeed())
				}

				By("reconcling the updated resource")
				Expect(c.reconcile(ctx, reconciler, typeNamespacedName)).To(Succeed())

				By("making sure the batch CronJob updated successfully")
				cronJob, err = Get[*pixivnetv1.CronJob](ctx, k8sClient, typeNamespacedName)
				Expect(err).To(Succeed())
				batchCronJob, err = GetBatchCronJobFromPixivNetCronJob(ctx, k8sClient, cronJob)
				Expect(err).To(Succeed())
				Expect(batchCronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Command).Should(Equal([]string{"sleep", "10"}))

				By("making sure the Status keeps OK")
				c.assertCronJobStatus(a.namespace, resourceName, pixivnetv1.CronJobAvailable, metav1.ConditionTrue, "OK")
				c.assertCronJobStatus(a.namespace, resourceName, pixivnetv1.CronJobDegraded, metav1.ConditionFalse, "OK")
			})
		},
	}
}
