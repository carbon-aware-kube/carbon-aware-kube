/*
Copyright 2025.

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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	batchv1alpha1 "carbon-aware-kube.dev/carbon-aware-kube/api/v1alpha1"
	carbonv1alpha1 "carbon-aware-kube.dev/carbon-aware-kube/api/v1alpha1"
)

var _ = Describe("CarbonAwareJob Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		var (
			flexWindow       = int32(10)
			expectedDuration = int32(30)
			jobTemplate      = batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "test-container",
									Image: "busybox",
								},
							},
						},
					},
				},
			}
		)

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		carbonawarejob := &carbonv1alpha1.CarbonAwareJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: "default",
			},
			Spec: carbonv1alpha1.CarbonAwareJobSpec{
				JobTemplate:             jobTemplate,
				StartFlexWindowSeconds:  &flexWindow,
				ExpectedDurationSeconds: &expectedDuration,
			},
		}

		BeforeEach(func() {
			By("creating the custom resource for the Kind CarbonAwareJob")
			err := k8sClient.Get(ctx, typeNamespacedName, carbonawarejob)
			if err != nil && errors.IsNotFound(err) {
				resource := &batchv1alpha1.CarbonAwareJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: carbonv1alpha1.CarbonAwareJobSpec{
						JobTemplate:             jobTemplate,
						StartFlexWindowSeconds:  &flexWindow,
						ExpectedDurationSeconds: &expectedDuration,
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
				Expect(k8sClient.Get(ctx, typeNamespacedName, carbonawarejob)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &batchv1alpha1.CarbonAwareJob{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance CarbonAwareJob")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &CarbonAwareJobReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})
})
