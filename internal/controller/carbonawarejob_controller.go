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
	"time"

	carbonv1alpha1 "carbon-aware-kube.dev/carbon-aware-kube/api/v1alpha1"
	carbonforecast "carbon-aware-kube.dev/carbon-aware-kube/internal/carbon_forecast"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// CarbonAwareJobReconciler reconciles a CarbonAwareJob object
type CarbonAwareJobReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	forecastProvider carbonforecast.ForecastProvider
}

// +kubebuilder:rbac:groups=batch.carbon-aware-kube.dev,resources=carbonawarejobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch.carbon-aware-kube.dev,resources=carbonawarejobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=batch.carbon-aware-kube.dev,resources=carbonawarejobs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.4/pkg/reconcile
func (r *CarbonAwareJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var caj carbonv1alpha1.CarbonAwareJob
	if err := r.Get(ctx, req.NamespacedName, &caj); err != nil {
		log.Error(err, "Failed to get CarbonAwareJob")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if caj.ObjectMeta.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	// Find the deadline for the job
	creationTime := caj.ObjectMeta.CreationTimestamp.Time
	flexWindow := time.Duration(*caj.Spec.StartFlexWindowSeconds) * time.Second
	deadline := creationTime.Add(flexWindow)

	// If job has not already been scheduled, schedule it
	if caj.Status.ScheduledStartTime == nil {
		expectedDuration := time.Duration(*caj.Spec.ExpectedDurationSeconds) * time.Second

		scheduledTime, err := r.forecastProvider.Evaluate(
			creationTime,
			deadline,
			expectedDuration,
		)
		if err != nil {
			log.Error(err, "Failed to evaluate carbon forecast, requeueing")
			return ctrl.Result{Requeue: true, RequeueAfter: 30 * time.Second}, err
		}

		// Add scheduled condition to the job
		caj.Status.ScheduledStartTime = &metav1.Time{Time: scheduledTime}
		caj.Status.Conditions = append(caj.Status.Conditions, metav1.Condition{
			Type:               carbonv1alpha1.ConditionScheduled,
			Status:             metav1.ConditionTrue,
			Reason:             carbonv1alpha1.ReasonForecastEvaluated,
			Message:            fmt.Sprintf("Scheduled for %s based on carbon forecast", scheduledTime.Format(time.RFC3339)),
			LastTransitionTime: metav1.Now(),
		})

		if err := r.Status().Update(ctx, &caj); err != nil {
			log.Error(err, "Failed to update CarbonAwareJob status")
			return ctrl.Result{}, err
		}

		return ctrl.Result{RequeueAfter: time.Until(scheduledTime)}, nil
	}

	// The jobs is already scheduled -- should we launch it?
	if time.Now().After(caj.Status.ScheduledStartTime.Time) {
		// Check if the job exists
		var existingJobs batchv1.JobList
		if err := r.List(ctx, &existingJobs, client.InNamespace(req.Namespace), client.MatchingFields{"metadata.ownerReferences.uid": string(caj.UID)}); err != nil {
			log.Error(err, "failed to list child jobs")
			return ctrl.Result{}, err
		}

		// If the job exists, update the status
		if len(existingJobs.Items) > 0 {
			job := &existingJobs.Items[0]
			for _, c := range job.Status.Conditions {
				switch c.Type {
				case batchv1.JobComplete:
					caj.Status.Conditions = append(caj.Status.Conditions, metav1.Condition{
						Type:               carbonv1alpha1.ConditionCompleted,
						Status:             metav1.ConditionTrue,
						Reason:             carbonv1alpha1.ReasonJobSucceeded,
						Message:            fmt.Sprintf("Job %s completed successfully", job.Name),
						LastTransitionTime: metav1.Now(),
					})
				case batchv1.JobFailed:
					caj.Status.Conditions = append(caj.Status.Conditions, metav1.Condition{
						Type:               carbonv1alpha1.ConditionFailed,
						Status:             metav1.ConditionTrue,
						Reason:             carbonv1alpha1.ReasonJobFailed,
						Message:            fmt.Sprintf("Job %s failed", job.Name),
						LastTransitionTime: metav1.Now(),
					})
				}
			}

			if err := r.Status().Update(ctx, &caj); err != nil {
				log.Error(err, "failed to update CarbonAwareJob status")
				return ctrl.Result{}, err
			}
		}

		// Otherwise, launch the job
		job := batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: caj.Name,
				Namespace:    caj.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(&caj, carbonv1alpha1.GroupVersion.WithKind("CarbonAwareJob")),
				},
			},
			Spec: caj.Spec.JobTemplate.Spec,
		}

		if err := r.Create(ctx, &job); err != nil {
			log.Error(err, "failed to create job")
			return ctrl.Result{}, err
		}

		caj.Status.Conditions = append(caj.Status.Conditions, metav1.Condition{
			Type:               carbonv1alpha1.ConditionStarted,
			Status:             metav1.ConditionTrue,
			Reason:             carbonv1alpha1.ReasonJobCreated,
			Message:            fmt.Sprintf("Job created: %s", job.Name),
			LastTransitionTime: metav1.Now(),
		})
		caj.Status.JobRef = &carbonv1alpha1.JobReference{
			Name:      job.Name,
			Namespace: job.Namespace,
		}

		if err := r.Status().Update(ctx, &caj); err != nil {
			log.Error(err, "failed to update CarbonAwareJob status")
			return ctrl.Result{}, err
		}

		log.Info("Job created", "name", job.Name)

		// Update CarbonAwareJob status with conditions
		if err := r.Status().Update(ctx, &caj); err != nil {
			log.Error(err, "failed to update CAJob status")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CarbonAwareJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Index Jobs by owner UID (for fast reverse lookup)
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &batchv1.Job{}, ".metadata.ownerReferences.uid", func(rawObj client.Object) []string {
		job := rawObj.(*batchv1.Job)
		ownerRefs := job.GetOwnerReferences()
		for _, ref := range ownerRefs {
			if ref.APIVersion == carbonv1alpha1.GroupVersion.String() && ref.Kind == "CarbonAwareJob" {
				return []string{string(ref.UID)}
			}
		}
		return nil
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&carbonv1alpha1.CarbonAwareJob{}).
		Named("carbonawarejob").
		Owns(&batchv1.Job{}).
		Complete(r)
}
