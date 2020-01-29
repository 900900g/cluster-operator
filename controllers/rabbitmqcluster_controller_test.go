/*
Copyright 2019 Pivotal.

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

package controllers_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	rabbitmqv1beta1 "github.com/pivotal/rabbitmq-for-kubernetes/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("RabbitmqclusterController", func() {

	var (
		rabbitmqCluster *rabbitmqv1beta1.RabbitmqCluster
	)

	Context("using minimal settings on the instance", func() {
		BeforeEach(func() {
			rabbitmqCluster = &rabbitmqv1beta1.RabbitmqCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rabbitmq-one",
					Namespace: "rabbitmq-one",
				},
				Spec: rabbitmqv1beta1.RabbitmqClusterSpec{
					Replicas: 1,
				},
			}

			Expect(client.Create(context.TODO(), rabbitmqCluster)).NotTo(HaveOccurred())
			waitForClusterCreation(rabbitmqCluster, client)

		})
		AfterEach(func() {
			Expect(client.Delete(context.TODO(), rabbitmqCluster)).To(Succeed())
		})

		It("works", func() {
			By("creating a statefulset with default configurations", func() {
				statefulSetName := rabbitmqCluster.ChildResourceName("server")
				var sts *appsv1.StatefulSet
				Eventually(func() error {
					var err error
					sts, err = clientSet.AppsV1().StatefulSets(rabbitmqCluster.Namespace).Get(statefulSetName, metav1.GetOptions{})
					return err
				}, 1).Should(Succeed())
				Expect(sts.Name).To(Equal(statefulSetName))

				Expect(sts.Spec.Template.Spec.ImagePullSecrets).To(BeEmpty())

				Expect(len(sts.Spec.VolumeClaimTemplates)).To(Equal(1))
				Expect(sts.Spec.VolumeClaimTemplates[0].Spec.StorageClassName).To(BeNil())
			})

			By("creating the server conf configmap", func() {
				configMapName := rabbitmqCluster.ChildResourceName("server-conf")
				configMap, err := clientSet.CoreV1().ConfigMaps(rabbitmqCluster.Namespace).Get(configMapName, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(configMap.Name).To(Equal(configMapName))
				Expect(configMap.OwnerReferences[0].Name).To(Equal(rabbitmqCluster.Name))
			})

			By("creating a rabbitmq admin secret", func() {
				secretName := rabbitmqCluster.ChildResourceName("admin")
				secret, err := clientSet.CoreV1().Secrets(rabbitmqCluster.Namespace).Get(secretName, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(secret.Name).To(Equal(secretName))
			})

			By("creating an erlang cookie secret", func() {
				secretName := rabbitmqCluster.ChildResourceName("erlang-cookie")
				secret, err := clientSet.CoreV1().Secrets(rabbitmqCluster.Namespace).Get(secretName, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(secret.Name).To(Equal(secretName))
			})

			By("creating a rabbitmq ingress service", func() {
				ingressServiceName := rabbitmqCluster.ChildResourceName("ingress")
				service, err := clientSet.CoreV1().Services(rabbitmqCluster.Namespace).Get(ingressServiceName, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(service.Name).To(Equal(ingressServiceName))
				Expect(service.OwnerReferences[0].Name).To(Equal(rabbitmqCluster.Name))
				Expect(service.Spec.Type).To(Equal(corev1.ServiceTypeClusterIP))
			})

			By("creating a rabbitmq headless service", func() {
				headlessServiceName := rabbitmqCluster.ChildResourceName("headless")
				service, err := clientSet.CoreV1().Services(rabbitmqCluster.Namespace).Get(headlessServiceName, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(service.Name).To(Equal(headlessServiceName))
				Expect(service.OwnerReferences[0].Name).To(Equal(rabbitmqCluster.Name))
			})

			By("creating a service account", func() {
				name := rabbitmqCluster.ChildResourceName("server")
				var serviceAccount *corev1.ServiceAccount
				Eventually(func() error {
					var err error
					serviceAccount, err = clientSet.CoreV1().ServiceAccounts(rabbitmqCluster.Namespace).Get(name, metav1.GetOptions{})
					return err
				}, 1).Should(Succeed())
				Expect(serviceAccount.Name).To(Equal(name))
			})

			By("creating a role", func() {
				name := rabbitmqCluster.ChildResourceName("endpoint-discovery")
				var role *rbacv1.Role
				Eventually(func() error {
					var err error
					role, err = clientSet.RbacV1().Roles(rabbitmqCluster.Namespace).Get(name, metav1.GetOptions{})
					return err
				}, 1).Should(Succeed())
				Expect(role.Name).To(Equal(name))
			})

			By("creating a role binding", func() {
				name := rabbitmqCluster.ChildResourceName("server")
				var roleBinding *rbacv1.RoleBinding
				Eventually(func() error {
					var err error
					roleBinding, err = clientSet.RbacV1().RoleBindings(rabbitmqCluster.Namespace).Get(name, metav1.GetOptions{})
					return err
				}, 1).Should(Succeed())
				Expect(roleBinding.Name).To(Equal(name))
			})
		})
	})

	Context("Annotations set on the instance", func() {
		BeforeEach(func() {
			rabbitmqCluster = &rabbitmqv1beta1.RabbitmqCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rabbitmq-annotations",
					Namespace: "rabbitmq-annotations",
					Annotations: map[string]string{
						"my-annotation": "this-annotation",
					},
				},
				Spec: rabbitmqv1beta1.RabbitmqClusterSpec{
					Replicas:        1,
					ImagePullSecret: "rabbit-two-secret",
				},
			}

			Expect(client.Create(context.TODO(), rabbitmqCluster)).To(Succeed())
			waitForClusterCreation(rabbitmqCluster, client)
		})

		AfterEach(func() {
			Expect(client.Delete(context.TODO(), rabbitmqCluster)).To(Succeed())
		})

		It("adds annotations to child resources", func() {
			Eventually(func() map[string]string {
				service, _ := clientSet.CoreV1().Services(rabbitmqCluster.Namespace).Get(rabbitmqCluster.ChildResourceName("headless"), metav1.GetOptions{})
				return service.Annotations
			}, 1).Should(HaveKeyWithValue("my-annotation", "this-annotation"))
			var sts *appsv1.StatefulSet
			Eventually(func() map[string]string {
				sts, _ = clientSet.AppsV1().StatefulSets(rabbitmqCluster.Namespace).Get(rabbitmqCluster.ChildResourceName("server"), metav1.GetOptions{})
				return sts.Annotations
			}, 1).Should(HaveKeyWithValue("my-annotation", "this-annotation"))
		})

	})

	Context("ImagePullSecret configure on the instance", func() {
		BeforeEach(func() {
			rabbitmqCluster = &rabbitmqv1beta1.RabbitmqCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rabbitmq-two",
					Namespace: "rabbitmq-two",
				},
				Spec: rabbitmqv1beta1.RabbitmqClusterSpec{
					Replicas:        1,
					ImagePullSecret: "rabbit-two-secret",
				},
			}

			Expect(client.Create(context.TODO(), rabbitmqCluster)).To(Succeed())
			waitForClusterCreation(rabbitmqCluster, client)
		})

		AfterEach(func() {
			Expect(client.Delete(context.TODO(), rabbitmqCluster)).To(Succeed())
		})

		It("configures the imagePullSecret on sts correctly", func() {
			By("using the instance spec secret", func() {
				stsName := rabbitmqCluster.ChildResourceName("server")
				Eventually(func() []corev1.LocalObjectReference {
					sts, _ := clientSet.AppsV1().StatefulSets(rabbitmqCluster.Namespace).Get(stsName, metav1.GetOptions{})
					return sts.Spec.Template.Spec.ImagePullSecrets
				}, 1).Should(ContainElement(corev1.LocalObjectReference{Name: "rabbit-two-secret"}))
			})
		})
	})

	Context("Affinity configurations", func() {
		var affinity = &corev1.Affinity{
			PodAffinity: &corev1.PodAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
					{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"affinity-label": "anti-affinity",
							},
						},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
			},
		}
		BeforeEach(func() {
			rabbitmqCluster = &rabbitmqv1beta1.RabbitmqCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rabbitmq-affinity",
					Namespace: "rabbitmq-affinity",
				},
				Spec: rabbitmqv1beta1.RabbitmqClusterSpec{
					Replicas:        1,
					Affinity:        affinity,
					ImagePullSecret: "rabbit-two-secret",
				},
			}
			Expect(client.Create(context.TODO(), rabbitmqCluster)).NotTo(HaveOccurred())
			waitForClusterCreation(rabbitmqCluster, client)
		})

		AfterEach(func() {
			Expect(client.Delete(context.TODO(), rabbitmqCluster)).To(Succeed())
		})

		It("adds the affinity rules to pod spec", func() {
			sts := statefulSet(rabbitmqCluster)
			podSpecAffinity := sts.Spec.Template.Spec.Affinity
			Expect(podSpecAffinity).To(Equal(affinity))
		})
	})

	Context("Ingress service configurations", func() {
		AfterEach(func() {
			Expect(client.Delete(context.TODO(), rabbitmqCluster)).To(Succeed())
			Expect(clientSet.CoreV1().Services(rabbitmqCluster.Namespace).Delete(rabbitmqCluster.ChildResourceName("ingress"), &metav1.DeleteOptions{}))
		})

		It("creates the service type and annotations as configured in instance spec", func() {
			rabbitmqCluster = &rabbitmqv1beta1.RabbitmqCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rabbit-service-2",
					Namespace: "rabbit-service-2",
				},
				Spec: rabbitmqv1beta1.RabbitmqClusterSpec{
					Replicas:        1,
					ImagePullSecret: "rabbit-service-secret",
				},
			}
			rabbitmqCluster.Spec.Service.Type = "LoadBalancer"
			rabbitmqCluster.Spec.Service.Annotations = map[string]string{"annotations": "cr-annotation"}

			Expect(client.Create(context.TODO(), rabbitmqCluster)).NotTo(HaveOccurred())
			waitForClusterCreation(rabbitmqCluster, client)

			serviceName := rabbitmqCluster.ChildResourceName("ingress")
			Eventually(func() string {
				svc, err := clientSet.CoreV1().Services(rabbitmqCluster.Namespace).Get(serviceName, metav1.GetOptions{})
				if err != nil {
					Expect(err).To(MatchError(fmt.Sprintf("services \"%s\" not found", serviceName)))
					return fmt.Sprintf("service: %s not found \n", serviceName)
				}
				return string(svc.Spec.Type)
			}, 1).Should(Equal("LoadBalancer"))

			Eventually(func() map[string]string {
				svc, err := clientSet.CoreV1().Services(rabbitmqCluster.Namespace).Get(serviceName, metav1.GetOptions{})
				if err != nil {
					Expect(err).To(MatchError(fmt.Sprintf("services \"%s\" not found", serviceName)))
					return nil
				}

				return svc.Annotations
			}, 1).Should(Equal(map[string]string{
				"annotations": "cr-annotation",
			}))
		})
	})

	Context("Resource requirements configurations", func() {
		AfterEach(func() {
			Expect(client.Delete(context.TODO(), rabbitmqCluster)).To(Succeed())
			Expect(clientSet.AppsV1().StatefulSets(rabbitmqCluster.Namespace).Delete(rabbitmqCluster.ChildResourceName("server"), nil)).To(Succeed())
		})

		It("uses resource requirements from instance spec when provided", func() {
			rabbitmqCluster = &rabbitmqv1beta1.RabbitmqCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rabbit-resource-2",
					Namespace: "rabbit-resource-2",
				},
				Spec: rabbitmqv1beta1.RabbitmqClusterSpec{
					Replicas:        1,
					ImagePullSecret: "rabbit-resource-secret",
				},
			}
			rabbitmqCluster.Spec.Resources = &corev1.ResourceRequirements{
				Limits: map[corev1.ResourceName]k8sresource.Quantity{
					corev1.ResourceMemory: k8sresource.MustParse("4Gi"),
				},
				Requests: map[corev1.ResourceName]k8sresource.Quantity{
					corev1.ResourceMemory: k8sresource.MustParse("4Gi"),
				},
			}

			Expect(client.Create(context.TODO(), rabbitmqCluster)).NotTo(HaveOccurred())
			waitForClusterCreation(rabbitmqCluster, client)

			sts := statefulSet(rabbitmqCluster)

			actualResources := sts.Spec.Template.Spec.Containers[0].Resources
			expectedResources := corev1.ResourceRequirements{
				Limits: map[corev1.ResourceName]k8sresource.Quantity{
					corev1.ResourceMemory: k8sresource.MustParse("4Gi"),
				},
				Requests: map[corev1.ResourceName]k8sresource.Quantity{
					corev1.ResourceMemory: k8sresource.MustParse("4Gi"),
				},
			}

			Expect(actualResources).To(Equal(expectedResources))

		})
	})

	Context("Persistence configurations", func() {
		AfterEach(func() {
			Expect(client.Delete(context.TODO(), rabbitmqCluster)).To(Succeed())
			Expect(clientSet.AppsV1().StatefulSets(rabbitmqCluster.Namespace).Delete(rabbitmqCluster.ChildResourceName("server"), nil)).To(Succeed())
		})

		It("creates the RabbitmqCluster with the specified storage from instance spec", func() {
			rabbitmqCluster = &rabbitmqv1beta1.RabbitmqCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rabbit-persistence-1",
					Namespace: "rabbit-persistence-1",
				},
				Spec: rabbitmqv1beta1.RabbitmqClusterSpec{
					Replicas:        1,
					ImagePullSecret: "rabbit-resource-secret",
				},
			}
			storageClassName := "my-storage-class"
			rabbitmqCluster.Spec.Persistence.StorageClassName = &storageClassName
			storage := k8sresource.MustParse("100Gi")
			rabbitmqCluster.Spec.Persistence.Storage = &storage
			Expect(client.Create(context.TODO(), rabbitmqCluster)).NotTo(HaveOccurred())
			waitForClusterCreation(rabbitmqCluster, client)

			sts := statefulSet(rabbitmqCluster)

			Expect(len(sts.Spec.VolumeClaimTemplates)).To(Equal(1))
			Expect(*sts.Spec.VolumeClaimTemplates[0].Spec.StorageClassName).To(Equal("my-storage-class"))
			actualStorageCapacity := sts.Spec.VolumeClaimTemplates[0].Spec.Resources.Requests[corev1.ResourceStorage]
			Expect(actualStorageCapacity).To(Equal(k8sresource.MustParse("100Gi")))
		})
	})

	Context("Custom Resource updates", func() {
		var (
			ingressServiceName string
			statefulSetName    string
		)
		BeforeEach(func() {
			rabbitmqCluster = &rabbitmqv1beta1.RabbitmqCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rabbitmq-cr-update",
					Namespace: "rabbitmq-cr-update",
				},
				Spec: rabbitmqv1beta1.RabbitmqClusterSpec{
					Replicas:        1,
					ImagePullSecret: "rabbit-two-secret",
				},
			}
			ingressServiceName = rabbitmqCluster.ChildResourceName("ingress")
			statefulSetName = rabbitmqCluster.ChildResourceName("server")

			Expect(client.Create(context.TODO(), rabbitmqCluster)).To(Succeed())
			waitForClusterCreation(rabbitmqCluster, client)
		})

		AfterEach(func() {
			Expect(client.Delete(context.TODO(), rabbitmqCluster)).To(Succeed())
		})

		It("reconciles an existing instance", func() {
			Expect(client.Get(
				context.TODO(),
				types.NamespacedName{Name: rabbitmqCluster.Name, Namespace: rabbitmqCluster.Namespace},
				rabbitmqCluster,
			)).To(Succeed())

			When("the service annotations are updated", func() {
				rabbitmqCluster.Spec.Service.Annotations = map[string]string{"test-key": "test-value"}
				Expect(client.Update(context.TODO(), rabbitmqCluster)).To(Succeed())
				Eventually(func() map[string]string {
					ingressServiceName := rabbitmqCluster.ChildResourceName("ingress")
					service, _ := clientSet.CoreV1().Services(rabbitmqCluster.Namespace).Get(ingressServiceName, metav1.GetOptions{})
					return service.Annotations
				}, 1).Should(HaveKeyWithValue("test-key", "test-value"))
			})

			When("the CPU and memory requirements are updated", func() {
				var resourceRequirements corev1.ResourceRequirements
				expectedRequirements := &corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    k8sresource.MustParse("1100m"),
						corev1.ResourceMemory: k8sresource.MustParse("5Gi"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    k8sresource.MustParse("1200m"),
						corev1.ResourceMemory: k8sresource.MustParse("6Gi"),
					},
				}
				rabbitmqCluster.Spec.Resources = expectedRequirements
				Expect(client.Update(context.TODO(), rabbitmqCluster)).To(Succeed())

				Eventually(func() corev1.ResourceList {
					stsName := rabbitmqCluster.ChildResourceName("server")
					sts, err := clientSet.AppsV1().StatefulSets(rabbitmqCluster.Namespace).Get(stsName, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					resourceRequirements = sts.Spec.Template.Spec.Containers[0].Resources
					return resourceRequirements.Requests
				}, 1).Should(HaveKeyWithValue(corev1.ResourceCPU, expectedRequirements.Requests[corev1.ResourceCPU]))
				Expect(resourceRequirements.Limits).To(HaveKeyWithValue(corev1.ResourceCPU, expectedRequirements.Limits[corev1.ResourceCPU]))

				Expect(resourceRequirements.Requests).To(HaveKeyWithValue(corev1.ResourceMemory, expectedRequirements.Requests[corev1.ResourceMemory]))
				Expect(resourceRequirements.Limits).To(HaveKeyWithValue(corev1.ResourceMemory, expectedRequirements.Limits[corev1.ResourceMemory]))
			})

			When("the rabbitmq image is updated", func() {
				rabbitmqCluster.Spec.Image = "rabbitmq:3.8.0"
				Expect(client.Update(context.TODO(), rabbitmqCluster)).To(Succeed())

				Eventually(func() string {
					stsName := rabbitmqCluster.ChildResourceName("server")
					sts, _ := clientSet.AppsV1().StatefulSets(rabbitmqCluster.Namespace).Get(stsName, metav1.GetOptions{})
					return sts.Spec.Template.Spec.Containers[0].Image
				}, 1).Should(Equal("rabbitmq:3.8.0"))
			})

			When("the rabbitmq ImagePullSecret is updated", func() {
				rabbitmqCluster.Spec.ImagePullSecret = "my-new-secret"
				Expect(client.Update(context.TODO(), rabbitmqCluster)).To(Succeed())

				Eventually(func() []corev1.LocalObjectReference {
					stsName := rabbitmqCluster.ChildResourceName("server")
					sts, _ := clientSet.AppsV1().StatefulSets(rabbitmqCluster.Namespace).Get(stsName, metav1.GetOptions{})
					Expect(len(sts.Spec.Template.Spec.ImagePullSecrets)).To(Equal(1))
					return sts.Spec.Template.Spec.ImagePullSecrets
				}, 1).Should(ConsistOf(corev1.LocalObjectReference{Name: "my-new-secret"}))
			})

			When("labels are updated", func() {
				rabbitmqCluster.Labels = make(map[string]string)
				rabbitmqCluster.Labels["foo"] = "bar"
				Expect(client.Update(context.TODO(), rabbitmqCluster)).To(Succeed())

				Eventually(func() map[string]string {
					service, err := clientSet.CoreV1().Services(rabbitmqCluster.Namespace).Get(ingressServiceName, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					return service.Labels
				}, 1).Should(HaveKeyWithValue("foo", "bar"))
				var sts *appsv1.StatefulSet
				Eventually(func() map[string]string {
					sts, _ = clientSet.AppsV1().StatefulSets(rabbitmqCluster.Namespace).Get(statefulSetName, metav1.GetOptions{})
					return sts.Labels
				}, 1).Should(HaveKeyWithValue("foo", "bar"))
			})

			When("instance annotations are updated", func() {
				rabbitmqCluster.Annotations = make(map[string]string)
				rabbitmqCluster.Annotations["anno-key"] = "anno-value"
				Expect(client.Update(context.TODO(), rabbitmqCluster)).To(Succeed())

				Eventually(func() map[string]string {
					service, err := clientSet.CoreV1().Services(rabbitmqCluster.Namespace).Get(rabbitmqCluster.ChildResourceName("headless"), metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					return service.Annotations
				}, 1).Should(HaveKeyWithValue("anno-key", "anno-value"))
				var sts *appsv1.StatefulSet
				Eventually(func() map[string]string {
					sts, _ = clientSet.AppsV1().StatefulSets(rabbitmqCluster.Namespace).Get(statefulSetName, metav1.GetOptions{})
					return sts.Annotations
				}, 1).Should(HaveKeyWithValue("anno-key", "anno-value"))
			})

			When("service type is updated", func() {
				rabbitmqCluster.Spec.Service.Type = "NodePort"
				Expect(client.Update(context.TODO(), rabbitmqCluster)).To(Succeed())
				Eventually(func() string {
					service, err := clientSet.CoreV1().Services(rabbitmqCluster.Namespace).Get(rabbitmqCluster.ChildResourceName("ingress"), metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					return string(service.Spec.Type)
				}, 1).Should(Equal("NodePort"))
			})

			When("affinity rules are updated", func() {
				affinity := &corev1.Affinity{
					NodeAffinity: &corev1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
							NodeSelectorTerms: []corev1.NodeSelectorTerm{
								{
									MatchExpressions: []corev1.NodeSelectorRequirement{
										{
											Key:      "foo",
											Operator: "Exists",
											Values:   nil,
										},
									},
								},
							},
						},
					},
				}

				rabbitmqCluster.Spec.Affinity = affinity
				Expect(client.Update(context.TODO(), rabbitmqCluster)).To(Succeed())
				Eventually(func() *corev1.Affinity {
					sts, _ := clientSet.AppsV1().StatefulSets(rabbitmqCluster.Namespace).Get(statefulSetName, metav1.GetOptions{})
					return sts.Spec.Template.Spec.Affinity
				}, 1).Should(Equal(affinity))

				affinity = nil
				rabbitmqCluster.Spec.Affinity = affinity
				Expect(client.Update(context.TODO(), rabbitmqCluster)).To(Succeed())
				Eventually(func() *corev1.Affinity {
					sts, _ := clientSet.AppsV1().StatefulSets(rabbitmqCluster.Namespace).Get(statefulSetName, metav1.GetOptions{})
					return sts.Spec.Template.Spec.Affinity
				}, 1).Should(BeNil())
			})
		})
	})

})

func statefulSet(rabbitmqCluster *rabbitmqv1beta1.RabbitmqCluster) *appsv1.StatefulSet {
	stsName := rabbitmqCluster.ChildResourceName("server")
	var sts *appsv1.StatefulSet
	Eventually(func() error {
		var err error
		sts, err = clientSet.AppsV1().StatefulSets(rabbitmqCluster.Namespace).Get(stsName, metav1.GetOptions{})
		return err
	}, 1).Should(Succeed())
	return sts
}

func waitForClusterCreation(rabbitmqCluster *rabbitmqv1beta1.RabbitmqCluster, client runtimeClient.Client) {
	Eventually(func() string {
		rabbitmqClusterCreated := rabbitmqv1beta1.RabbitmqCluster{}
		err := client.Get(
			context.TODO(),
			types.NamespacedName{Name: rabbitmqCluster.Name, Namespace: rabbitmqCluster.Namespace},
			&rabbitmqClusterCreated,
		)
		if err != nil {
			return fmt.Sprintf("%v+", err)
		}

		return rabbitmqClusterCreated.Status.ClusterStatus

	}, 2, 1).Should(ContainSubstring("created"))
}
