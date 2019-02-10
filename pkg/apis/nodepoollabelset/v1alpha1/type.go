// Copyright © 2019 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodePoolLabelSet is a specification for a NodePoolLabelSet resource
type NodePoolLabelSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   NodePoolLabelSetSpec   `json:"spec"`
	Status NodePoolLabelSetStatus `json:"status,omitempty"`
}

// NodePoolLabelSetSpec is the spec for an NodePoolLabelSet resource
type NodePoolLabelSetSpec struct {
	Labels map[string]string `json:"labels"`
}

// NodePoolLabelSetStatus is the status for an NodePoolLabelSet resource
type NodePoolLabelSetStatus struct {
	State   NodePoolLabelSetState `json:"state,omitempty"`
	Message string                `json:"message,omitempty"`
}

type NodePoolLabelSetState string

const (
	NodePoolLabelSetStateCreated NodePoolLabelSetState = "Created"
	NodePoolLabelSetStateSyncing NodePoolLabelSetState = "Syncing"
	NodePoolLabelSetStateSynced  NodePoolLabelSetState = "Synced"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodePoolLabelSetList is a list of NodePoolLabelSet resources
type NodePoolLabelSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []NodePoolLabelSet `json:"items"`
}
