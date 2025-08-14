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
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pixivnetv1 "github.com/pixiv/k8s-job-wrapper/api/v1"
	"github.com/pixiv/k8s-job-wrapper/internal/kubectl"
	"github.com/pixiv/k8s-job-wrapper/internal/kustomize"
)

var _ = Describe("CronJob Controller", func() {
	Context("When reconciling a resource", func() {
		reconcile := func(resourceName string) error {
			controllerReconciler := &CronJobReconciler{
				Client:  k8sClient,
				Scheme:  k8sClient.Scheme(),
				Patcher: kustomize.NewPatchRunner(kubectl.NewCommand(os.Getenv("KUBECTL"))),
			}
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: newKey(resourceName),
			})
			return err
		}

		newCronJob := func(resourceName string) *pixivnetv1.CronJob {
			return &pixivnetv1.CronJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: testNamespace,
				},
				Spec: pixivnetv1.CronJobSpec{
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
					Schedule: "* * * * *",
				},
			}
		}

		batchCronJobName := func(resourceName string) string {
			return resourceName + "-pxvcjob"
		}

		getBatchCronJob := func(resourceName string) *batchv1.CronJob {
			batchCronJob := &batchv1.CronJob{}
			Eventually(func() error {
				return k8sClient.Get(ctx, newKey(batchCronJobName(resourceName)), batchCronJob)
			}).Should(Succeed())
			return batchCronJob
		}

		getCronJob := func(resourceName string) *pixivnetv1.CronJob {
			cronJob := &pixivnetv1.CronJob{}
			Eventually(func() error {
				return k8sClient.Get(ctx, newKey(resourceName), cronJob)
			}).Should(Succeed())
			return cronJob
		}

		assertCronJobStatus := func(resourceName string, key pixivnetv1.CronJobConditionType, status metav1.ConditionStatus, reason string, message ...string) {
			cronJob := getCronJob(resourceName)
			assertStatus(cronJob.Status.Conditions, string(key), status, reason, message...)
		}

		beforeEach := func(resourceName string) {
			By(fmt.Sprintf("creating the custom resource for the Kind PodProfile %s", resourceName))
			Expect(k8sClient.Create(ctx, newPodProfile(resourceName))).To(Succeed())
			By(fmt.Sprintf("creating the custom resource for the Kind CronJob: %s", resourceName))
			Expect(k8sClient.Create(ctx, newCronJob(resourceName))).To(Succeed())
		}

		It("should failed to reconcile the resource because the PodProfile is missing", func() {
			const resourceName = "cronjob-missing"
			By(fmt.Sprintf("creating the custom resource for the Kind CronJob: %s", resourceName))
			Expect(k8sClient.Create(ctx, newCronJob(resourceName))).To(Succeed())
			By("reconciling")
			Expect(reconcile(resourceName)).Should(HaveOccurred())
			By("making sure the Status")
			assertCronJobStatus(resourceName, pixivnetv1.CronJobAvailable, metav1.ConditionFalse, "Reconciling", "PodProfile not found")
			assertCronJobStatus(resourceName, pixivnetv1.CronJobDegraded, metav1.ConditionTrue, "Reconciling")
			By("making sure no batch CronJobs are generated")
			Expect(apierrors.IsNotFound(k8sClient.Get(ctx, newKey(batchCronJobName(resourceName)), &batchv1.CronJob{}))).To(BeTrue())
		})

		It("should successfully reconcile the resource", func() {
			const resourceName = "cronjob-reconcile"
			beforeEach(resourceName)

			By("Reconciling the created resource")
			Expect(reconcile(resourceName)).To(Succeed())

			By("making sure the batch CronJob created successfully")
			batchCronJob := getBatchCronJob(resourceName)
			Expect(batchCronJob.Spec.Schedule).To(Equal("* * * * *"))
			Expect(batchCronJob.Spec.JobTemplate.Spec.Suspend).Should(Equal(ptr.To(true)))
			Expect(batchCronJob.Spec.JobTemplate.Spec.Template.Spec.Containers).Should(HaveLen(1))
			Expect(batchCronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Name).Should(Equal("pi"))
			Expect(batchCronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image).Should(Equal("debian:bookworm-slim"))
			Expect(batchCronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Command).Should(Equal([]string{
				"perl", "-Mbignum=bpi", "-wle", "print bpi(2000)",
			}))

			By("making sure the Status updated successfully")
			assertCronJobStatus(resourceName, pixivnetv1.CronJobAvailable, metav1.ConditionTrue, "OK")
			assertCronJobStatus(resourceName, pixivnetv1.CronJobDegraded, metav1.ConditionFalse, "OK")

			By("update the CronJob")
			{
				cronJob := getCronJob(resourceName)
				cronJob.Spec.Profile.Patches[0].Value.Raw = []byte(`"nginx:latest"`)
				Expect(k8sClient.Update(ctx, cronJob)).To(Succeed())
			}

			By("reconcling the updated resource")
			Expect(reconcile(resourceName)).To(Succeed())

			By("making sure the batch CronJob updated successfully")
			batchCronJob = getBatchCronJob(resourceName)
			Expect(batchCronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image).Should(Equal("nginx:latest"))

			By("making sure the Status keeps OK")
			assertCronJobStatus(resourceName, pixivnetv1.CronJobAvailable, metav1.ConditionTrue, "OK")
			assertCronJobStatus(resourceName, pixivnetv1.CronJobDegraded, metav1.ConditionFalse, "OK")

			By("update the PodProfile")
			{
				podProfile := getPodProfile(resourceName)
				podProfile.Spec.Template.Spec.Containers[0].Command = []string{"sleep", "10"}
				Expect(k8sClient.Update(ctx, podProfile)).To(Succeed())
			}

			By("reconcling the updated resource")
			Expect(reconcile(resourceName)).To(Succeed())

			By("making sure the batch CronJob updated successfully")
			batchCronJob = getBatchCronJob(resourceName)
			Expect(batchCronJob.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Command).Should(Equal([]string{"sleep", "10"}))

			By("making sure the Status keeps OK")
			assertCronJobStatus(resourceName, pixivnetv1.CronJobAvailable, metav1.ConditionTrue, "OK")
			assertCronJobStatus(resourceName, pixivnetv1.CronJobDegraded, metav1.ConditionFalse, "OK")
		})

	})
})
