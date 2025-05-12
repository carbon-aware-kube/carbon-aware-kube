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

package controller

import (
	"context"
	"fmt"
	"os"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	batchv1alpha1 "github.com/carbon-aware-kube/operator/api/v1alpha1"
	schedulingclient "github.com/carbon-aware-kube/operator/internal/client"
)

// SchedulingState represents the current state of the carbon-aware scheduling process
type SchedulingState string

const (
	// SchedulingStateNew indicates a newly created CarbonAwareJob that hasn't been processed yet
	SchedulingStateNew SchedulingState = "New"

	// SchedulingStatePending indicates a CarbonAwareJob that is waiting for its optimal start time
	SchedulingStatePending SchedulingState = "Pending"

	// SchedulingStateScheduled indicates a CarbonAwareJob that has been scheduled and the Job has been created
	SchedulingStateScheduled SchedulingState = "Scheduled"

	// SchedulingStateRunning indicates a CarbonAwareJob whose underlying Job is running
	SchedulingStateRunning SchedulingState = "Running"

	// SchedulingStateCompleted indicates a CarbonAwareJob whose underlying Job has completed successfully
	SchedulingStateCompleted SchedulingState = "Completed"

	// SchedulingStateFailed indicates a CarbonAwareJob whose underlying Job has failed
	SchedulingStateFailed SchedulingState = "Failed"

	// CarbonAwareJobFinalizer is the finalizer name for CarbonAwareJob resources
	CarbonAwareJobFinalizer = "batch.carbon-aware-kube.dev/finalizer"
)

// CarbonAwareJobReconciler reconciles a CarbonAwareJob object
type CarbonAwareJobReconciler struct {
	ctrlclient.Client
	Scheme *runtime.Scheme
	// SchedulingClient is a client for fetching carbon intensity forecasts
	SchedulingClient schedulingclient.SchedulingClientInterface
}

// +kubebuilder:rbac:groups=batch.carbon-aware-kube.dev,resources=carbonawarejobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch.carbon-aware-kube.dev,resources=carbonawarejobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=batch.carbon-aware-kube.dev,resources=carbonawarejobs/finalizers,verbs=update
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *CarbonAwareJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the CarbonAwareJob instance
	var carbonAwareJob batchv1alpha1.CarbonAwareJob
	if err := r.Get(ctx, req.NamespacedName, &carbonAwareJob); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		logger.Error(err, "Failed to get CarbonAwareJob")
		return ctrl.Result{}, err
	}

	// Initialize status if it's a new CarbonAwareJob
	if carbonAwareJob.Status.SubmissionTime == nil {
		logger.Info("Initializing CarbonAwareJob status", "name", carbonAwareJob.Name)
		now := metav1.Now()
		carbonAwareJob.Status.SubmissionTime = &now
		carbonAwareJob.Status.SchedulingState = string(SchedulingStateNew)
		if err := r.Status().Update(ctx, &carbonAwareJob); err != nil {
			logger.Error(err, "Failed to initialize CarbonAwareJob status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	// Add finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(&carbonAwareJob, CarbonAwareJobFinalizer) {
		logger.Info("Adding finalizer to CarbonAwareJob", "finalizer", CarbonAwareJobFinalizer)
		controllerutil.AddFinalizer(&carbonAwareJob, CarbonAwareJobFinalizer)

		if err := r.Update(ctx, &carbonAwareJob); err != nil {
			logger.Error(err, "Failed to add finalizer to CarbonAwareJob")
			return ctrl.Result{}, err
		}
		logger.Info("Successfully added finalizer to CarbonAwareJob")
		return ctrl.Result{Requeue: true}, nil
	}

	// Check if the resource is being deleted
	if !carbonAwareJob.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &carbonAwareJob)
	}

	// Handle the CarbonAwareJob based on its current state
	switch carbonAwareJob.Status.SchedulingState {
	case string(SchedulingStateNew):
		return r.handleNewJob(ctx, &carbonAwareJob)
	case string(SchedulingStatePending):
		return r.handlePendingJob(ctx, &carbonAwareJob)
	case string(SchedulingStateScheduled), string(SchedulingStateRunning):
		return r.handleScheduledJob(ctx, &carbonAwareJob)
	case string(SchedulingStateCompleted), string(SchedulingStateFailed):
		// Job is in a terminal state, no further action needed
		return ctrl.Result{}, nil
	default:
		logger.Info("CarbonAwareJob in unknown state", "state", carbonAwareJob.Status.SchedulingState)
		return ctrl.Result{}, nil
	}
}

// initializeStatus initializes the status of a new CarbonAwareJob
func (r *CarbonAwareJobReconciler) initializeStatus(ctx context.Context, carbonAwareJob *batchv1alpha1.CarbonAwareJob) error {
	logger := log.FromContext(ctx)
	logger.Info("Initializing CarbonAwareJob status", "name", carbonAwareJob.Name)

	// Set submission time to now
	now := metav1.NewTime(time.Now())
	carbonAwareJob.Status.SubmissionTime = &now
	carbonAwareJob.Status.SchedulingState = string(SchedulingStateNew)

	// Update the status
	return r.Status().Update(ctx, carbonAwareJob)
}

// handleDeletion handles the deletion of a CarbonAwareJob
func (r *CarbonAwareJobReconciler) handleDeletion(ctx context.Context, carbonAwareJob *batchv1alpha1.CarbonAwareJob) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Handling deletion of CarbonAwareJob", "name", carbonAwareJob.Name)

	// Check if the underlying Job exists and delete it if it does
	if carbonAwareJob.Status.JobName != "" {
		job := &batchv1.Job{}
		jobNamespacedName := types.NamespacedName{
			Namespace: carbonAwareJob.Namespace,
			Name:      carbonAwareJob.Status.JobName,
		}

		err := r.Get(ctx, jobNamespacedName, job)
		if err == nil {
			// Job exists, delete it
			propagationPolicy := metav1.DeletePropagationBackground
			if err := r.Delete(ctx, job, &ctrlclient.DeleteOptions{
				PropagationPolicy: &propagationPolicy,
			}); err != nil && !errors.IsNotFound(err) {
				logger.Error(err, "Failed to delete Job", "job", carbonAwareJob.Status.JobName)
				return ctrl.Result{}, err
			}
			logger.Info("Deleted Job", "job", carbonAwareJob.Status.JobName)
		} else if !errors.IsNotFound(err) {
			logger.Error(err, "Failed to get Job", "job", carbonAwareJob.Status.JobName)
			return ctrl.Result{}, err
		}
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(carbonAwareJob, CarbonAwareJobFinalizer)
	if err := r.Update(ctx, carbonAwareJob); err != nil {
		logger.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// handleNewJob processes a newly created CarbonAwareJob
func (r *CarbonAwareJobReconciler) handleNewJob(ctx context.Context, carbonAwareJob *batchv1alpha1.CarbonAwareJob) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Handling new CarbonAwareJob", "name", carbonAwareJob.Name)

	// Get submission time and max delay
	submissionTime := carbonAwareJob.Status.SubmissionTime.Time
	maxDelay := carbonAwareJob.Spec.MaxDelay.Duration

	// Default job duration to 1 hour if not specified
	jobDuration := 1 * time.Hour
	if carbonAwareJob.Spec.MaxDuration != nil && carbonAwareJob.Spec.MaxDuration.Duration > 0 {
		jobDuration = carbonAwareJob.Spec.MaxDuration.Duration
	}

	// Default location to "aws:us-east-1" if not specified
	location := "aws:us-east-1"
	if carbonAwareJob.Spec.Location != "" {
		location = carbonAwareJob.Spec.Location
	}

	// Get the optimal schedule from the scheduling API
	scheduleResp, err := r.SchedulingClient.GetOptimalSchedule(
		ctx,
		submissionTime,
		maxDelay,
		jobDuration,
		location,
	)

	if err != nil {
		logger.Error(err, "Failed to get optimal schedule from API")

		// Fallback to immediate scheduling if API fails
		optimalTime := metav1.NewTime(submissionTime)
		carbonAwareJob.Status.SchedulingDecision = &batchv1alpha1.SchedulingDecision{
			OptimalTime:        &optimalTime,
			OptimalIntensity:   "unknown",
			WorstCaseTime:      &metav1.Time{Time: submissionTime},
			WorstCaseIntensity: "unknown",
			ImmediateIntensity: "unknown",
			ForecastSource:     "fallback",
			DecisionReason:     fmt.Sprintf("Failed to get forecast: %v. Scheduling immediately.", err),
		}

		// Set the scheduled time to now
		carbonAwareJob.Status.ScheduledTime = &optimalTime
		carbonAwareJob.Status.CarbonIntensity = "unknown"
		carbonAwareJob.Status.CarbonSavings = &batchv1alpha1.CarbonSavings{
			VsWorstCase:  "0.00%",
			VsNaiveCase:  "0.00%",
			VsMedianCase: "0.00%",
		}
	} else {
		// Use the optimal schedule from the API response
		optimalTime := metav1.NewTime(scheduleResp.Ideal.Time)
		worstCaseTime := metav1.NewTime(scheduleResp.WorstCase.Time)

		// Format zone information
		optimalZone := fmt.Sprintf("%s:%s", scheduleResp.Ideal.Zone.Provider, scheduleResp.Ideal.Zone.Region)
		
		// Update the scheduling decision
		carbonAwareJob.Status.SchedulingDecision = &batchv1alpha1.SchedulingDecision{
			OptimalTime:        &optimalTime,
			OptimalIntensity:   fmt.Sprintf("%.2f gCO2eq/kWh", scheduleResp.Ideal.CO2Intensity),
			WorstCaseTime:      &worstCaseTime,
			WorstCaseIntensity: fmt.Sprintf("%.2f gCO2eq/kWh", scheduleResp.WorstCase.CO2Intensity),
			ImmediateIntensity: fmt.Sprintf("%.2f gCO2eq/kWh", scheduleResp.NaiveCase.CO2Intensity),
			ForecastSource:     "carbon-aware-scheduler-api",
			DecisionReason:     fmt.Sprintf("Optimal time determined for %s based on carbon intensity forecast", optimalZone),
		}

		// Set the scheduled time
		carbonAwareJob.Status.ScheduledTime = &optimalTime

		// Set carbon savings
		carbonAwareJob.Status.CarbonSavings = &batchv1alpha1.CarbonSavings{
			VsWorstCase:  fmt.Sprintf("-%.2f%%", scheduleResp.CarbonSavings.VsWorstCase),
			VsNaiveCase:  fmt.Sprintf("-%.2f%%", scheduleResp.CarbonSavings.VsNaiveCase),
			VsMedianCase: fmt.Sprintf("-%.2f%%", scheduleResp.CarbonSavings.VsMedianCase),
		}

		// Set carbon intensity
		carbonAwareJob.Status.CarbonIntensity = carbonAwareJob.Status.SchedulingDecision.OptimalIntensity
	}

	// Update state to pending
	carbonAwareJob.Status.SchedulingState = string(SchedulingStatePending)

	// Update the status
	if err := r.Status().Update(ctx, carbonAwareJob); err != nil {
		logger.Error(err, "Failed to update CarbonAwareJob status")
		return ctrl.Result{}, err
	}

	// Requeue to check if it's time to create the job
	return ctrl.Result{RequeueAfter: time.Until(carbonAwareJob.Status.ScheduledTime.Time)}, nil
}

// handlePendingJob checks if it's time to create the underlying Job
func (r *CarbonAwareJobReconciler) handlePendingJob(ctx context.Context, carbonAwareJob *batchv1alpha1.CarbonAwareJob) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Handling pending CarbonAwareJob", "name", carbonAwareJob.Name)

	// Check if it's time to create the job
	now := time.Now()
	scheduledTime := carbonAwareJob.Status.ScheduledTime.Time

	if now.Before(scheduledTime) {
		// Not time yet, requeue
		return ctrl.Result{RequeueAfter: time.Until(scheduledTime)}, nil
	}

	// Time to create the job
	job, err := r.constructJobFromTemplate(carbonAwareJob)
	if err != nil {
		logger.Error(err, "Failed to construct Job from template")
		return ctrl.Result{}, err
	}

	// Set the owner reference
	if err := controllerutil.SetControllerReference(carbonAwareJob, job, r.Scheme); err != nil {
		logger.Error(err, "Failed to set controller reference")
		return ctrl.Result{}, err
	}

	// Create the job
	if err := r.Create(ctx, job); err != nil {
		logger.Error(err, "Failed to create Job")
		return ctrl.Result{}, err
	}

	// Update status
	carbonAwareJob.Status.JobName = job.Name
	carbonAwareJob.Status.SchedulingState = string(SchedulingStateScheduled)

	// Update the status
	if err := r.Status().Update(ctx, carbonAwareJob); err != nil {
		logger.Error(err, "Failed to update CarbonAwareJob status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil // Check job status periodically
}

// handleScheduledJob checks the status of the underlying Job
func (r *CarbonAwareJobReconciler) handleScheduledJob(ctx context.Context, carbonAwareJob *batchv1alpha1.CarbonAwareJob) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Handling scheduled CarbonAwareJob", "name", carbonAwareJob.Name, "job", carbonAwareJob.Status.JobName)

	// Get the job
	job := &batchv1.Job{}
	jobNamespacedName := types.NamespacedName{
		Namespace: carbonAwareJob.Namespace,
		Name:      carbonAwareJob.Status.JobName,
	}

	if err := r.Get(ctx, jobNamespacedName, job); err != nil {
		if errors.IsNotFound(err) {
			// Job was deleted, update status
			carbonAwareJob.Status.SchedulingState = string(SchedulingStateFailed)
			carbonAwareJob.Status.JobStatus = nil
			if err := r.Status().Update(ctx, carbonAwareJob); err != nil {
				logger.Error(err, "Failed to update CarbonAwareJob status")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get Job")
		return ctrl.Result{}, err
	}

	// Update job status
	carbonAwareJob.Status.JobStatus = &job.Status

	// Check job status
	if job.Status.Active > 0 {
		carbonAwareJob.Status.SchedulingState = string(SchedulingStateRunning)
	} else if job.Status.Succeeded > 0 {
		carbonAwareJob.Status.SchedulingState = string(SchedulingStateCompleted)
	} else if job.Status.Failed > 0 {
		carbonAwareJob.Status.SchedulingState = string(SchedulingStateFailed)
	}

	// Update the status
	if err := r.Status().Update(ctx, carbonAwareJob); err != nil {
		logger.Error(err, "Failed to update CarbonAwareJob status")
		return ctrl.Result{}, err
	}

	// If job is still running, requeue to check again later
	if carbonAwareJob.Status.SchedulingState == string(SchedulingStateRunning) {
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Job is in a terminal state, no need to requeue
	return ctrl.Result{}, nil
}

// constructJobFromTemplate creates a Job from the CarbonAwareJob template
func (r *CarbonAwareJobReconciler) constructJobFromTemplate(carbonAwareJob *batchv1alpha1.CarbonAwareJob) (*batchv1.Job, error) {
	// Create a unique name for the job
	jobName := fmt.Sprintf("%s-%d", carbonAwareJob.Name, time.Now().Unix())

	// Create the job
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: carbonAwareJob.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "carbon-aware-job",
				"app.kubernetes.io/instance":   carbonAwareJob.Name,
				"app.kubernetes.io/managed-by": "carbon-aware-operator",
			},
			Annotations: map[string]string{
				"carbon-aware-kube.dev/carbon-intensity":     carbonAwareJob.Status.CarbonIntensity,
				"carbon-aware-kube.dev/scheduled-time":       carbonAwareJob.Status.ScheduledTime.Format(time.RFC3339),
				"carbon-aware-kube.dev/carbon-savings-pct":   carbonAwareJob.Status.CarbonSavings.VsNaiveCase,
				"carbon-aware-kube.dev/parent-resource-name": carbonAwareJob.Name,
				"carbon-aware-kube.dev/parent-resource-uid":  string(carbonAwareJob.UID),
			},
		},
		Spec: carbonAwareJob.Spec.Template.Spec,
	}

	// Copy any labels and annotations from the template metadata
	if carbonAwareJob.Spec.Template.Metadata.Labels != nil {
		for k, v := range carbonAwareJob.Spec.Template.Metadata.Labels {
			job.Labels[k] = v
		}
	}

	if carbonAwareJob.Spec.Template.Metadata.Annotations != nil {
		for k, v := range carbonAwareJob.Spec.Template.Metadata.Annotations {
			job.Annotations[k] = v
		}
	}

	return job, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CarbonAwareJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Initialize the scheduling client
	schedulerURL := os.Getenv("CARBON_AWARE_SCHEDULER_URL")
	if schedulerURL == "" {
		schedulerURL = "http://carbon-aware-scheduler:8080" // Default URL if not specified
	}
	r.SchedulingClient = schedulingclient.NewSchedulingClient(schedulerURL)

	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1alpha1.CarbonAwareJob{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}
