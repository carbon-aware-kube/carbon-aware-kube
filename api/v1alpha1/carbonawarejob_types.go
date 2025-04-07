/*
Copyright 2025.

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
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ConditionScheduled indicates that the CarbonAwareJob has been scheduled for execution
	ConditionScheduled string = "Scheduled"

	// ConditionStarted indicates that the underlying Job has been created and is running
	ConditionStarted string = "Started"

	// ConditionCompleted indicates that the Job has finished successfully
	ConditionCompleted string = "Completed"

	// ConditionFailed indicates that the Job has failed
	ConditionFailed string = "Failed"
)

const (
	ReasonForecastEvaluated = "ForecastEvaluated"
	ReasonWindowExpired     = "WindowExpired"
	ReasonJobCreated        = "JobCreated"
	ReasonJobFailed         = "JobFailed"
	ReasonJobSucceeded      = "JobSucceeded"
)

// CarbonAwareJobSpec defines the desired state of CarbonAwareJob.
type CarbonAwareJobSpec struct {
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=172800
	// StartFlexWindowSeconds defines the maximum number of seconds the controller can delay the start of the job, to allow for carbon-aware scheduling.
	StartFlexWindowSeconds *int32 `json:"startFlexWindowSeconds,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=86400
	// ExpectedDurationSeconds defines the expected duration of the job.
	ExpectedDurationSeconds *int32 `json:"expectedDurationSeconds,omitempty"`

	// +kubebuilder:validation:Required
	// JobTemplate is the template for the job to be run.
	JobTemplate batchv1.JobTemplateSpec `json:"jobTemplate"`
}

// CarbonAwareJobStatus defines the observed state of CarbonAwareJob.
type CarbonAwareJobStatus struct {
	// ScheduledStartTime is the chosen execution time based on carbon forecast
	ScheduledStartTime *metav1.Time `json:"scheduledStartTime,omitempty"`

	// Conditions describe the current state of the CAJob
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// JobRef holds a reference to the Job that was created
	JobRef *JobReference `json:"jobRef,omitempty"`
}

// JobReference identifies the real Job created by the controller
type JobReference struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// CarbonAwareJob is the Schema for the carbonawarejobs API.
type CarbonAwareJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CarbonAwareJobSpec   `json:"spec,omitempty"`
	Status CarbonAwareJobStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CarbonAwareJobList contains a list of CarbonAwareJob.
type CarbonAwareJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CarbonAwareJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CarbonAwareJob{}, &CarbonAwareJobList{})
}
