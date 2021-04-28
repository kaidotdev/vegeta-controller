package v1

import (
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AttackSpec defines the desired state of Attack
type AttackSpec struct {
	// Parallelism of Attack
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=1
	Parallelism int32 `json:"parallelism,omitempty"`
	// Scenario of Attack
	// More info: https://github.com/tsenart/vegeta#http-format
	Scenario string `json:"scenario"`
	// +kubebuilder:validation:Enum=text;json
	// +kubebuilder:default=text
	Output              string              `json:"output,omitempty"`
	Option              VegetaOption        `json:"option,omitempty"`
	Template            Template            `json:"template,omitempty"`
	AttackContainerSpec AttackContainerSpec `json:"attackContainerSpec,omitempty"`
}

// Additional Spec for attack container.
type AttackContainerSpec struct {
	// Compute Resources required by this container.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/
	Resources v1.ResourceRequirements `json:"resources,omitempty" protobuf:"bytes,8,opt,name=resources"`
}

// AttackStatus defines the observed state of Attack
type AttackStatus struct{}

// VegetaOption defines the vegeta options
type VegetaOption struct {
	// Duration of the test [0 = forever]
	// More info: https://github.com/tsenart/vegeta#usage-manual
	// +kubebuilder:validation:Pattern=^\d+s$
	// +kubebuilder:default="10s"
	Duration string `json:"duration,omitempty"`
	// Max open idle connections per target host (default 10000)
	// More info: https://github.com/tsenart/vegeta#usage-manual
	// +kubebuilder:validation:Minimum=1
	Connections int `json:"connections,omitempty"`
	// Number of requests per time unit [0 = infinity] (default 50/1s)
	// More info: https://github.com/tsenart/vegeta#usage-manual
	// +kubebuilder:validation:Minimum=1
	Rate int `json:"rate,omitempty"`
	// Requests timeout (default 30s)
	// More info: https://github.com/tsenart/vegeta#usage-manual
	// +kubebuilder:validation:Pattern=^\d+s$
	Timeout string `json:"timeout,omitempty"`
	// Initial number of workers (default 10)
	// More info: https://github.com/tsenart/vegeta#usage-manual
	// +kubebuilder:validation:Minimum=1
	Workers int `json:"workers,omitempty"`
	// Targets format [http, json] (default "http")
	// More info: https://github.com/tsenart/vegeta#usage-manual
	// +kubebuilder:validation:Enum=http;json
	Format string `json:"format,omitempty"`
}

// Template defines the pod template generated by job
type Template struct {
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +kubebuilder:pruning:PreserveUnknownFields
	metaV1.ObjectMeta `json:"metadata,omitempty"`
	Spec              Spec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

// Spec defines the additional pod spec generated by job
type Spec struct {
	// HostAliases is an optional list of hosts and IPs that will be injected into the pod's hosts
	// file if specified. This is only valid for non-hostNetwork pods.
	HostAliases []v1.HostAlias `json:"hostAliases,omitempty" patchStrategy:"merge" patchMergeKey:"ip" protobuf:"bytes,23,rep,name=hostAliases"`
}

// +kubebuilder:object:root=true

// Attack is the schema for the attacks API
type Attack struct {
	metaV1.TypeMeta   `json:",inline"`
	metaV1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AttackSpec   `json:"spec,omitempty"`
	Status AttackStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AttackList contains a list of Attack
type AttackList struct {
	metaV1.TypeMeta `json:",inline"`
	metaV1.ListMeta `json:"metadata,omitempty"`
	Items           []Attack `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Attack{}, &AttackList{})
}
