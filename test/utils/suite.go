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

package utils

import (
	"fmt"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type Suite struct {
	Name            string
	NamespacePrefix string
	KustomizeRoot   string
	Testcases       []Testcase
}

func (s Suite) namespace(i int) string {
	return fmt.Sprintf("%s-%d", s.NamespacePrefix, i)
}

func (s Suite) createNamespace(namespace string) error {
	_, err := Run(KubectlCmd("create", "namespace", namespace))
	return err
}

func (s Suite) deleteNamespace(namespace string, wait bool) error {
	_, err := Run(KubectlCmd("delete", "namespace", namespace, "--ignore-not-found=true", fmt.Sprintf("--wait=%v", wait)))
	return err
}

func (s Suite) ensureNamespace(namespace string) error {
	if err := s.deleteNamespace(namespace, true); err != nil {
		return err
	}
	return s.createNamespace(namespace)
}

func (s Suite) testcaseArg(i int) *TestcaseArg {
	return &TestcaseArg{
		Namespace:     s.namespace(i),
		KustomizeRoot: s.KustomizeRoot,
	}
}

func (s Suite) Run() bool {
	return Context(s.Name, func() {
		It("should successfully create namespace", func() {
			for i := range s.Testcases {
				Expect(s.ensureNamespace(s.namespace(i))).To(Succeed())
			}
		})
		for i, tc := range s.Testcases {
			tc.run(s.testcaseArg(i))
		}
		It("should successfully delete namespace", func() {
			for i := range s.Testcases {
				Expect(s.deleteNamespace(s.namespace(i), false)).To(Succeed())
			}
		})
	})
}

type TestcaseArg struct {
	Namespace     string
	KustomizeRoot string
}

type Testcase struct {
	Name         string
	KustomizeDir string
	Steps        []Step
}

func (t Testcase) stepArg(a *TestcaseArg) *StepArg {
	return &StepArg{
		Namespace:     a.Namespace,
		KustomizeRoot: filepath.Join(a.KustomizeRoot, t.KustomizeDir),
	}
}

func (t Testcase) cleanup(a *TestcaseArg) error {
	for _, s := range t.Steps {
		if err := s.deleteManifest(t.stepArg(a), false); err != nil {
			return err
		}
	}
	return nil
}

func (t Testcase) run(a *TestcaseArg) {
	Context(t.Name, Ordered, func() {
		AfterAll(func() {
			By("cleanup")
			t.cleanup(a)
		})
		for _, s := range t.Steps {
			s.run(t.stepArg(a))
		}
	})
}

type StepArg struct {
	Namespace     string
	KustomizeRoot string
}

type Step struct {
	Name              string
	KustomizeDir      string
	Assert            func(a *StepArg)
	DeleteBeforeApply bool
}

func (s Step) manifestPath(kustomizeRoot string) string {
	return filepath.Join(kustomizeRoot, s.KustomizeDir)
}

func (s Step) deleteManifest(a *StepArg, wait bool) error {
	_, err := Run(KubectlCmd("-n", a.Namespace, "delete", "-k", s.manifestPath(a.KustomizeRoot),
		fmt.Sprintf("--wait=%v", wait),
	))
	return err
}

func (s Step) applyManifest(a *StepArg) error {
	if s.DeleteBeforeApply {
		if err := s.deleteManifest(a, true); err != nil {
			return err
		}
	}
	_, err := Run(KubectlCmd("-n", a.Namespace, "apply", "-k", s.manifestPath(a.KustomizeRoot),
		"--server-side=true",
	))
	return err
}

func (s Step) run(a *StepArg) {
	It(s.Name, func() {
		By("apply manifest")
		Expect(s.applyManifest(a)).To(Succeed())

		if f := s.Assert; f != nil {
			f(a)
		}
	})
}
