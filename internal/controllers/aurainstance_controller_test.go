package controllers

import (
	"context"
	"fmt"
	"strings"
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
					Secret: v1beta1.SecretReference{
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
						Message: "secret must contain clientID and clientSecret keys",
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
						Message: "secret must contain clientID and clientSecret keys",
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

			secret.StringData = map[string]string{"clientSecret": "secret"}
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
						Message: "xx",
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

	When("using custom clientIDKey mapping only", func() {
		It("should reconcile with custom clientID key", func() {
			By("creating a secret with custom clientId key")
			ctx := context.Background()

			secretName := fmt.Sprintf("custom-id-secret-%s", rand.String(5))
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: "default",
				},
				StringData: map[string]string{
					"customClientId": "test-id",
					"clientSecret":   "test-secret",
				},
			}
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())

			By("creating an AuraInstance with custom clientIDKey mapping")
			instanceName := fmt.Sprintf("test-custom-id-%s", rand.String(5))
			instance := &v1beta1.AuraInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instanceName,
					Namespace: "default",
				},
				Spec: v1beta1.AuraInstanceSpec{
					Tier:          v1beta1.AuraInstanceTierFreeDb,
					Region:        "us-east-1",
					CloudProvider: v1beta1.CloudProviderAWS,
					Neo4jVersion:  "5",
					TenantID:      fmt.Sprintf("tenant-%s", rand.String(5)),
					Secret: v1beta1.SecretReference{
						Name:        secretName,
						ClientIDKey: "customClientId",
						// ClientSecretKey not specified, should use default "clientSecret"
					},
				},
			}
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			By("verifying reconciliation succeeds with custom clientID key")
			instanceLookupKey := types.NamespacedName{Name: instanceName, Namespace: "default"}
			reconciledInstance := &v1beta1.AuraInstance{}

			// the reconciliation should succeed without "secret must contain" errors
			Eventually(func() bool {
				err := k8sClient.Get(ctx, instanceLookupKey, reconciledInstance)
				if err != nil {
					return false
				}
				// check that we don't get key-related errors
				for _, condition := range reconciledInstance.Status.Conditions {
					if condition.Type == v1beta1.ConditionReady &&
						condition.Status == metav1.ConditionFalse &&
						strings.Contains(condition.Message, "secret must contain") {
						return false
					}
				}
				return true
			}, timeout, interval).Should(BeTrue())
		})
	})

	When("using custom clientSecretKey mapping only", func() {
		It("should reconcile with custom clientSecret key", func() {
			By("creating a secret with custom clientSecret key")
			ctx := context.Background()

			secretName := fmt.Sprintf("custom-secret-secret-%s", rand.String(5))
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: "default",
				},
				StringData: map[string]string{
					"clientID":           "test-id",
					"customClientSecret": "test-secret",
				},
			}
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())

			By("creating an AuraInstance with custom clientSecretKey mapping")
			instanceName := fmt.Sprintf("test-custom-secret-%s", rand.String(5))
			instance := &v1beta1.AuraInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instanceName,
					Namespace: "default",
				},
				Spec: v1beta1.AuraInstanceSpec{
					Tier:          v1beta1.AuraInstanceTierFreeDb,
					Region:        "eu-west-1",
					CloudProvider: v1beta1.CloudProviderAWS,
					Neo4jVersion:  "5",
					TenantID:      fmt.Sprintf("tenant-%s", rand.String(5)),
					Secret: v1beta1.SecretReference{
						Name:            secretName,
						ClientSecretKey: "customClientSecret",
					},
				},
			}
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			By("verifying reconciliation succeeds with custom clientSecret key")
			instanceLookupKey := types.NamespacedName{Name: instanceName, Namespace: "default"}
			reconciledInstance := &v1beta1.AuraInstance{}

			// The reconciliation should succeed without "secret must contain" errors
			Eventually(func() bool {
				err := k8sClient.Get(ctx, instanceLookupKey, reconciledInstance)
				if err != nil {
					return false
				}
				// Check that we don't get key-related errors
				for _, condition := range reconciledInstance.Status.Conditions {
					if condition.Type == v1beta1.ConditionReady &&
						condition.Status == metav1.ConditionFalse &&
						strings.Contains(condition.Message, "secret must contain") {
						return false
					}
				}
				return true
			}, timeout, interval).Should(BeTrue())
		})
	})

	When("using both custom key mappings", func() {
		It("should reconcile with both custom keys", func() {
			By("creating a secret with both custom keys")
			ctx := context.Background()

			secretName := fmt.Sprintf("custom-both-secret-%s", rand.String(5))
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: "default",
				},
				StringData: map[string]string{
					"myClientId":     "test-id",
					"myClientSecret": "test-secret",
				},
			}
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())

			By("creating an AuraInstance with both custom key mappings")
			instanceName := fmt.Sprintf("test-custom-both-%s", rand.String(5))
			instance := &v1beta1.AuraInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instanceName,
					Namespace: "default",
				},
				Spec: v1beta1.AuraInstanceSpec{
					Tier:          v1beta1.AuraInstanceTierFreeDb,
					Region:        "ap-southeast-1",
					CloudProvider: v1beta1.CloudProviderGCP,
					Neo4jVersion:  "5",
					TenantID:      fmt.Sprintf("tenant-%s", rand.String(5)),
					Secret: v1beta1.SecretReference{
						Name:            secretName,
						ClientIDKey:     "myClientId",
						ClientSecretKey: "myClientSecret",
					},
				},
			}
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			By("verifying reconciliation succeeds with both custom keys")
			instanceLookupKey := types.NamespacedName{Name: instanceName, Namespace: "default"}
			reconciledInstance := &v1beta1.AuraInstance{}

			// The reconciliation should succeed without "secret must contain" errors
			Eventually(func() bool {
				err := k8sClient.Get(ctx, instanceLookupKey, reconciledInstance)
				if err != nil {
					return false
				}
				// Check that we don't get key-related errors
				for _, condition := range reconciledInstance.Status.Conditions {
					if condition.Type == v1beta1.ConditionReady &&
						condition.Status == metav1.ConditionFalse &&
						strings.Contains(condition.Message, "secret must contain") {
						return false
					}
				}
				return true
			}, timeout, interval).Should(BeTrue())
		})
	})

	When("custom key mapping points to non-existent key", func() {
		It("should fail with appropriate error message", func() {
			By("creating a secret without the custom key")
			ctx := context.Background()

			secretName := fmt.Sprintf("missing-key-secret-%s", rand.String(5))
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: "default",
				},
				StringData: map[string]string{
					"clientID":     "test-id",
					"clientSecret": "test-secret",
				},
			}
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())

			By("creating an AuraInstance with wrong custom key mapping")
			instanceName := fmt.Sprintf("test-wrong-key-%s", rand.String(5))
			instance := &v1beta1.AuraInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instanceName,
					Namespace: "default",
				},
				Spec: v1beta1.AuraInstanceSpec{
					Tier:          v1beta1.AuraInstanceTierFreeDb,
					Region:        "us-west-2",
					CloudProvider: v1beta1.CloudProviderAWS,
					Neo4jVersion:  "5",
					TenantID:      fmt.Sprintf("tenant-%s", rand.String(5)),
					Secret: v1beta1.SecretReference{
						Name:            secretName,
						ClientIDKey:     "wrongClientId", // key doesn't exist in the secret
						ClientSecretKey: "clientSecret",
					},
				},
			}
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			By("verifying reconciliation fails with key not found error")
			instanceLookupKey := types.NamespacedName{Name: instanceName, Namespace: "default"}
			reconciledInstance := &v1beta1.AuraInstance{}

			expectedStatus := &v1beta1.AuraInstanceStatus{
				ObservedGeneration: 1,
				Conditions: []metav1.Condition{
					{
						Type:    v1beta1.ConditionReady,
						Status:  metav1.ConditionFalse,
						Reason:  "ReconciliationFailed",
						Message: "secret must contain wrongClientId and clientSecret keys",
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
