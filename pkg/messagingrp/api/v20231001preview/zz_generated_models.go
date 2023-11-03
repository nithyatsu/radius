//go:build go1.18
// +build go1.18

// Licensed under the Apache License, Version 2.0 . See LICENSE in the repository root for license information.
// Code generated by Microsoft (R) AutoRest Code Generator. DO NOT EDIT.
// Changes may cause incorrect behavior and will be lost if the code is regenerated.

package v20231001preview

import "time"

// EnvironmentCompute - Represents backing compute resource
type EnvironmentCompute struct {
	// REQUIRED; Discriminator property for EnvironmentCompute.
	Kind *string

	// Configuration for supported external identity providers
	Identity *IdentitySettings

	// The resource id of the compute resource for application environment.
	ResourceID *string
}

// GetEnvironmentCompute implements the EnvironmentComputeClassification interface for type EnvironmentCompute.
func (e *EnvironmentCompute) GetEnvironmentCompute() *EnvironmentCompute { return e }

// ErrorAdditionalInfo - The resource management error additional info.
type ErrorAdditionalInfo struct {
	// READ-ONLY; The additional info.
	Info map[string]any

	// READ-ONLY; The additional info type.
	Type *string
}

// ErrorDetail - The error detail.
type ErrorDetail struct {
	// READ-ONLY; The error additional info.
	AdditionalInfo []*ErrorAdditionalInfo

	// READ-ONLY; The error code.
	Code *string

	// READ-ONLY; The error details.
	Details []*ErrorDetail

	// READ-ONLY; The error message.
	Message *string

	// READ-ONLY; The error target.
	Target *string
}

// ErrorResponse - Common error response for all Azure Resource Manager APIs to return error details for failed operations.
// (This also follows the OData error response format.).
type ErrorResponse struct {
	// The error object.
	Error *ErrorDetail
}

// IdentitySettings is the external identity setting.
type IdentitySettings struct {
	// REQUIRED; kind of identity setting
	Kind *IdentitySettingKind

	// The URI for your compute platform's OIDC issuer
	OidcIssuer *string

	// The resource ID of the provisioned identity
	Resource *string
}

// KubernetesCompute - The Kubernetes compute configuration
type KubernetesCompute struct {
	// REQUIRED; Discriminator property for EnvironmentCompute.
	Kind *string

	// REQUIRED; The namespace to use for the environment.
	Namespace *string

	// Configuration for supported external identity providers
	Identity *IdentitySettings

	// The resource id of the compute resource for application environment.
	ResourceID *string
}

// GetEnvironmentCompute implements the EnvironmentComputeClassification interface for type KubernetesCompute.
func (k *KubernetesCompute) GetEnvironmentCompute() *EnvironmentCompute {
	return &EnvironmentCompute{
		Identity: k.Identity,
		Kind: k.Kind,
		ResourceID: k.ResourceID,
	}
}

// Operation - Details of a REST API operation, returned from the Resource Provider Operations API
type Operation struct {
	// Localized display information for this particular operation.
	Display *OperationDisplay

	// READ-ONLY; Enum. Indicates the action type. "Internal" refers to actions that are for internal only APIs.
	ActionType *ActionType

	// READ-ONLY; Whether the operation applies to data-plane. This is "true" for data-plane operations and "false" for ARM/control-plane
// operations.
	IsDataAction *bool

	// READ-ONLY; The name of the operation, as per Resource-Based Access Control (RBAC). Examples: "Microsoft.Compute/virtualMachines/write",
// "Microsoft.Compute/virtualMachines/capture/action"
	Name *string

	// READ-ONLY; The intended executor of the operation; as in Resource Based Access Control (RBAC) and audit logs UX. Default
// value is "user,system"
	Origin *Origin
}

// OperationDisplay - Localized display information for this particular operation.
type OperationDisplay struct {
	// READ-ONLY; The short, localized friendly description of the operation; suitable for tool tips and detailed views.
	Description *string

	// READ-ONLY; The concise, localized friendly name for the operation; suitable for dropdowns. E.g. "Create or Update Virtual
// Machine", "Restart Virtual Machine".
	Operation *string

	// READ-ONLY; The localized friendly form of the resource provider name, e.g. "Microsoft Monitoring Insights" or "Microsoft
// Compute".
	Provider *string

	// READ-ONLY; The localized friendly name of the resource type related to this operation. E.g. "Virtual Machines" or "Job
// Schedule Collections".
	Resource *string
}

// OperationListResult - A list of REST API operations supported by an Azure Resource Provider. It contains an URL link to
// get the next set of results.
type OperationListResult struct {
	// READ-ONLY; URL to get the next set of operation list results (if there are any).
	NextLink *string

	// READ-ONLY; List of operations supported by the resource provider
	Value []*Operation
}

// OutputResource - Properties of an output resource.
type OutputResource struct {
	// The UCP resource ID of the underlying resource.
	ID *string

	// The logical identifier scoped to the owning Radius resource. This is only needed or used when a resource has a dependency
// relationship. LocalIDs do not have any particular format or meaning beyond
// being compared to determine dependency relationships.
	LocalID *string

	// Determines whether Radius manages the lifecycle of the underlying resource.
	RadiusManaged *bool
}

// RabbitMQListSecretsResult - The secret values for the given RabbitMQQueue resource
type RabbitMQListSecretsResult struct {
	// The password used to connect to the RabbitMQ instance
	Password *string

	// The connection URI of the RabbitMQ instance. Generated automatically from host, port, SSL, username, password, and vhost.
// Can be overridden with a custom value
	URI *string
}

// RabbitMQQueueProperties - RabbitMQQueue portable resource properties
type RabbitMQQueueProperties struct {
	// REQUIRED; Fully qualified resource ID for the environment that the portable resource is linked to
	Environment *string

	// Fully qualified resource ID for the application that the portable resource is consumed by (if applicable)
	Application *string

	// The hostname of the RabbitMQ instance
	Host *string

	// The port of the RabbitMQ instance. Defaults to 5672
	Port *int32

	// The name of the queue
	Queue *string

	// The recipe used to automatically deploy underlying infrastructure for the resource
	Recipe *Recipe

	// Specifies how the underlying service/resource is provisioned and managed.
	ResourceProvisioning *ResourceProvisioning

	// List of the resource IDs that support the rabbitMQ resource
	Resources []*ResourceReference

	// The secrets to connect to the RabbitMQ instance
	Secrets *RabbitMQSecrets

	// Specifies whether to use SSL when connecting to the RabbitMQ instance
	TLS *bool

	// The username to use when connecting to the RabbitMQ instance
	Username *string

	// The RabbitMQ virtual host (vHost) the client will connect to. Defaults to no vHost.
	VHost *string

	// READ-ONLY; The status of the asynchronous operation.
	ProvisioningState *ProvisioningState

	// READ-ONLY; Status of a resource.
	Status *ResourceStatus
}

// RabbitMQQueueResource - RabbitMQQueue portable resource
type RabbitMQQueueResource struct {
	// REQUIRED; The geo-location where the resource lives
	Location *string

	// The resource-specific properties for this resource.
	Properties *RabbitMQQueueProperties

	// Resource tags.
	Tags map[string]*string

	// READ-ONLY; Fully qualified resource ID for the resource. Ex - /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}
	ID *string

	// READ-ONLY; The name of the resource
	Name *string

	// READ-ONLY; Azure Resource Manager metadata containing createdBy and modifiedBy information.
	SystemData *SystemData

	// READ-ONLY; The type of the resource. E.g. "Microsoft.Compute/virtualMachines" or "Microsoft.Storage/storageAccounts"
	Type *string
}

// RabbitMQQueueResourceListResult - The response of a RabbitMQQueueResource list operation.
type RabbitMQQueueResourceListResult struct {
	// REQUIRED; The RabbitMQQueueResource items on this page
	Value []*RabbitMQQueueResource

	// The link to the next page of items
	NextLink *string
}

// RabbitMQQueueResourceUpdate - The type used for update operations of the RabbitMQQueueResource.
type RabbitMQQueueResourceUpdate struct {
	// The updatable properties of the RabbitMQQueueResource.
	Properties *RabbitMQQueueResourceUpdateProperties

	// Resource tags.
	Tags map[string]*string
}

// RabbitMQQueueResourceUpdateProperties - The updatable properties of the RabbitMQQueueResource.
type RabbitMQQueueResourceUpdateProperties struct {
	// Fully qualified resource ID for the application that the portable resource is consumed by (if applicable)
	Application *string

	// Fully qualified resource ID for the environment that the portable resource is linked to
	Environment *string

	// The hostname of the RabbitMQ instance
	Host *string

	// The port of the RabbitMQ instance. Defaults to 5672
	Port *int32

	// The name of the queue
	Queue *string

	// The recipe used to automatically deploy underlying infrastructure for the resource
	Recipe *RecipeUpdate

	// Specifies how the underlying service/resource is provisioned and managed.
	ResourceProvisioning *ResourceProvisioning

	// List of the resource IDs that support the rabbitMQ resource
	Resources []*ResourceReference

	// The secrets to connect to the RabbitMQ instance
	Secrets *RabbitMQSecrets

	// Specifies whether to use SSL when connecting to the RabbitMQ instance
	TLS *bool

	// The username to use when connecting to the RabbitMQ instance
	Username *string

	// The RabbitMQ virtual host (vHost) the client will connect to. Defaults to no vHost.
	VHost *string
}

// RabbitMQSecrets - The connection secrets properties to the RabbitMQ instance
type RabbitMQSecrets struct {
	// The password used to connect to the RabbitMQ instance
	Password *string

	// The connection URI of the RabbitMQ instance. Generated automatically from host, port, SSL, username, password, and vhost.
// Can be overridden with a custom value
	URI *string
}

// Recipe - The recipe used to automatically deploy underlying infrastructure for a portable resource
type Recipe struct {
	// REQUIRED; The name of the recipe within the environment to use
	Name *string

	// Key/value parameters to pass into the recipe at deployment
	Parameters map[string]any
}

// RecipeStatus - Recipe status at deployment time for a resource.
type RecipeStatus struct {
	// REQUIRED; TemplateKind is the kind of the recipe template used by the portable resource upon deployment.
	TemplateKind *string

	// REQUIRED; TemplatePath is the path of the recipe consumed by the portable resource upon deployment.
	TemplatePath *string

	// TemplateVersion is the version number of the template.
	TemplateVersion *string
}

// RecipeUpdate - The recipe used to automatically deploy underlying infrastructure for a portable resource
type RecipeUpdate struct {
	// The name of the recipe within the environment to use
	Name *string

	// Key/value parameters to pass into the recipe at deployment
	Parameters map[string]any
}

// Resource - Common fields that are returned in the response for all Azure Resource Manager resources
type Resource struct {
	// READ-ONLY; Fully qualified resource ID for the resource. Ex - /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}
	ID *string

	// READ-ONLY; The name of the resource
	Name *string

	// READ-ONLY; Azure Resource Manager metadata containing createdBy and modifiedBy information.
	SystemData *SystemData

	// READ-ONLY; The type of the resource. E.g. "Microsoft.Compute/virtualMachines" or "Microsoft.Storage/storageAccounts"
	Type *string
}

// ResourceReference - Describes a reference to an existing resource
type ResourceReference struct {
	// REQUIRED; Resource id of an existing resource
	ID *string
}

// ResourceStatus - Status of a resource.
type ResourceStatus struct {
	// The compute resource associated with the resource.
	Compute EnvironmentComputeClassification

	// Properties of an output resource
	OutputResources []*OutputResource

	// READ-ONLY; The recipe data at the time of deployment
	Recipe *RecipeStatus
}

// SystemData - Metadata pertaining to creation and last modification of the resource.
type SystemData struct {
	// The timestamp of resource creation (UTC).
	CreatedAt *time.Time

	// The identity that created the resource.
	CreatedBy *string

	// The type of identity that created the resource.
	CreatedByType *CreatedByType

	// The timestamp of resource last modification (UTC)
	LastModifiedAt *time.Time

	// The identity that last modified the resource.
	LastModifiedBy *string

	// The type of identity that last modified the resource.
	LastModifiedByType *CreatedByType
}

// TrackedResource - The resource model definition for an Azure Resource Manager tracked top level resource which has 'tags'
// and a 'location'
type TrackedResource struct {
	// REQUIRED; The geo-location where the resource lives
	Location *string

	// Resource tags.
	Tags map[string]*string

	// READ-ONLY; Fully qualified resource ID for the resource. Ex - /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/{resourceProviderNamespace}/{resourceType}/{resourceName}
	ID *string

	// READ-ONLY; The name of the resource
	Name *string

	// READ-ONLY; Azure Resource Manager metadata containing createdBy and modifiedBy information.
	SystemData *SystemData

	// READ-ONLY; The type of the resource. E.g. "Microsoft.Compute/virtualMachines" or "Microsoft.Storage/storageAccounts"
	Type *string
}

