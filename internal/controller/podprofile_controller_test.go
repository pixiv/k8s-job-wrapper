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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	pixivnetv1 "github.com/pixiv/k8s-job-wrapper/api/v1"
)

var _ = new(podProfileControllerTest).run()

type podProfileControllerTest struct {
	controllerTestBase
}

func (p podProfileControllerTest) run() bool {
	return controllerTest{
		name:            "PodProfileController",
		namespacePrefix: "podprofile",
		contexts: []controllerTestContext{
			p.reconcileNormalTest(),
		},
	}.run()
}

func (podProfileControllerTest) newReconciler() *PodProfileReconciler {
	return &PodProfileReconciler{
		Client: k8sClient,
		Scheme: k8sClient.Scheme(),
	}
}

func (p podProfileControllerTest) reconcileNormalTest() controllerTestContext {
	const resourceName = "sample"
	return controllerTestContext{
		name: "When reconciling a resource",
		beforeEach: func(a *controllerTestContextArg) {
			typeNamespacedName := p.newNSName(a.namespace, resourceName)
			By("creating the custom resource for the Kind PodProfile")
			exist, err := Exist[*pixivnetv1.PodProfile](ctx, k8sClient, typeNamespacedName)
			Expect(err).To(Succeed())
			if !exist {
				Expect(k8sClient.Create(ctx, p.newPodProfile(
					typeNamespacedName.Namespace, typeNamespacedName.Name,
				))).To(Succeed())
			}
		},
		afterEach: func(a *controllerTestContextArg) {
			typeNamespacedName := p.newNSName(a.namespace, resourceName)
			resource, err := Get[*pixivnetv1.PodProfile](ctx, k8sClient, typeNamespacedName)
			Expect(err).To(Succeed())
			By("Cleanup the specific resource instance PodProfile")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		},
		test: func(a *controllerTestContextArg) {
			typeNamespacedName := p.newNSName(a.namespace, resourceName)
			It("should successfully reconcile the resource", func() {
				By("Reconciling the created resource")
				controllerReconciler := p.newReconciler()

				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).To(Succeed())
			})
		},
	}
}
