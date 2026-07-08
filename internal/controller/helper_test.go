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

package controller

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type controllerTest struct {
	name            string
	namespacePrefix string
	contexts        []controllerTestContext
	beforeEach      func()
	afterEach       func()
}

func (c controllerTest) run() bool {
	return Describe(c.name, func() {
		if f := c.beforeEach; f != nil {
			BeforeEach(func() {
				f()
			})
		}
		for i, x := range c.contexts {
			a := &controllerTestContextArg{
				namespace: fmt.Sprintf("%s-%d", c.namespacePrefix, i),
			}
			x.run(a)
		}
		if f := c.afterEach; f != nil {
			AfterEach(func() {
				f()
			})
		}
	})
}

type controllerTestContextArg struct {
	namespace string
}

type controllerTestContext struct {
	name       string
	test       func(a *controllerTestContextArg)
	beforeEach func(a *controllerTestContextArg)
	afterEach  func(a *controllerTestContextArg)
}

func (c controllerTestContext) run(a *controllerTestContextArg) {
	Context(c.name, func() {
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: a.namespace,
			},
		}
		BeforeEach(func() {
			Expect(k8sClient.Create(ctx, namespace)).To(Succeed())
			if f := c.beforeEach; f != nil {
				f(a)
			}
		})
		c.test(a)
		AfterEach(func() {
			if f := c.afterEach; f != nil {
				f(a)
			}
			Expect(k8sClient.Delete(ctx, namespace)).To(Succeed())
		})
	})
}
