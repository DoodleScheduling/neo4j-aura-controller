/*
Copyright 2022 Doodle.

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

package controllers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"

	infrav1beta1 "github.com/doodlescheduling/neo4j-aura-controller/api/v1beta1"
	auraclient "github.com/doodlescheduling/neo4j-aura-controller/pkg/aura/client"
	"github.com/fluxcd/pkg/runtime/conditions"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

//+kubebuilder:rbac:groups=neo4j.infra.doodle.com,resources=aurainstances,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=neo4j.infra.doodle.com,resources=aurainstances/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=neo4j.infra.doodle.com,resources=aurainstances/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;delete;patch;update

// AuraInstanceReconciler reconciles an AuraInstance object
type AuraInstanceReconciler struct {
	client.Client
	TokenURL   string
	BaseURL    string
	HTTPClient *http.Client
	Log        logr.Logger
	Recorder   record.EventRecorder
}

type AuraInstanceReconcilerOptions struct {
	MaxConcurrentReconciles int
}

// SetupWithManager sets up the controller with the Manager.
func (r *AuraInstanceReconciler) SetupWithManager(mgr ctrl.Manager, opts AuraInstanceReconcilerOptions) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1beta1.AuraInstance{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: opts.MaxConcurrentReconciles}).
		Complete(r)
}

func (r *AuraInstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("namespace", req.Namespace, "name", req.Name)

	instance := infrav1beta1.AuraInstance{}
	err := r.Get(ctx, req.NamespacedName, &instance)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if instance.Spec.Suspend {
		logger.Info("aura instance is suspended")
		return ctrl.Result{}, nil
	}

	logger.Info("reconciling aura instance")
	instance, result, err := r.reconcile(ctx, instance, logger)
	instance.Status.ObservedGeneration = instance.GetGeneration()

	if err != nil {
		logger.Error(err, "reconcile error occurred")
		instance = infrav1beta1.AuraInstanceReady(instance, metav1.ConditionFalse, "ReconciliationFailed", err.Error())
		r.Recorder.Event(&instance, "Warning", "ReconciliationFailed", err.Error())
	}

	// Update status after reconciliation
	if err := r.patchStatus(ctx, &instance); err != nil {
		logger.Error(err, "unable to update status after reconciliation")
		return ctrl.Result{Requeue: true}, err
	}

	if err == nil && instance.Spec.Interval != nil {
		result.RequeueAfter = instance.Spec.Interval.Duration
	}

	return result, err
}

func (r *AuraInstanceReconciler) httpClient(ctx context.Context, instance infrav1beta1.AuraInstance) (*http.Client, error) {
	var secret corev1.Secret
	if err := r.Get(ctx, types.NamespacedName{
		Name:      instance.Spec.Secret.Name,
		Namespace: instance.Namespace,
	}, &secret); err != nil {
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}
	clientIDKey := instance.Spec.Secret.ClientIDKey
	if clientIDKey == "" {
		clientIDKey = "clientID"
	}
	clientSecretKey := instance.Spec.Secret.ClientSecretKey
	if clientSecretKey == "" {
		clientSecretKey = "clientSecret"
	}
	clientID := string(secret.Data[clientIDKey])
	clientSecret := string(secret.Data[clientSecretKey])
	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("secret must contain %s and %s keys", clientIDKey, clientSecretKey)
	}
	conf := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     r.TokenURL,
	}

	ctx = context.WithValue(ctx, oauth2.HTTPClient, r.HTTPClient)
	tokenSource := conf.TokenSource(ctx)
	transport := &oauth2.Transport{
		Source: tokenSource,
		Base:   r.HTTPClient.Transport,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   r.HTTPClient.Timeout,
	}, nil
}

func (r *AuraInstanceReconciler) reconcile(ctx context.Context, instance infrav1beta1.AuraInstance, logger logr.Logger) (infrav1beta1.AuraInstance, ctrl.Result, error) {
	httpClient, err := r.httpClient(ctx, instance)
	if err != nil {
		return instance, reconcile.Result{}, err
	}

	auraClient, err := auraclient.NewClientWithResponses(r.BaseURL, auraclient.WithHTTPClient(httpClient))
	if err != nil {
		return instance, reconcile.Result{}, fmt.Errorf("failed to create aura client: %w", err)
	}

	connectionSecretName := fmt.Sprintf("%s-connection", instance.Name)
	if instance.Spec.ConnectionSecret.Name != "" {
		connectionSecretName = instance.Spec.ConnectionSecret.Name
	}

	if instance.Status.InstanceID != "" {
		auraInstance, err := auraClient.GetInstanceIdWithResponse(ctx, instance.Status.InstanceID)
		if err != nil {
			logger.Error(err, "failed to get Aura instance")
			return instance, reconcile.Result{}, err
		}

		if auraInstance.StatusCode() == http.StatusNotFound {
			var secret corev1.Secret
			err := r.Get(ctx, types.NamespacedName{
				Name:      connectionSecretName,
				Namespace: instance.Namespace,
			}, &secret)

			if err != nil && !kerrors.IsNotFound(err) {
				return instance, reconcile.Result{}, fmt.Errorf("failed to get secret: %w", err)
			} else if err == nil {
				if len(secret.OwnerReferences) > 0 {
					if secret.OwnerReferences[0].UID != instance.UID {
						return instance, reconcile.Result{}, fmt.Errorf("failed to delete secret, owner uid %s does not match", secret.OwnerReferences[0].UID)
					}
				}

				if err := r.Delete(ctx, &secret); err != nil {
					return instance, reconcile.Result{}, fmt.Errorf("failed to delete secret: %w", err)
				}
			}

			instance.Status.InstanceID = ""
			instance.Status.ConnectionSecret = ""

			return instance, reconcile.Result{Requeue: true}, nil
		}

		if auraInstance.StatusCode() != http.StatusOK {
			return instance, reconcile.Result{}, fmt.Errorf("failed to get instance, request failed with code %d - %s", auraInstance.StatusCode(), auraInstance.Body)
		}

		instance.Status.InstanceStatus = string(auraInstance.JSON200.Data.Status)

		switch auraInstance.JSON200.Data.Status {
		case auraclient.InstanceDataStatusRunning:
			conditions.Delete(&instance, infrav1beta1.ConditionReconciling)
			instance = infrav1beta1.AuraInstanceReady(instance, metav1.ConditionTrue, "InstanceRunning", "Instance is running")
		case auraclient.InstanceDataStatusCreating:
			instance = infrav1beta1.AuraInstanceReconciling(instance, metav1.ConditionTrue, "InstanceCreating", "Instance is being created")
			return instance, reconcile.Result{RequeueAfter: time.Second * 30}, nil
		default:
			instance = infrav1beta1.AuraInstanceReady(instance, metav1.ConditionFalse, "InstanceNotReady", fmt.Sprintf("Instance status: %s", instance.Status.InstanceStatus))
		}

		if instance.Spec.Memory != string(auraInstance.JSON200.Data.Memory) ||
			instance.Spec.GraphAnalyticsPlugin != *auraInstance.JSON200.Data.GraphAnalyticsPlugin ||
			instance.Spec.VectorOptimized != *auraInstance.JSON200.Data.VectorOptimized {
			logger.Info("updating aura instance")
			instance = infrav1beta1.AuraInstanceReconciling(instance, metav1.ConditionTrue, "UpdatingInstance", "Updating Aura instance")

			memory := auraclient.InstanceMemory(instance.Spec.Memory)

			patchReq := auraclient.PatchInstanceIdJSONRequestBody{
				GraphAnalyticsPlugin: &instance.Spec.GraphAnalyticsPlugin,
				Memory:               &memory,
				VectorOptimized:      &instance.Spec.VectorOptimized,
			}

			auraInstance, err := auraClient.PatchInstanceIdWithResponse(ctx, instance.Status.InstanceID, patchReq)
			if err != nil {
				return instance, reconcile.Result{}, fmt.Errorf("failed to update instance: %w", err)
			}

			if auraInstance.StatusCode() != http.StatusAccepted {
				return instance, reconcile.Result{}, fmt.Errorf("failed to update instance, request failed with code %d - %s", auraInstance.StatusCode(), auraInstance.Body)
			}
		}

		return instance, reconcile.Result{}, nil
	}

	params := auraclient.GetInstancesParams{
		TenantId: &instance.Spec.TenantID,
	}

	auraInstances, err := auraClient.GetInstancesWithResponse(ctx, &params)
	if err != nil {
		logger.Error(err, "failed to get aura instance")
		return instance, reconcile.Result{}, err
	}

	if auraInstances.StatusCode() != http.StatusOK {
		return instance, reconcile.Result{}, fmt.Errorf("failed to get instance list, request failed with code %d - %s", auraInstances.StatusCode(), auraInstances.Body)
	}

	for _, remoteInstance := range auraInstances.JSON200.Data {
		if instance.Name == remoteInstance.Name {
			instance.Status.InstanceID = remoteInstance.Id
			return instance, reconcile.Result{Requeue: true}, nil
		}
	}

	logger.Info("creating new aura instance")
	instance = infrav1beta1.AuraInstanceReconciling(instance, metav1.ConditionTrue, "CreatingInstance", "Creating new Aura instance")

	createReq := auraclient.PostInstancesJSONBody{
		CloudProvider:        auraclient.CloudProvider(instance.Spec.CloudProvider),
		Memory:               auraclient.InstanceMemory(instance.Spec.Memory),
		Name:                 instance.Name,
		Region:               auraclient.InstanceRegion(instance.Spec.Region),
		TenantId:             instance.Spec.TenantID,
		Type:                 auraclient.InstanceType(instance.Spec.Tier),
		Version:              auraclient.InstanceVersion(instance.Spec.Neo4jVersion),
		VectorOptimized:      &instance.Spec.VectorOptimized,
		GraphAnalyticsPlugin: &instance.Spec.GraphAnalyticsPlugin,
	}

	auraInstance, err := auraClient.PostInstancesWithResponse(ctx, auraclient.PostInstancesJSONRequestBody(createReq))
	if err != nil {
		return instance, reconcile.Result{}, fmt.Errorf("failed to create the instance: %w", err)
	}

	if auraInstance.StatusCode() != http.StatusAccepted {
		return instance, reconcile.Result{}, fmt.Errorf("failed to create the instance, request failed with code %d - %s", auraInstance.StatusCode(), auraInstance.Body)
	}

	instance.Status.InstanceID = auraInstance.JSON202.Data.Id
	instance.Status.ConnectionSecret = connectionSecretName

	connectionDetails := corev1.Secret{
		StringData: map[string]string{
			"username":      auraInstance.JSON202.Data.Username,
			"password":      auraInstance.JSON202.Data.Password,
			"connectionURL": auraInstance.JSON202.Data.ConnectionUrl,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      connectionSecretName,
			Namespace: instance.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: instance.APIVersion,
					Kind:       instance.Kind,
					Name:       instance.Name,
					UID:        instance.UID,
				},
			},
		},
	}

	if err := r.Create(ctx, &connectionDetails); err != nil {
		return instance, reconcile.Result{}, fmt.Errorf("failed to create connection secret: %w", err)
	}

	r.Recorder.Event(&instance, "Normal", "InstanceCreated", fmt.Sprintf("Created aura instance %q", instance.Status.InstanceID))
	return instance, reconcile.Result{RequeueAfter: time.Second * 30}, nil
}

func (r *AuraInstanceReconciler) patchStatus(ctx context.Context, instance *infrav1beta1.AuraInstance) error {
	key := client.ObjectKeyFromObject(instance)
	latest := &infrav1beta1.AuraInstance{}
	if err := r.Get(ctx, key, latest); err != nil {
		return err
	}

	return r.Status().Patch(ctx, instance, client.MergeFrom(latest))
}
