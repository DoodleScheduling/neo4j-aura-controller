package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AuraInstanceTier represents the Neo4j Aura instance tier
// +kubebuilder:validation:Enum=free-db;professional-db;business-critical;enterprise-db
type AuraInstanceTier string

const (
	AuraInstanceTierFreeDb           AuraInstanceTier = "free-db"
	AuraInstanceTierProfessionalDb   AuraInstanceTier = "professional-db"
	AuraInstanceTierBusinessCritical AuraInstanceTier = "business-critical"
	AuraInstanceTierEnterpriseDb     AuraInstanceTier = "enterprise-db"
)

// CloudProvider represents supported cloud providers
// +kubebuilder:validation:Enum=aws;gcp;azure
type CloudProvider string

const (
	CloudProviderAWS   CloudProvider = "aws"
	CloudProviderGCP   CloudProvider = "gcp"
	CloudProviderAzure CloudProvider = "azure"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type AuraInstance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AuraInstanceSpec   `json:"spec,omitempty"`
	Status AuraInstanceStatus `json:"status,omitempty"`
}

type AuraInstanceSpec struct {
	// Tier specifies the Neo4j Aura instance tier
	// +kubebuilder:validation:Required
	Tier AuraInstanceTier `json:"tier"`

	// Region specifies the cloud region for the instance
	// +kubebuilder:validation:Required
	Region string `json:"region"`

	// CloudProvider specifies the cloud provider
	// +kubebuilder:validation:Required
	CloudProvider CloudProvider `json:"cloudProvider"`

	// Memory specifies the memory allocation (e.g., "1GB", "8GB", "16GB")
	// +optional
	Memory string `json:"memory,omitempty"`

	// Neo4jVersion specifies the Neo4j version
	// +kubebuilder:validation:Required
	Neo4jVersion string `json:"neo4jVersion"`

	// TenantID specifies the Aura tenant/project ID
	// +kubebuilder:validation:Required
	TenantID string `json:"tenantID"`

	// Secret is a reference to a secret containing Aura API credentials
	// By default expects keys: clientID, clientSecret
	// Use clientIDKey and clientSecretKey fields to override the default keys
	Secret SecretReference `json:"secret"`

	// ConnectionSecret is a reference to a secret which will contain the connection details.
	// By default this will be ${metadataname}-connection
	ConnectionSecret LocalObjectReference `json:"connectionSecret,omitempty"`

	// VectorOptimized specifies the vector optimization configuration of the instance
	// +optional
	VectorOptimized bool `json:"vectorOptimized,omitempty"`

	// GraphAnalyticsPlugin specifies the graph analytics plugin configuration of the instance
	// +optional
	GraphAnalyticsPlugin bool `json:"graphAnalyticsPlugin,omitempty"`

	// Suspend tells the controller to suspend reconciliation for this instance
	// +optional
	Suspend bool `json:"suspend,omitempty"`

	// Timeout used for upstream http requests
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// Interval at which the controller should reconcile the instance
	// +optional
	Interval *metav1.Duration `json:"interval,omitempty"`
}

type AuraInstanceStatus struct {
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration is the last generation reconciled by the controller
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// InstanceID is the Aura instance ID
	// +optional
	InstanceID string `json:"instanceId,omitempty"`

	// ConnectionSecret is the secret name which contains the connection details
	// +optional
	ConnectionSecret string `json:"connectionUri,omitempty"`

	// Status represents the current status of the Aura instance
	// +optional
	InstanceStatus string `json:"instanceStatus,omitempty"`
}

// AuraInstanceList contains a list of AuraInstance.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AuraInstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AuraInstance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AuraInstance{}, &AuraInstanceList{})
}

func AuraInstanceReconciling(set AuraInstance, status metav1.ConditionStatus, reason, message string) AuraInstance {
	setResourceCondition(&set, ConditionReconciling, status, reason, message, set.Generation)
	return set
}

func AuraInstanceReady(set AuraInstance, status metav1.ConditionStatus, reason, message string) AuraInstance {
	setResourceCondition(&set, ConditionReady, status, reason, message, set.Generation)
	return set
}

// GetStatusConditions returns a pointer to the Status.Conditions slice
func (in *AuraInstance) GetStatusConditions() *[]metav1.Condition {
	return &in.Status.Conditions
}

func (in *AuraInstance) GetConditions() []metav1.Condition {
	return in.Status.Conditions
}

func (in *AuraInstance) SetConditions(conditions []metav1.Condition) {
	in.Status.Conditions = conditions
}
