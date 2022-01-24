/*
Copyright 2022 kubetrail.io authors.

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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TokenSpec defines the desired state of Token
type TokenSpec struct {
	Permissions []Permission `json:"permissions,omitempty"`
	SecretName  string       `json:"secretName,omitempty"`
	ConfigName  string       `json:"configName,omitempty"`
}

// Permission defines permission for an asset in influxdb
type Permission struct {
	ResourceName   string `json:"resourceName,omitempty"`
	ResourceType   string `json:"resourceType,omitempty"`
	PermissionType string `json:"permissionType,omitempty"`
}

// TokenStatus defines the observed state of Token
type TokenStatus struct {
	Phase      string             `json:"phase,omitempty"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	Message    string             `json:"message,omitempty"`
	Reason     string             `json:"reason,omitempty"`
	Data       map[string]string  `json:"data,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="Status of token"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Token is the Schema for the tokens API
type Token struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TokenSpec   `json:"spec,omitempty"`
	Status TokenStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TokenList contains a list of Token
type TokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Token `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Token{}, &TokenList{})
}
