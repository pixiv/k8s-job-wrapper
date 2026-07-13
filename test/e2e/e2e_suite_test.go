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
	"fmt"
	"os/exec"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

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

var (
	// Optional Environment Variables:
	// - CERT_MANAGER_INSTALL_SKIP=true: Skips CertManager installation during test setup.
	// These variables are useful if CertManager is already installed, avoiding
	// re-installation and conflicts.
	skipCertManagerInstall = utils.IsEnvTrue("CERT_MANAGER_INSTALL_SKIP")
	// isCertManagerAlreadyInstalled will be set true when CertManager CRDs be found on the cluster
	isCertManagerAlreadyInstalled = false

	controllerPodName string // should be initialized by SynchronizedBeforeSuite.
)

// TestE2E runs the end-to-end (e2e) test suite for the project. These tests execute in an isolated,
// temporary environment to validate project changes with the the purposed to be used in CI jobs.
// The default setup requires Kind, builds/loads the Manager Docker image locally, and installs
// CertManager.
func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	_, _ = fmt.Fprintf(GinkgoWriter, "Starting k8s-job-wrapper integration test suite\n")
	RunSpecs(t, "e2e suite")
}

var _ = SynchronizedBeforeSuite(func() {
	if !utils.IsEnvTrue("E2E_SKIP_BUILD") {
		_, _ = fmt.Fprintf(GinkgoWriter, "If you want to skip docker-build, use E2E_SKIP_BUILD=true\n")
		By("building the manager(Operator) image")
		cmd := exec.Command("make", "docker-build")
		_, err := utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to build the manager(Operator) image")
	} else {
		_, _ = fmt.Fprintf(GinkgoWriter, "Skip docker-build because E2E_SKIP_BUILD=true\n")
	}
	// TODO(user): If you want to change the e2e test vendor from Kind, ensure the image is
	// built and available before running the tests. Also, remove the following block.
	if !utils.IsEnvTrue("E2E_SKIP_LOAD") {
		By("loading the manager(Operator) image on Kind")
		ExpectWithOffset(1, utils.LoadImageToKindCluster()).
			NotTo(HaveOccurred(), "Failed to load the manager(Operator) image into Kind")
	}
	// The tests-e2e are intended to run on a temporary cluster that is created and destroyed for testing.
	// To prevent errors when tests run in environments with CertManager already installed,
	// we check for its presence before execution.
	// Setup CertManager before the suite if not skipped and if not already installed
	if !skipCertManagerInstall {
		By("checking if cert manager is installed already")
		isCertManagerAlreadyInstalled = utils.IsCertManagerCRDsInstalled()
		if !isCertManagerAlreadyInstalled {
			_, _ = fmt.Fprintf(GinkgoWriter, "Installing CertManager...\n")
			Expect(utils.InstallCertManager()).To(Succeed(), "Failed to install CertManager")
		} else {
			_, _ = fmt.Fprintf(GinkgoWriter, "WARNING: CertManager is already installed. Skipping installation...\n")
		}
	}

	// Before running the tests, set up the environment by creating the namespace,
	// enforce the restricted security policy to the namespace, installing CRDs,
	// and deploying the controller.
	By("creating manager namespace")
	cmd := utils.KubectlCmd("create", "ns", namespace)
	_, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to create namespace")

	By("labeling the namespace to enforce the restricted security policy")
	cmd = utils.KubectlCmd("label", "--overwrite", "ns", namespace,
		"pod-security.kubernetes.io/enforce=restricted")
	_, err = utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to label namespace with restricted policy")

	if !utils.IsEnvTrue("E2E_HELM") {
		By("installing CRDs")
		cmd = exec.Command("make", "install")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to install CRDs")

		By("deploying the controller-manager")
		cmd = exec.Command("make", "deploy")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy the controller-manager")
	} else {
		By("deploying the chart")
		cmd = exec.Command("make", "deploy-chart")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy the controller-manager")
	}

	// The timeout duration for Eventually().
	// If it takes longer than this, the test will be treated as a failure.
	// Therefore, the verification job is designed to be quick to execute.
	// The container image is also pre-pulled.
	// The cronjob is set to trigger every minute.
	// It's best to split Eventually() calls and reduce the number of operations to wait for within a single call.
	SetDefaultEventuallyTimeout(2 * time.Minute)
	SetDefaultEventuallyPollingInterval(time.Second)

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

	By("creating a ClusterRoleBinding for the service account to allow access to metrics")
	cmd = utils.KubectlCmd("create", "clusterrolebinding", metricsRoleBindingName,
		"--clusterrole=k8s-job-wrapper-metrics-reader",
		fmt.Sprintf("--serviceaccount=%s:%s", namespace, serviceAccountName),
	)
	_, err = utils.Run(cmd)
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

	// +kubebuilder:scaffold:e2e-webhooks-checks
}, func() {})

var _ = SynchronizedAfterSuite(func() {}, func() {
	// After all tests have been executed, clean up by undeploying the controller, uninstalling CRDs,
	// and deleting the namespace.
	By("cleaning up the curl pod for metrics")
	cmd := utils.KubectlCmd("delete", "pod", "curl-metrics", "-n", namespace)
	_, _ = utils.Run(cmd)

	if !utils.IsEnvTrue("E2E_HELM") {
		By("undeploying the controller-manager")
		cmd = exec.Command("make", "undeploy")
		_, _ = utils.Run(cmd)

		By("uninstalling CRDs")
		cmd = exec.Command("make", "uninstall")
		_, _ = utils.Run(cmd)
	} else {
		By("undeploying the chart")
		cmd = exec.Command("make", "undeploy-chart")
		_, _ = utils.Run(cmd)
	}

	By("removing manager namespace")
	cmd = utils.KubectlCmd("delete", "ns", namespace)
	_, _ = utils.Run(cmd)

	// Teardown CertManager after the suite if not skipped and if it was not already installed
	if !skipCertManagerInstall && !isCertManagerAlreadyInstalled {
		_, _ = fmt.Fprintf(GinkgoWriter, "Uninstalling CertManager...\n")
		utils.UninstallCertManager()
	}
})
