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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	batchv1alpha1 "github.com/carbon-aware-kube/operator/api/v1alpha1"
	schedulingclient "github.com/carbon-aware-kube/operator/internal/client"
)

var _ = Describe("CarbonAwareJob Controller", func() {
	// Define constants for the test
	const (
		jobName      = "test-carbonawarejob"
		containerImg = "test:latest"
	)

	// Define the test context
	var (
		namespacedName types.NamespacedName
		testNS         *corev1.Namespace
		reconciler     *CarbonAwareJobReconciler
		schedulingErr  error
	)

	// Mock server variables
	var (
		mockServer *httptest.Server
	)

	// Setup before each test
	BeforeEach(func() {
		schedulingErr = nil

		// Create a test namespace with a unique name to avoid conflicts
		testNS = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "test-ns-",
			},
		}
		err := k8sClient.Create(ctx, testNS)
		Expect(err).NotTo(HaveOccurred())

		// Set up the namespaced name for our CarbonAwareJob
		namespacedName = types.NamespacedName{
			Name:      jobName,
			Namespace: testNS.Name,
		}

		// Initialize mock server with response for scheduling API
		now := time.Now()
		mockResponse := schedulingclient.ScheduleResponse{
			Ideal: schedulingclient.ScheduleOption{
				Time:         now.Add(15 * time.Minute),
				CO2Intensity: 100.0,
			},
			Options: []schedulingclient.ScheduleOption{
				{
					Time:         now,
					CO2Intensity: 150.0,
				},
				{
					Time:         now.Add(15 * time.Minute),
					CO2Intensity: 100.0,
				},
				{
					Time:         now.Add(30 * time.Minute),
					CO2Intensity: 120.0,
				},
			},
			WorstCase: schedulingclient.ScheduleOption{
				Time:         now,
				CO2Intensity: 150.0,
			},
			NaiveCase: schedulingclient.ScheduleOption{
				Time:         now,
				CO2Intensity: 150.0,
			},
			MedianCase: schedulingclient.ScheduleOption{
				Time:         now.Add(30 * time.Minute),
				CO2Intensity: 120.0,
			},
			CarbonSavings: schedulingclient.CarbonSavings{
				VsWorstCase:  33.33,
				VsNaiveCase:  33.33,
				VsMedianCase: 16.67,
			},
		}

		// Create a mock HTTP server for the scheduling API
		mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if schedulingErr != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(schedulingErr.Error()))
				return
			}

			// Only respond to /api/schedule
			if r.URL.Path == "/api/schedule" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(mockResponse)
				return
			}

			w.WriteHeader(http.StatusNotFound)
		}))

		// Create a test server with the handler
		// Set the environment variable for the scheduler URL to point to our test server
		os.Setenv("CARBON_AWARE_SCHEDULER_URL", mockServer.URL)

		// Create a real scheduling client that will connect to our test server
		schedulingClient := schedulingclient.NewSchedulingClient(mockServer.URL)

		// Create a CarbonAwareJobReconciler with the real client connected to our mock server
		reconciler = &CarbonAwareJobReconciler{
			Client:           k8sClient,
			Scheme:           k8sClient.Scheme(),
			SchedulingClient: schedulingClient,
		}

	})

	// Cleanup after each test
	AfterEach(func() {
		// Close the mock server
		if mockServer != nil {
			mockServer.Close()
		}
		// Delete any CarbonAwareJob resources
		carbonAwareJob := &batchv1alpha1.CarbonAwareJob{}
		err := k8sClient.Get(ctx, namespacedName, carbonAwareJob)
		if err == nil {
			// Remove finalizers to allow deletion
			carbonAwareJob.Finalizers = nil
			err = k8sClient.Update(ctx, carbonAwareJob)
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Delete(ctx, carbonAwareJob)
			Expect(err).NotTo(HaveOccurred())
		}

		// Delete the namespace which will delete all resources in it
		err = k8sClient.Delete(ctx, testNS)
		Expect(err).NotTo(HaveOccurred())
	})

	// Test case 1: Creating a new CarbonAwareJob
	Context("When creating a new CarbonAwareJob", func() {
		It("Should initialize the status and add a finalizer", func() {
			// Create a test CarbonAwareJob
			carbonAwareJob := &batchv1alpha1.CarbonAwareJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-carbonawarejob",
					Namespace: testNS.Name,
				},
				Spec: batchv1alpha1.CarbonAwareJobSpec{
					JobTemplate: batchv1alpha1.JobTemplateSpec{
						Metadata: metav1.ObjectMeta{},
						Spec: batchv1.JobSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name:  "test",
											Image: "test:latest",
										},
									},
									RestartPolicy: corev1.RestartPolicyNever,
								},
							},
						},
					},
					MaxDelay: metav1.Duration{Duration: time.Hour},
					Location: "gcp:us-west2",
				},
			}

			// Create the CarbonAwareJob
			err := k8sClient.Create(ctx, carbonAwareJob)
			Expect(err).NotTo(HaveOccurred())

			// Define the namespaced name for lookup
			namespacedName := types.NamespacedName{
				Namespace: testNS.Name,
				Name:      "test-carbonawarejob",
			}

			// Create a new CarbonAwareJob object for updates
			createdJob := &batchv1alpha1.CarbonAwareJob{}

			// Verify the job exists before reconciliation
			err = k8sClient.Get(ctx, namespacedName, createdJob)
			Expect(err).NotTo(HaveOccurred())
			Expect(createdJob.Finalizers).To(BeEmpty())

			// Trigger reconciliation
			var result ctrl.Result
			result, err = reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			// The controller should requeue the request
			Expect(result.Requeue).To(BeTrue())

			By("Checking that the CarbonAwareJob status is initialized")
			// Verify that the status is initialized
			Eventually(func() bool {
				err = k8sClient.Get(ctx, namespacedName, createdJob)
				if err != nil {
					return false
				}
				return createdJob.Status.SubmissionTime != nil &&
					createdJob.Status.SchedulingState == string(SchedulingStateNew)
			}, time.Second*10, time.Millisecond*250).Should(BeTrue())

			By("Triggering second reconciliation to add finalizer")
			// Trigger second reconciliation to add finalizer
			result, err = reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			// Verify finalizer was added
			Eventually(func() bool {
				err = k8sClient.Get(ctx, namespacedName, createdJob)
				if err != nil {
					return false
				}
				return controllerutil.ContainsFinalizer(createdJob, CarbonAwareJobFinalizer)
			}, time.Second*10, time.Millisecond*250).Should(BeTrue())

			By("Checking that the scheduling state is set")
			Eventually(func() string {
				err := k8sClient.Get(ctx, namespacedName, createdJob)
				if err != nil {
					return ""
				}
				return createdJob.Status.SchedulingState
			}, time.Second*10, time.Millisecond*250).ShouldNot(BeEmpty())

			By("Triggering third reconciliation to process the job")
			// Trigger third reconciliation to process the job (handleNewJob)
			result, err = reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())

			// Verify the job is now in pending state with scheduling decision
			Eventually(func() string {
				err := k8sClient.Get(ctx, namespacedName, createdJob)
				if err != nil {
					return ""
				}
				return createdJob.Status.SchedulingState
			}, time.Second*10, time.Millisecond*250).Should(Equal(string(SchedulingStatePending)))

			By("Checking that scheduling decision is set")
			// Verify scheduling decision was made
			Eventually(func() bool {
				err := k8sClient.Get(ctx, namespacedName, createdJob)
				if err != nil {
					return false
				}
				return createdJob.Status.SchedulingDecision != nil
			}, time.Second*10, time.Millisecond*250).Should(BeTrue())
		})
	})

	// Test case 2: When the scheduling API fails
	Context("When the scheduling API fails", func() {
		It("Should fall back to immediate scheduling", func() {
			By("Setting up the mock server to fail")
			schedulingErr = errors.New("API unavailable")

			By("Creating a new CarbonAwareJob")
			carbonAwareJob := &batchv1alpha1.CarbonAwareJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      jobName,
					Namespace: testNS.Name,
				},
				Spec: batchv1alpha1.CarbonAwareJobSpec{
					MaxDelay: metav1.Duration{Duration: 1 * time.Hour},
					Location: "gcp:us-west2",
					JobTemplate: batchv1alpha1.JobTemplateSpec{
						Spec: batchv1.JobSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name:  "test",
											Image: containerImg,
										},
									},
									RestartPolicy: corev1.RestartPolicyNever,
								},
							},
						},
					},
				},
			}
			err := k8sClient.Create(ctx, carbonAwareJob)
			Expect(err).NotTo(HaveOccurred())

			// First reconciliation - initialize status
			result, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			// Second reconciliation - add finalizer
			result, err = reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())

			// Third reconciliation - process job (this will trigger the API call that fails)
			result, err = reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that the job was scheduled with fallback")
			createdJob := &batchv1alpha1.CarbonAwareJob{}
			err = k8sClient.Get(ctx, namespacedName, createdJob)
			Expect(err).NotTo(HaveOccurred())
			Expect(createdJob.Status.SchedulingDecision).NotTo(BeNil())
			Expect(createdJob.Status.SchedulingDecision.ForecastSource).To(Equal("fallback"))

			By("Checking that the carbon intensity is marked as unknown")
			Eventually(func() string {
				err := k8sClient.Get(ctx, namespacedName, createdJob)
				if err != nil {
					return ""
				}
				return createdJob.Status.CarbonIntensity
			}, time.Second*10, time.Millisecond*250).Should(Equal("unknown"))
		})
	})

	// Test case 3: When it's time to create the underlying job
	Context("When it's time to create the underlying job", func() {
		It("Should create the job and update the status", func() {
			By("Creating a CarbonAwareJob with a scheduled time in the past")
			scheduledTime := metav1.NewTime(time.Now().Add(-5 * time.Minute))
			carbonAwareJob := &batchv1alpha1.CarbonAwareJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:       jobName,
					Namespace:  testNS.Name,
					Finalizers: []string{CarbonAwareJobFinalizer},
				},
				Spec: batchv1alpha1.CarbonAwareJobSpec{
					MaxDelay: metav1.Duration{Duration: 1 * time.Hour},
					Location: "gcp:us-west2",
					JobTemplate: batchv1alpha1.JobTemplateSpec{
						Spec: batchv1.JobSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name:  "test",
											Image: containerImg,
										},
									},
									RestartPolicy: corev1.RestartPolicyNever,
								},
							},
						},
					},
				},
			}
			err := k8sClient.Create(ctx, carbonAwareJob)
			Expect(err).NotTo(HaveOccurred())

			// Update the status to be scheduled
			carbonAwareJob.Status = batchv1alpha1.CarbonAwareJobStatus{
				SubmissionTime:  &metav1.Time{Time: time.Now().Add(-30 * time.Minute)},
				ScheduledTime:   &scheduledTime,
				SchedulingState: string(SchedulingStatePending),
				CarbonIntensity: "400.00",
				CarbonSavings: &batchv1alpha1.CarbonSavings{
					VsWorstCase:  "42.85",
					VsNaiveCase:  "33.33",
					VsMedianCase: "27.27",
				},
				SchedulingDecision: &batchv1alpha1.SchedulingDecision{
					OptimalTime:        &scheduledTime,
					OptimalIntensity:   "400.00",
					WorstCaseIntensity: "700.00",
					ImmediateIntensity: "600.00",
					ForecastSource:     "carbon-aware-scheduler-api",
					DecisionReason:     "Optimal time determined based on carbon intensity forecast",
				},
			}
			err = k8sClient.Status().Update(ctx, carbonAwareJob)
			Expect(err).NotTo(HaveOccurred())

			// First reconciliation - process job
			_, err = reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that the job state is updated to Scheduled")
			// Define createdJob here so it can be used in the rest of the test
			createdJob := &batchv1alpha1.CarbonAwareJob{}

			// Use Eventually to wait for the job state to be updated
			Eventually(func() string {
				err := k8sClient.Get(ctx, namespacedName, createdJob)
				if err != nil {
					return ""
				}
				return createdJob.Status.SchedulingState
			}, time.Second*10, time.Millisecond*250).Should(Equal(string(SchedulingStateScheduled)))

			By("Checking that the job name is set")
			Eventually(func() string {
				err := k8sClient.Get(ctx, namespacedName, createdJob)
				if err != nil {
					return ""
				}
				return createdJob.Status.JobName
			}, time.Second*10, time.Millisecond*250).ShouldNot(BeEmpty())

			By("Checking that the underlying job exists")
			Eventually(func() bool {
				if createdJob.Status.JobName == "" {
					return false
				}

				job := &batchv1.Job{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Namespace: testNS.Name,
					Name:      createdJob.Status.JobName,
				}, job)
				return err == nil
			}, time.Second*10, time.Millisecond*250).Should(BeTrue())

			By("Checking that the job has the carbon-aware annotations")
			Eventually(func() bool {
				if createdJob.Status.JobName == "" {
					return false
				}

				job := &batchv1.Job{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Namespace: testNS.Name,
					Name:      createdJob.Status.JobName,
				}, job)

				if err != nil {
					return false
				}

				_, hasIntensity := job.Annotations["carbon-aware-kube.dev/carbon-intensity"]
				_, hasSavings := job.Annotations["carbon-aware-kube.dev/carbon-savings-pct"]

				return hasIntensity && hasSavings
			}, time.Second*10, time.Millisecond*250).Should(BeTrue())
		})
	})

	// Test case 4: When a job is deleted
	Context("When a CarbonAwareJob is deleted", func() {
		It("Should clean up resources and remove finalizer", func() {
			By("Creating and initializing a CarbonAwareJob")
			carbonAwareJob := &batchv1alpha1.CarbonAwareJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:       jobName,
					Namespace:  testNS.Name,
					Finalizers: []string{CarbonAwareJobFinalizer},
				},
				Spec: batchv1alpha1.CarbonAwareJobSpec{
					MaxDelay: metav1.Duration{Duration: 1 * time.Hour},
					Location: "gcp:us-west2",
					JobTemplate: batchv1alpha1.JobTemplateSpec{
						Spec: batchv1.JobSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name:  "test",
											Image: containerImg,
										},
									},
									RestartPolicy: corev1.RestartPolicyNever,
								},
							},
						},
					},
				},
			}
			err := k8sClient.Create(ctx, carbonAwareJob)
			Expect(err).NotTo(HaveOccurred())

			// Set the status to Scheduled
			carbonAwareJob.Status = batchv1alpha1.CarbonAwareJobStatus{
				SubmissionTime:  &metav1.Time{Time: time.Now()},
				SchedulingState: string(SchedulingStateScheduled),
				JobName:         fmt.Sprintf("%s-job", jobName),
			}
			err = k8sClient.Status().Update(ctx, carbonAwareJob)
			Expect(err).NotTo(HaveOccurred())

			// Create the underlying job with an explicit name
			jobNameToUse := fmt.Sprintf("%s-job", jobName)
			job := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      jobNameToUse,
					Namespace: testNS.Name,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: batchv1alpha1.GroupVersion.String(),
							Kind:       "CarbonAwareJob",
							Name:       jobName,
							UID:        carbonAwareJob.UID,
							Controller: &[]bool{true}[0],
						},
					},
				},
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "test",
									Image: containerImg,
								},
							},
							RestartPolicy: corev1.RestartPolicyNever,
						},
					},
				},
			}
			err = k8sClient.Create(ctx, job)
			Expect(err).NotTo(HaveOccurred())

			By("Deleting the CarbonAwareJob")
			err = k8sClient.Delete(ctx, carbonAwareJob)
			Expect(err).NotTo(HaveOccurred())

			// Reconciliation - process job
			_, err = reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that the finalizer is removed")
			deletingJob := &batchv1alpha1.CarbonAwareJob{}
			err = k8sClient.Get(ctx, namespacedName, deletingJob)
			Expect(apierrors.IsNotFound(err)).To(BeTrue(), "CarbonAwareJob should be deleted")

			By("Checking that the underlying job is also deleted")
			deletedJob := &batchv1.Job{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Namespace: testNS.Name,
				Name:      jobNameToUse,
			}, deletedJob)
			Expect(apierrors.IsNotFound(err)).To(BeTrue(), "Underlying Job should be deleted")
		})
	})
})
