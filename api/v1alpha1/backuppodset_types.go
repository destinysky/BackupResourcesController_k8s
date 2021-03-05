/*
Copyright 2021.

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

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BackupPodSetSpec defines the desired state of BackupPodSet
type BackupPodSetSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of BackupPodSet. Edit BackupPodSet_types.go to remove/update
	//Selector    metav1.LabelSelector `json:"selector,omitempty"`
	Replicas      int32              `json:"replicas"`
	HotBackups    int32              `json:"hotBackups"`
	ColdBackups   int32              `json:"coldBackups"`
	Strategy      Strategy           `json:"strategy"`
	SchedulerName string             `json:"schedulerName"`
	WakeupTimeout int32              `json:"wakeupTimeout"`
	Template      v1.PodTemplateSpec `json:"template"`
}

// BackupPodSetStatus defines the observed state of BackupPodSet
type BackupPodSetStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Replicas           int32    `json:"replicas"`
	PrimaryPodNames    []string `json:"primaryPodNames"`
	HotBackups         int32    `json:"hotbackups"`
	HotBackupPodNames  []string `json:"hotbackupPodNames"`
	ColdBackups        int32    `json:"coldbackups"`
	ColdBackupPodNames []string `json:"coldbackupPodNames"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// BackupPodSet is the Schema for the backuppodsets API
type BackupPodSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BackupPodSetSpec   `json:"spec,omitempty"`
	Status BackupPodSetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BackupPodSetList contains a list of BackupPodSet
type BackupPodSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BackupPodSet `json:"items"`
}

type Strategy struct {
	Init                    string `json:"init"`
	UnallocatedToColdbackup string `json:"unallocatedToColdbackup"`
	UnallocatedToHotbackup  string `json:"unallocatedToHotbackup"`
	UnallocatedToReplicas   string `json:"unallocatedToReplicas"`

	ColdBackupToHotBackup string `json:"coldbackupToHotbackup"`
	ColdBackupToReplicas  string `json:"coldbackupToReplicas"`
	HotBackupToReplicas   string `json:"hotbackupToReplicas"`
}

func init() {
	SchemeBuilder.Register(&BackupPodSet{}, &BackupPodSetList{})
}
