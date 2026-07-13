/*
Copyright 2026 pixiv Inc.

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

package e2e

import (
	"fmt"
	"slices"

	"github.com/pixiv/k8s-job-wrapper/test/utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type e2eTest struct{}

func (e e2eTest) run() bool {
	testcases := e.reconcileSuccessfully()
	testcases = append(testcases, e.reconcileChanges()...)
	return utils.Suite{
		Name:            "E2ETest",
		NamespacePrefix: "e2e",
		KustomizeRoot:   "test/manifests",
		Testcases:       testcases,
	}.Run()
}

func (e2eTest) reconcileChanges() []utils.Testcase {
	const (
		podProfileName = "changes-podprofile-sample"
		jobName        = "changes-job-sample"
		cronJobName    = "changes-cronjob-sample"
	)
	var (
		ensureResources = func(namespace string) {
			GinkgoHelper()
			utils.EnsurePodProfile(namespace, podProfileName)
			utils.EnsureJob(namespace, jobName)
			utils.EnsureCronJob(namespace, cronJobName)
		}

		excludeBatchJobNames []string
		getBatchJobName      = func(namespace string) string {
			GinkgoHelper()
			By(fmt.Sprintf("get batch job from %s", jobName))
			var name string
			Eventually(func() error {
				By(fmt.Sprintf("list batch jobs %s", jobName))
				names, err := utils.ListBatchJobs(namespace, jobName)
				if err != nil {
					return err
				}
				names = slices.DeleteFunc(names, func(x string) bool {
					return slices.Contains(excludeBatchJobNames, x)
				})
				if len(names) != 1 {
					return fmt.Errorf("expect %s has just 1 job but %d", jobName, len(names))
				}
				name = names[0]
				return nil
			}).Should(Succeed())
			return name
		}
	)

	return []utils.Testcase{
		{
			Name:         "reconcileChanges",
			KustomizeDir: "changes",
			Steps: []utils.Step{
				{
					Name:         "before",
					KustomizeDir: "overlays/before",
					Assert: func(a *utils.StepArg) {
						ensureResources(a.Namespace)
						By("ensure cronjob schedule")
						utils.EnsureResourceValue(a.Namespace, utils.CronJobResource, cronJobName, "{.spec.schedule}", "* * * * *")
						By("ensure batch cronjob schedule")
						utils.EnsureResourceValue(a.Namespace, "cronjob", utils.BatchCronJobName(cronJobName),
							"{.spec.schedule}", "* * * * *")
						By("ensure job activeDealineSeconds")
						utils.EnsureResourceValue(a.Namespace, utils.JobResource, jobName,
							"{.spec.profile.jobParams.activeDeadlineSeconds}", "120")
						By("ensure batch job activeDeadlineSeconds")
						{
							name := getBatchJobName(a.Namespace)
							utils.EnsureResourceValue(a.Namespace, "job", name, "{.spec.activeDeadlineSeconds}", "120")
						}
						By("ensure podprofile image")
						utils.EnsureResourceValue(a.Namespace, utils.PodProfileResource, podProfileName,
							"{.spec.template.spec.containers[0].image}", "perl:5.34.0")
						By("ensure batch cronjob container image")
						utils.EnsureResourceValue(a.Namespace, "cronjob", utils.BatchCronJobName(cronJobName),
							"{.spec.jobTemplate.spec.template.spec.containers[0].image}", "perl:5.34.0")
						By("ensure batch job container image")
						{
							name := getBatchJobName(a.Namespace)
							utils.EnsureResourceValue(a.Namespace, "job", name, "{.spec.template.spec.containers[0].image}", "perl:5.34.0")
							excludeBatchJobNames = append(excludeBatchJobNames, name)
						}
					},
				},
				{
					Name:         "cronjob",
					KustomizeDir: "overlays/cronjob",
					Assert: func(a *utils.StepArg) {
						ensureResources(a.Namespace)
						By("ensure cronjob schedule change")
						utils.EnsureResourceValue(a.Namespace, utils.CronJobResource, cronJobName, "{.spec.schedule}", "0 0 31 2 *")
						By("ensure batch cronjob schedule change")
						utils.EnsureResourceValue(a.Namespace, "cronjob", utils.BatchCronJobName(cronJobName),
							"{.spec.schedule}", "0 0 31 2 *")
					},
				},
				{
					Name:         "job",
					KustomizeDir: "overlays/job",
					Assert: func(a *utils.StepArg) {
						ensureResources(a.Namespace)
						By("ensure job activeDeadlineSeconds change")
						utils.EnsureResourceValue(a.Namespace, utils.JobResource, jobName,
							"{.spec.profile.jobParams.activeDeadlineSeconds}", "1200")
						By("ensure batch job activeDeadlineSeconds change")
						{
							name := getBatchJobName(a.Namespace)
							utils.EnsureResourceValue(a.Namespace, "job", name, "{.spec.activeDeadlineSeconds}", "1200")
							excludeBatchJobNames = append(excludeBatchJobNames, name)
						}
					},
				},
				{
					Name:         "podprofile",
					KustomizeDir: "overlays/podprofile",
					Assert: func(a *utils.StepArg) {
						ensureResources(a.Namespace)
						By("ensure podprofile container image change")
						utils.EnsureResourceValue(a.Namespace, utils.PodProfileResource, podProfileName,
							"{.spec.template.spec.containers[0].image}", "perl:5.42.0")
						By("ensure batch cronjob container image change")
						utils.EnsureResourceValue(a.Namespace, "cronjob", utils.BatchCronJobName(cronJobName),
							"{.spec.jobTemplate.spec.template.spec.containers[0].image}", "perl:5.42.0")
						By("ensure batch job container image change")
						{
							name := getBatchJobName(a.Namespace)
							utils.EnsureResourceValue(a.Namespace, "job", name, "{.spec.template.spec.containers[0].image}", "perl:5.42.0")
							excludeBatchJobNames = append(excludeBatchJobNames, name)
						}
					},
				},
			},
		},
	}
}

func (e2eTest) reconcileSuccessfully() []utils.Testcase {
	return []utils.Testcase{
		{
			Name:         "pod-metadata",
			KustomizeDir: "pod-metadata",
			Steps: []utils.Step{
				{
					Name: "reconcile",
					Assert: func(a *utils.StepArg) {
						utils.EnsurePodProfile(a.Namespace, "podprofile-pod-metadata")
						utils.EnsureJob(a.Namespace, "job-pod-metadata")
						utils.EnsureBatchJobCompleted(a.Namespace, "job-pod-metadata")

						name := utils.EnsureOnlyOneBatchJobCreated(a.Namespace, "job-pod-metadata")
						output, err := utils.Run(utils.KubectlCmd("-n", a.Namespace,
							"get", "job", name, "-o", "jsonpath={.spec.template.spec.containers[0].command}",
						))
						Expect(err).To(Succeed())
						Expect(output).To(Equal(`["perl","-Mbignum=bpi","-wle","print bpi(11)"]`))

						pod := utils.EnsureOnlyOneBatchJobManagedPodCreated(a.Namespace, name)
						for _, check := range []struct {
							path string
							want string
						}{
							{
								path: `{.metadata.labels.app}`,
								want: "pod-metadata",
							},
							{
								path: `{.metadata.annotations.desc}`,
								want: "add pod-metadata",
							},
						} {
							output, err := utils.Run(utils.KubectlCmd("-n", a.Namespace,
								"get", "pod", pod, "-o", "jsonpath="+check.path,
							))
							Expect(err).To(Succeed())
							Expect(output).To(Equal(check.want))
						}
					},
				},
			},
		},
		{
			Name:         "sample",
			KustomizeDir: "sample",
			Steps: []utils.Step{
				{
					Name: "reconcile",
					Assert: func(a *utils.StepArg) {
						utils.EnsurePodProfile(a.Namespace, "podprofile-sample")
						utils.EnsureJob(a.Namespace, "job-sample")
						utils.EnsureBatchJobCompleted(a.Namespace, "job-sample")

						name := utils.EnsureOnlyOneBatchJobCreated(a.Namespace, "job-sample")
						output, err := utils.Run(utils.KubectlCmd("-n", a.Namespace,
							"get", "job", name, "-o", "jsonpath={.spec.template.spec.containers[0].command}",
						))
						Expect(err).To(Succeed())
						Expect(output).To(Equal(`["perl","-Mbignum=bpi","-wle","print bpi(100)"]`))
					},
				},
			},
		},
		{
			Name:         "sample-cronjob",
			KustomizeDir: "sample",
			Steps: []utils.Step{
				{
					Name: "reconcile",
					Assert: func(a *utils.StepArg) {
						utils.EnsurePodProfile(a.Namespace, "podprofile-sample")
						utils.EnsureJob(a.Namespace, "job-sample")
						utils.EnsureBatchJobCompleted(a.Namespace, "job-sample")

						utils.EnsureCronJob(a.Namespace, "cronjob-sample")
						utils.EnsureBatchCronJob(a.Namespace, "cronjob-sample")
						utils.EnsureBatchCronJobWorking(a.Namespace, "cronjob-sample")
					},
				},
			},
		},
		{
			Name:         "simple",
			KustomizeDir: "simple",
			Steps: []utils.Step{
				{
					Name: "reconcile",
					Assert: func(a *utils.StepArg) {
						utils.EnsurePodProfile(a.Namespace, "podprofile-simple")
						utils.EnsureJob(a.Namespace, "job-simple")
						utils.EnsureBatchJobCompleted(a.Namespace, "job-simple")

						name := utils.EnsureOnlyOneBatchJobCreated(a.Namespace, "job-simple")
						output, err := utils.Run(utils.KubectlCmd("-n", a.Namespace,
							"get", "job", name, "-o", "jsonpath={.spec.template.spec.containers[0].name}",
						))
						Expect(err).To(Succeed())
						Expect(output).To(Equal(`simple`))
					},
				},
			},
		},
		{
			Name:         "complex",
			KustomizeDir: "complex",
			Steps: []utils.Step{
				{
					Name: "reconcile",
					Assert: func(a *utils.StepArg) {
						utils.EnsurePodProfile(a.Namespace, "podprofile-complex")
						utils.EnsureJob(a.Namespace, "job-complex")
						utils.EnsureBatchJobCompleted(a.Namespace, "job-complex")

						name := utils.EnsureOnlyOneBatchJobCreated(a.Namespace, "job-complex")
						output, err := utils.Run(utils.KubectlCmd("-n", a.Namespace,
							"get", "job", name, "-o", "jsonpath={.spec.template.spec.containers[*].name}",
						))
						Expect(err).To(Succeed())
						Expect(output).To(Equal(`pi complex`))
						output, err = utils.Run(utils.KubectlCmd("-n", a.Namespace,
							"get", "job", name, "-o", "jsonpath={.spec.template.spec.containers[1].command}",
						))
						Expect(err).To(Succeed())
						Expect(output).To(Equal(`["perl","-Mbignum=bpi","-wle","print bpi(10)"]`))
					},
				},
			},
		},
	}
}
