package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/doodlescheduling/neo4j-aura-controller/api/v1beta1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
)

var _ = Describe("AuraInstance controller", func() {
	const (
		timeout  = time.Second * 4
		interval = time.Millisecond * 600
	)

	When("reconciling a suspendended AuraInstance", func() {
		instanceName := fmt.Sprintf("cluster-%s", rand.String(5))

		It("should not update the status", func() {
			By("creating a new AuraInstance")
			ctx := context.Background()

			gi := &v1beta1.AuraInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instanceName,
					Namespace: "default",
				},
				Spec: v1beta1.AuraInstanceSpec{
					TenantID:      "x",
					Neo4jVersion:  "5",
					Tier:          "free-db",
					CloudProvider: "gcp",
					Suspend:       true,
				},
			}
			Expect(k8sClient.Create(ctx, gi)).Should(Succeed())

			By("waiting for the reconciliation")
			instanceLookupKey := types.NamespacedName{Name: instanceName, Namespace: "default"}
			reconciledInstance := &v1beta1.AuraInstance{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, instanceLookupKey, reconciledInstance)
				if err != nil {
					return false
				}

				return len(reconciledInstance.Status.Conditions) == 0
			}, timeout, interval).Should(BeTrue())
		})
	})

	instanceName := fmt.Sprintf("instance-%s", rand.String(5))
	secretName := fmt.Sprintf("secret-%s", rand.String(5))

	When("it can't find the referenced secret with credentials", func() {
		It("should update the status", func() {
			By("creating a new AuraInstance")
			ctx := context.Background()

			gi := &v1beta1.AuraInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instanceName,
					Namespace: "default",
				},
				Spec: v1beta1.AuraInstanceSpec{
					TenantID:      "x",
					Neo4jVersion:  "5",
					Tier:          "free-db",
					CloudProvider: "gcp",
					Secret: v1beta1.LocalObjectReference{
						Name: secretName,
					},
				},
			}
			Expect(k8sClient.Create(ctx, gi)).Should(Succeed())

			By("waiting for the reconciliation")
			instanceLookupKey := types.NamespacedName{Name: instanceName, Namespace: "default"}
			reconciledInstance := &v1beta1.AuraInstance{}

			expectedStatus := &v1beta1.AuraInstanceStatus{
				ObservedGeneration: 1,
				Conditions: []metav1.Condition{
					{
						Type:    v1beta1.ConditionReady,
						Status:  metav1.ConditionFalse,
						Reason:  "ReconciliationFailed",
						Message: fmt.Sprintf(`failed to get secret: Secret "%s" not found`, secretName),
					},
				},
			}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, instanceLookupKey, reconciledInstance)
				if err != nil {
					return false
				}

				return needConditions(expectedStatus.Conditions, reconciledInstance.Status.Conditions)
			}, timeout, interval).Should(BeTrue())
		})
	})

	When("it can't find the clientID from the secret", func() {
		It("should update the status", func() {
			By("creating a secret")
			ctx := context.Background()

			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: "default",
				},
			}
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())

			By("waiting for the reconciliation")
			instanceLookupKey := types.NamespacedName{Name: instanceName, Namespace: "default"}
			reconciledInstance := &v1beta1.AuraInstance{}

			expectedStatus := &v1beta1.AuraInstanceStatus{
				ObservedGeneration: 1,
				Conditions: []metav1.Condition{
					{
						Type:    v1beta1.ConditionReady,
						Status:  metav1.ConditionFalse,
						Reason:  "ReconciliationFailed",
						Message: "secret must contain clientID and clientSecret",
					},
				},
			}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, instanceLookupKey, reconciledInstance)
				if err != nil {
					return false
				}

				return needConditions(expectedStatus.Conditions, reconciledInstance.Status.Conditions)
			}, timeout, interval).Should(BeTrue())
		})
	})

	When("it can't find the clientSecret from the secret", func() {
		It("should update the status", func() {
			By("creating a secret")
			ctx := context.Background()

			var secret corev1.Secret
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: "default"}, &secret)).Should(Succeed())

			secret.StringData = map[string]string{"clientID": "id"}
			Expect(k8sClient.Update(ctx, &secret)).Should(Succeed())

			By("waiting for the reconciliation")
			instanceLookupKey := types.NamespacedName{Name: instanceName, Namespace: "default"}
			reconciledInstance := &v1beta1.AuraInstance{}

			expectedStatus := &v1beta1.AuraInstanceStatus{
				ObservedGeneration: 1,
				Conditions: []metav1.Condition{
					{
						Type:    v1beta1.ConditionReady,
						Status:  metav1.ConditionFalse,
						Reason:  "ReconciliationFailed",
						Message: "secret must contain clientID and clientSecret",
					},
				},
			}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, instanceLookupKey, reconciledInstance)
				if err != nil {
					return false
				}

				return needConditions(expectedStatus.Conditions, reconciledInstance.Status.Conditions)
			}, timeout, interval).Should(BeTrue())
		})
	})
})
