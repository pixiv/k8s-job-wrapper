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

package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pixiv/k8s-job-wrapper/test/utils"
)

// namespace where the project is deployed in
const namespace = "k8s-job-wrapper-system"

// serviceAccountName created for the project
const serviceAccountName = "k8s-job-wrapper-controller-manager"

// metricsServiceName is the name of the metrics service of the project
const metricsServiceName = "k8s-job-wrapper-controller-manager-metrics-service"

// metricsRoleBindingName is the name of the RBAC that will be created to allow get the metrics data
const metricsRoleBindingName = "k8s-job-wrapper-metrics-binding"

var _ = Describe("Manager", Ordered, func() {
	var controllerPodName string

	// Before running the tests, set up the environment by creating the namespace,
	// enforce the restricted security policy to the namespace, installing CRDs,
	// and deploying the controller.
	BeforeAll(func() {
		By("creating manager namespace")
		cmd := utils.KubectlCmd("create", "ns", namespace)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create namespace")

		By("labeling the namespace to enforce the restricted security policy")
		cmd = utils.KubectlCmd("label", "--overwrite", "ns", namespace,
			"pod-security.kubernetes.io/enforce=restricted")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to label namespace with restricted policy")

		By("installing CRDs")
		cmd = exec.Command("make", "install")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to install CRDs")

		By("deploying the controller-manager")
		cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", projectImage))
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy the controller-manager")
	})

	// After all tests have been executed, clean up by undeploying the controller, uninstalling CRDs,
	// and deleting the namespace.
	AfterAll(func() {
		By("cleaning up the curl pod for metrics")
		cmd := utils.KubectlCmd("delete", "pod", "curl-metrics", "-n", namespace)
		_, _ = utils.Run(cmd)

		By("undeploying the controller-manager")
		cmd = exec.Command("make", "undeploy")
		_, _ = utils.Run(cmd)

		By("uninstalling CRDs")
		cmd = exec.Command("make", "uninstall")
		_, _ = utils.Run(cmd)

		By("removing manager namespace")
		cmd = utils.KubectlCmd("delete", "ns", namespace)
		_, _ = utils.Run(cmd)
	})

	// After each test, check for failures and collect logs, events,
	// and pod descriptions for debugging.
	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			By("Fetching controller manager pod logs")
			cmd := utils.KubectlCmd("logs", controllerPodName, "-n", namespace)
			controllerLogs, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Controller logs:\n %s", controllerLogs)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Controller logs: %s", err)
			}

			By("Fetching Kubernetes events")
			cmd = utils.KubectlCmd("get", "events", "-n", namespace, "--sort-by=.lastTimestamp")
			eventsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Kubernetes events:\n%s", eventsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Kubernetes events: %s", err)
			}

			By("Fetching curl-metrics logs")
			cmd = utils.KubectlCmd("logs", "curl-metrics", "-n", namespace)
			metricsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Metrics logs:\n %s", metricsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get curl-metrics logs: %s", err)
			}

			By("Fetching controller manager pod description")
			cmd = utils.KubectlCmd("describe", "pod", controllerPodName, "-n", namespace)
			podDescription, err := utils.Run(cmd)
			if err == nil {
				fmt.Println("Pod description:\n", podDescription)
			} else {
				fmt.Println("Failed to describe controller pod")
			}
		}
	})

	// Eventually() が待機する時間
	// これ以上時間がかかると失敗扱いになる
	// なので検証のための job はそれほど実行に時間がかからないような内容にしている
	// コンテナイメージも揃えている
	// cronjob は毎分発火にしている
	// なるべく Eventually を分割し、1回の Eventually の中で待機する処理を少なくするとよい
	SetDefaultEventuallyTimeout(2 * time.Minute)
	SetDefaultEventuallyPollingInterval(time.Second)

	Context("Manager", func() {
		It("should run successfully", func() {
			By("validating that the controller-manager pod is running as expected")
			verifyControllerUp := func(g Gomega) {
				// Get the name of the controller-manager pod
				cmd := utils.KubectlCmd("get",
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}"+
						"{{ if not .metadata.deletionTimestamp }}"+
						"{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", namespace,
				)

				podOutput, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve controller-manager pod information")
				podNames := utils.GetNonEmptyLines(podOutput)
				g.Expect(podNames).To(HaveLen(1), "expected 1 controller pod running")
				controllerPodName = podNames[0]
				g.Expect(controllerPodName).To(ContainSubstring("controller-manager"))

				// Validate the pod's status
				cmd = utils.KubectlCmd("get",
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
					"-n", namespace,
				)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Running"), "Incorrect controller-manager pod status")
			}
			Eventually(verifyControllerUp).Should(Succeed())
		})

		It("should ensure the metrics endpoint is serving metrics", func() {
			By("creating a ClusterRoleBinding for the service account to allow access to metrics")
			cmd := utils.KubectlCmd("create", "clusterrolebinding", metricsRoleBindingName,
				"--clusterrole=k8s-job-wrapper-metrics-reader",
				fmt.Sprintf("--serviceaccount=%s:%s", namespace, serviceAccountName),
			)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create ClusterRoleBinding")

			By("validating that the metrics service is available")
			cmd = utils.KubectlCmd("get", "service", metricsServiceName, "-n", namespace)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Metrics service should exist")

			By("getting the service account token")
			token, err := serviceAccountToken()
			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeEmpty())

			By("waiting for the metrics endpoint to be ready")
			verifyMetricsEndpointReady := func(g Gomega) {
				cmd := utils.KubectlCmd("get", "endpoints", metricsServiceName, "-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("8443"), "Metrics endpoint is not ready")
			}
			Eventually(verifyMetricsEndpointReady).Should(Succeed())

			By("verifying that the controller manager is serving the metrics server")
			verifyMetricsServerStarted := func(g Gomega) {
				cmd := utils.KubectlCmd("logs", controllerPodName, "-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("controller-runtime.metrics\tServing metrics server"),
					"Metrics server not yet started")
			}
			Eventually(verifyMetricsServerStarted).Should(Succeed())

			By("creating the curl-metrics pod to access the metrics endpoint")
			cmd = utils.KubectlCmd("run", "curl-metrics", "--restart=Never",
				"--namespace", namespace,
				"--image=curlimages/curl:latest",
				"--overrides",
				fmt.Sprintf(`{
					"spec": {
						"containers": [{
							"name": "curl",
							"image": "curlimages/curl:latest",
							"command": ["/bin/sh", "-c"],
							"args": ["curl -v -k -H 'Authorization: Bearer %s' https://%s.%s.svc.cluster.local:8443/metrics"],
							"securityContext": {
								"allowPrivilegeEscalation": false,
								"capabilities": {
									"drop": ["ALL"]
								},
								"runAsNonRoot": true,
								"runAsUser": 1000,
								"seccompProfile": {
									"type": "RuntimeDefault"
								}
							}
						}],
						"serviceAccount": "%s"
					}
				}`, token, metricsServiceName, namespace, serviceAccountName))
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create curl-metrics pod")

			By("waiting for the curl-metrics pod to complete.")
			verifyCurlUp := func(g Gomega) {
				cmd := utils.KubectlCmd("get", "pods", "curl-metrics",
					"-o", "jsonpath={.status.phase}",
					"-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Succeeded"), "curl pod in wrong status")
			}
			Eventually(verifyCurlUp, 5*time.Minute).Should(Succeed())

			By("getting the metrics by checking curl-metrics logs")
			metricsOutput := getMetricsOutput()
			Expect(metricsOutput).To(ContainSubstring(
				"controller_runtime_reconcile_total",
			))
		})

		// +kubebuilder:scaffold:e2e-webhooks-checks

		// TODO: Customize the e2e test suite with scenarios specific to your project.
		// Consider applying sample/CR(s) and check their status and/or verifying
		// the reconciliation by using the metrics, i.e.:
		// metricsOutput := getMetricsOutput()
		// Expect(metricsOutput).To(ContainSubstring(
		//    fmt.Sprintf(`controller_runtime_reconcile_total{controller="%s",result="success"} 1`,
		//    strings.ToLower(<Kind>),
		// ))

		// --------------------------------------------------------------------------------
		//
		// ここからリソースを作成してリソースの状態を検証するテスト
		//
		// --------------------------------------------------------------------------------
		const (
			namespace          = "default"
			podProfileResource = "podprofiles.pixiv.net"
			jobResource        = "jobs.pixiv.net"
			cronJobResource    = "cronjobs.pixiv.net"
		)
		// test/manifests 以下のディレクトリを指定して apply する
		applyManifests := func(name string) {
			location := fmt.Sprintf("test/manifests/%s", name)
			By(fmt.Sprintf("applying %s", location))
			apply := func() error {
				_, err := utils.Run(utils.KubectlCmd("-n", namespace, "apply", "-k", location))

				return err
			}
			Eventually(apply).Should(Succeed())
		}
		// PodProfile の存在確認をする
		ensurePodProfile := func(name string) {
			By(fmt.Sprintf("making ensure the PodProfile %s is created", name))
			ensurePodProfile := func() error {
				_, err := utils.Run(utils.KubectlCmd("-n", namespace, "get", podProfileResource, name))
				return err
			}
			Eventually(ensurePodProfile).Should(Succeed())
		}
		ensureCustomResourceStatus := func(name, resource string) {
			By(fmt.Sprintf("making ensure the %s %s is created", resource, name))
			ensure := func(g Gomega) {
				output, err := utils.Run(
					utils.KubectlCmd("-n", namespace, "get", resource, name, "-o", "jsonpath={.status.conditions}"),
				)
				if g.Expect(err).To(Succeed(), fmt.Sprintf("Failed to get the %s", resource)) {
					return
				}
				var conditions []metav1.Condition
				if err := json.Unmarshal([]byte(output), &conditions); g.Expect(err).To(Succeed()) {
					return
				}
				if !g.Expect(conditions).Should(HaveLen(2), fmt.Sprintf("Invalid %s conditions", resource)) {
					return
				}
				d := map[string]metav1.Condition{}
				for _, c := range conditions {
					d[c.Type] = c
				}
				available, exist := d["Available"]
				if g.Expect(exist).To(BeTrue()) {
					g.Expect(available.Status).To(Equal(metav1.ConditionTrue))
					g.Expect(available.Reason).To(Equal("OK"))
				}
				degraded, exist := d["Degraded"]
				if g.Expect(exist).To(BeTrue()) {
					g.Expect(degraded.Status).To(Equal(metav1.ConditionFalse))
					g.Expect(degraded.Reason).To(Equal("OK"))
				}
			}
			Eventually(ensure).Should(Succeed())
		}
		// Job の存在確認をする
		ensureJob := func(name string) {
			ensureCustomResourceStatus(name, jobResource)
		}
		// CronJob の存在確認をする
		ensureCronJob := func(name string) {
			ensureCustomResourceStatus(name, cronJobResource)
		}
		batchCronJobName := func(name string) string {
			return name + "-pxvcjob"
		}
		// CronJob から生み出された batch CronJob の存在確認をする
		ensureBatchCronJob := func(name string) {
			By(fmt.Sprintf("making ensure the batch CronJob %s generated by the CronJob %s", batchCronJobName(name), name))
			Eventually(func() error {
				_, err := utils.Run(
					utils.KubectlCmd("-n", namespace, "get", "cronjob", batchCronJobName(name)),
				)
				return err
			}).Should(Succeed())
		}
		// batch Job のステータスが成功であることを確認する
		ensureBatchJobStatusCompleted := func(batchJobName string) {
			By(fmt.Sprintf("making ensure the batch Job %s is completed successfully", batchJobName))
			if !Expect(batchJobName).ShouldNot(BeEmpty(), "batchJobName should not be empty") {
				return
			}
			ensureBatchJobStatusCompleted := func(g Gomega) {
				output, err := utils.Run(utils.KubectlCmd(
					"get", "-n", namespace, "job", batchJobName, "-o", "jsonpath={.status.conditions}"),
				)
				if !g.Expect(err).To(Succeed(), "Failed to get the batch Job conditions") {
					return
				}
				var conditions []batchv1.JobCondition
				if err := json.Unmarshal([]byte(output), &conditions); g.Expect(err).To(Succeed()) {
					return
				}
				d := map[string]batchv1.JobCondition{}
				for _, c := range conditions {
					d[string(c.Type)] = c
				}
				successCriteriaMet, exist := d[string(batchv1.JobSuccessCriteriaMet)]
				if g.Expect(exist).To(BeTrue()) {
					g.Expect(successCriteriaMet.Status).To(Equal(corev1.ConditionTrue))
					g.Expect(successCriteriaMet.Reason).To(Equal(batchv1.JobReasonCompletionsReached))
				}
				complete, exist := d[string(batchv1.JobComplete)]
				if g.Expect(exist).To(BeTrue()) {
					g.Expect(complete.Status).To(Equal(corev1.ConditionTrue))
					g.Expect(complete.Reason).To(Equal(batchv1.JobReasonCompletionsReached))
				}
			}
			Eventually(ensureBatchJobStatusCompleted).Should(Succeed())
		}
		// CronJob から生み出された batch CronJob が batch Job を生み出し、それが成功していることを確認する
		ensureBatchCronJobWorking := func(name string) {
			By(fmt.Sprintf("making ensure the batch CronJob %s is working", batchCronJobName(name)))
			Eventually(func(g Gomega) {
				output, err := utils.Run(
					utils.KubectlCmd("-n", namespace, "get", "cronjob", batchCronJobName(name), "-o",
						"jsonpath={.status.lastSuccessfulTime}}"),
				)
				g.Expect(err).To(Succeed())
				g.Expect(output).ShouldNot(BeEmpty())
			}).Should(Succeed())
			By(fmt.Sprintf("get the generated batch Job by the batch CronJob %s", batchCronJobName(name)))
			var batchJobName string
			// batch CronJob から生み出された batch Job は batch CronJob を ownerReferences に持つ
			// name\towner1 owner2 のような出力をさせる
			Eventually(func(g Gomega) {
				output, err := utils.Run(
					utils.KubectlCmd("-n", namespace, "get", "job", "-o",
						`jsonpath={range .items[*]}{.metadata.name}{"\t"}{range .metadata.ownerReferences[*]}{.name}{" "}{end}{"\n"}{end}`), //nolint:lll
				)
				if !g.Expect(err).To(Succeed()) {
					return
				}
				lines := strings.Split(strings.TrimSpace(output), "\n")
				if !g.Expect(lines).ShouldNot(BeEmpty()) {
					return
				}
				// batch Job の名前 -> owner names の辞書
				nameToOwners := map[string][]string{}
				for _, line := range lines {
					xs := strings.Split(line, "\t")
					if len(xs) != 2 {
						continue
					}

					owners := []string{}
					for _, x := range strings.Split(xs[1], " ") {
						s := strings.TrimSpace(x)
						if s != "" {
							owners = append(owners, s)
						}
					}
					nameToOwners[xs[0]] = owners
				}
				By(fmt.Sprintf("find the batch Job with owner %s", batchCronJobName(name)))
				for bjName, owners := range nameToOwners {
					if slices.Contains(owners, batchCronJobName(name)) {
						// owner が cronjob である job が見つかった
						batchJobName = bjName
						break
					}
				}
				g.Expect(batchJobName).ShouldNot(BeEmpty())
			}).Should(Succeed())

			By(fmt.Sprintf("making ensure the batch Job %s generated by the batch CronJob %s is completed",
				batchJobName, batchCronJobName(name)))
			ensureBatchJobStatusCompleted(batchJobName)
		}
		// Job から生み出された batch Job の名前一覧を取得する
		listBatchJobs := func(jobName string) ([]string, error) {
			output, err := utils.Run(
				utils.KubectlCmd(
					"-n", namespace, "get", "job",
					"-o", "jsonpath={.items[*].metadata.name}",
					"-l", "app.kubernetes.io/created-by=pixiv-job-controller",
					"-l", fmt.Sprintf("jobs.pixiv.net/name=%s", jobName),
				),
			)
			if err != nil {
				return nil, err
			}
			return strings.Split(output, " "), nil
		}
		// Job から生み出された batch Job が成功することを確認する
		ensureBatchJobCompleted := func(jobName string) {
			By(fmt.Sprintf("making ensure the batch Job generated by the Job %s is successfully completed", jobName))
			var name string
			ensureBatchJob := func(g Gomega) {
				names, err := listBatchJobs(jobName)
				if !g.Expect(err).To(Succeed(), "Failed to get generated batch Jobs") {
					return
				}
				if !g.Expect(names).Should(HaveLen(1)) {
					return
				}
				name = names[0]
				g.Expect(name).ShouldNot(BeEmpty())
			}
			Eventually(ensureBatchJob).Should(Succeed()) // batch Job の名前が取れるまで待機する
			ensureBatchJobStatusCompleted(name)
		}
		// Job が生み出した batch Job が1つだけであることを確認する
		// もしそうならその batch Job の名前を返す
		ensureOnlyOneBatchJobCreated := func(jobName string) (string, bool) {
			By(fmt.Sprintf("making ensure the Job %s created only one batch Job", jobName))
			names, err := listBatchJobs(jobName)
			if !Expect(err).To(Succeed()) {
				return "", false
			}
			if !Expect(names).Should(HaveLen(1)) {
				return "", false
			}
			return names[0], true
		}

		It("should reconcile resources successfully", func() {
			type testCase struct {
				manifest            string          // test/manifests/ のディレクトリ
				additionalAssertion func(*testCase) // 追加アサーション. nil ならば何もしない
			}
			testCases := []testCase{
				{
					manifest: "sample",
					additionalAssertion: func(tc *testCase) {
						name, ok := ensureOnlyOneBatchJobCreated("job-" + tc.manifest)
						if !ok {
							return
						}
						output, err := utils.Run(utils.KubectlCmd("-n", namespace,
							"get", "job", name, "-o", "jsonpath={.spec.template.spec.containers[0].command}",
						))
						if !Expect(err).To(Succeed()) {
							return
						}
						Expect(output).To(Equal(`["perl","-Mbignum=bpi","-wle","print bpi(100)"]`))
					},
				},
				{
					manifest: "simple",
					additionalAssertion: func(tc *testCase) {
						name, ok := ensureOnlyOneBatchJobCreated("job-" + tc.manifest)
						if !ok {
							return
						}
						output, err := utils.Run(utils.KubectlCmd("-n", namespace,
							"get", "job", name, "-o", "jsonpath={.spec.template.spec.containers[0].name}",
						))
						if !Expect(err).To(Succeed()) {
							return
						}
						Expect(output).To(Equal(`simple`))
					},
				},
				{
					manifest: "complex",
					additionalAssertion: func(tc *testCase) {
						name, ok := ensureOnlyOneBatchJobCreated("job-" + tc.manifest)
						if !ok {
							return
						}
						output, err := utils.Run(utils.KubectlCmd("-n", namespace,
							"get", "job", name, "-o", "jsonpath={.spec.template.spec.containers[*].name}",
						))
						if !Expect(err).To(Succeed()) {
							return
						}
						Expect(output).To(Equal(`pi complex`))
						output, err = utils.Run(utils.KubectlCmd("-n", namespace,
							"get", "job", name, "-o", "jsonpath={.spec.template.spec.containers[1].command}",
						))
						if !Expect(err).To(Succeed()) {
							return
						}
						Expect(output).To(Equal(`["perl","-Mbignum=bpi","-wle","print bpi(10)"]`))
					},
				},
			}

			for _, tc := range testCases {
				By(fmt.Sprintf("reconcile manifest %s", tc.manifest))
				applyManifests(tc.manifest)
				ensurePodProfile("podprofile-" + tc.manifest)
				ensureJob("job-" + tc.manifest)
			}
			for _, tc := range testCases {
				ensureBatchJobCompleted("job-" + tc.manifest)
			}
			for _, tc := range testCases {
				if f := tc.additionalAssertion; f != nil {
					By(fmt.Sprintf("additional assertion of %s", tc.manifest))
					f(&tc)
				}
			}

			//
			// CronJob の検証
			//
			const cronJobName = "cronjob-sample"
			ensureCronJob(cronJobName)
			ensureBatchCronJob(cronJobName)
			ensureBatchCronJobWorking(cronJobName)
		})
	})
})

// serviceAccountToken returns a token for the specified service account in the given namespace.
// It uses the Kubernetes TokenRequest API to generate a token by directly sending a request
// and parsing the resulting token from the API response.
func serviceAccountToken() (string, error) {
	const tokenRequestRawString = `{
		"apiVersion": "authentication.k8s.io/v1",
		"kind": "TokenRequest"
	}`

	// Temporary file to store the token request
	secretName := fmt.Sprintf("%s-token-request", serviceAccountName)
	tokenRequestFile := filepath.Join("/tmp", secretName)
	err := os.WriteFile(tokenRequestFile, []byte(tokenRequestRawString), os.FileMode(0o644))
	if err != nil {
		return "", err
	}

	var out string
	verifyTokenCreation := func(g Gomega) {
		// Execute kubectl command to create the token
		cmd := utils.KubectlCmd("create", "--raw", fmt.Sprintf(
			"/api/v1/namespaces/%s/serviceaccounts/%s/token",
			namespace,
			serviceAccountName,
		), "-f", tokenRequestFile)

		output, err := cmd.CombinedOutput()
		g.Expect(err).NotTo(HaveOccurred())

		// Parse the JSON output to extract the token
		var token tokenRequest
		err = json.Unmarshal(output, &token)
		g.Expect(err).NotTo(HaveOccurred())

		out = token.Status.Token
	}
	Eventually(verifyTokenCreation).Should(Succeed())

	return out, err
}

// getMetricsOutput retrieves and returns the logs from the curl pod used to access the metrics endpoint.
func getMetricsOutput() string {
	By("getting the curl-metrics logs")
	cmd := utils.KubectlCmd("logs", "curl-metrics", "-n", namespace)
	metricsOutput, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to retrieve logs from curl pod")
	Expect(metricsOutput).To(ContainSubstring("< HTTP/1.1 200 OK"))
	return metricsOutput
}

// tokenRequest is a simplified representation of the Kubernetes TokenRequest API response,
// containing only the token field that we need to extract.
type tokenRequest struct {
	Status struct {
		Token string `json:"token"`
	} `json:"status"`
}
