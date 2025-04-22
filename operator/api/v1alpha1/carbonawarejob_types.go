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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CarbonAwareJobSpec defines the desired state of CarbonAwareJob
type CarbonAwareJobSpec struct {
	// JobTemplate is the job template that will be created when the carbon intensity is optimal
	// This follows the same structure as the Kubernetes Job template
	// +kubebuilder:validation:Required
	JobTemplate JobTemplateSpec `json:"jobTemplate"`

	// MaxDelay defines the maximum time to delay the job execution from submission time
	// The controller will schedule the job at the optimal time within this delay window
	// based on carbon intensity forecasts
	// +kubebuilder:validation:Required
	MaxDelay metav1.Duration `json:"maxDelay"`

	// MaxDuration is the maximum duration the job is expected to run
	// This is used to calculate the optimal start time to minimize carbon emissions
	// +optional
	MaxDuration *metav1.Duration `json:"maxDuration,omitempty"`

	// Location specifies the geographical location where the job will run
	// This is used to determine the carbon intensity for that region
	// If not specified, the controller will use the location of the Kubernetes cluster
	// +optional
	Location string `json:"location,omitempty"`
}

// JobTemplateSpec is a subset of the Kubernetes batch/v1.JobTemplateSpec
// It defines the template for the job that will be created
type JobTemplateSpec struct {
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	Metadata metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of the job.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	Spec batchv1.JobSpec `json:"spec"`
}

// CarbonSavings represents the carbon savings compared to different scenarios
type CarbonSavings struct {
	// VsWorstCase is the percentage of carbon saved compared to worst case
	// +optional
	VsWorstCase string `json:"vsWorstCase,omitempty"`

	// VsNaiveCase is the percentage of carbon saved compared to naive case
	// +optional
	VsNaiveCase string `json:"vsNaiveCase,omitempty"`

	// VsMedianCase is the percentage of carbon saved compared to median case
	// +optional
	VsMedianCase string `json:"vsMedianCase,omitempty"`
}

// SchedulingDecision contains details about the carbon-aware scheduling decision
type SchedulingDecision struct {
	// OptimalTime is the calculated optimal time to run the job based on carbon intensity
	// +optional
	OptimalTime *metav1.Time `json:"optimalTime,omitempty"`

	// WorstCaseTime is the time with the highest carbon intensity within the scheduling window
	// +optional
	WorstCaseTime *metav1.Time `json:"worstCaseTime,omitempty"`

	// WorstCaseIntensity is the carbon intensity at the worst case time
	// +optional
	WorstCaseIntensity string `json:"worstCaseIntensity,omitempty"`

	// ImmediateIntensity is the carbon intensity if the job were to run immediately
	// +optional
	ImmediateIntensity string `json:"immediateIntensity,omitempty"`

	// OptimalIntensity is the carbon intensity at the optimal time
	// +optional
	OptimalIntensity string `json:"optimalIntensity,omitempty"`

	// ForecastSource indicates the source of the carbon intensity forecast data
	// +optional
	ForecastSource string `json:"forecastSource,omitempty"`

	// DecisionReason provides the reason for the scheduling decision
	// +optional
	DecisionReason string `json:"decisionReason,omitempty"`
}

// CarbonAwareJobStatus defines the observed state of CarbonAwareJob
type CarbonAwareJobStatus struct {
	// SubmissionTime is when the CarbonAwareJob was submitted
	// +optional
	SubmissionTime *metav1.Time `json:"submissionTime,omitempty"`

	// ScheduledTime is the time when the job is scheduled to run
	// +optional
	ScheduledTime *metav1.Time `json:"scheduledTime,omitempty"`

	// JobName is the name of the Kubernetes Job that was created
	// +optional
	JobName string `json:"jobName,omitempty"`

	// JobStatus is the current status of the underlying Kubernetes Job
	// +optional
	JobStatus *batchv1.JobStatus `json:"jobStatus,omitempty"`

	// SchedulingState represents the current state of the carbon-aware scheduling process
	// +optional
	SchedulingState string `json:"schedulingState,omitempty"`

	// CarbonIntensity is the forecasted carbon intensity at the scheduled time
	// +optional
	CarbonIntensity string `json:"carbonIntensity,omitempty"`

	// CarbonSavings is the estimated carbon savings compared to running at peak intensity
	// +optional
	CarbonSavings *CarbonSavings `json:"carbonSavings,omitempty"`

	// SchedulingDecision contains details about the scheduling decision
	// +optional
	SchedulingDecision *SchedulingDecision `json:"schedulingDecision,omitempty"`

	// Conditions represent the latest available observations of the job's current state
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Scheduled",type="string",JSONPath=".status.scheduledTime",description="Time when the job is scheduled to run"
// +kubebuilder:printcolumn:name="Job",type="string",JSONPath=".status.jobName",description="Name of the created Kubernetes Job"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.schedulingState",description="Current scheduling state"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:shortName=cajob;carbonjob
// +kubebuilder:storageversion

// CarbonAwareJob is the Schema for the carbonawarejobs API
type CarbonAwareJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CarbonAwareJobSpec   `json:"spec,omitempty"`
	Status CarbonAwareJobStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CarbonAwareJobList contains a list of CarbonAwareJob
type CarbonAwareJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CarbonAwareJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CarbonAwareJob{}, &CarbonAwareJobList{})
}
