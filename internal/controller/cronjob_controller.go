/*
Copyright 2025 pixiv Inc.

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

	pixivnetv1 "github.com/pixiv/k8s-job-wrapper/api/v1"
	"github.com/pixiv/k8s-job-wrapper/internal/construct"
	"github.com/pixiv/k8s-job-wrapper/internal/kustomize"
	"github.com/pixiv/k8s-job-wrapper/internal/util"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	applybatchv1 "k8s.io/client-go/applyconfigurations/batch/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// CronJobReconciler reconciles a CronJob object
type CronJobReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Patcher kustomize.Patcher
}

// +kubebuilder:rbac:groups=pixiv.net,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pixiv.net,resources=cronjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=pixiv.net,resources=cronjobs/finalizers,verbs=update
// +kubebuilder:rbac:groups=pixiv.net,resources=podProfiles,verbs=get;list;watch
// +kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CronJob object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.2/pkg/reconcile
func (r *CronJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	cronJob, err := r.getCronJob(ctx, req)
	if apierrors.IsNotFound(err) {
		return ctrl.Result{}, nil
	}
	if err != nil {
		logger.Error(err, "unable to get CronJob")
		return ctrl.Result{}, err
	}
	if !cronJob.DeletionTimestamp.IsZero() {
		// Finalizers are deleting the CronJob.
		// https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers/
		return ctrl.Result{}, nil
	}

	logger = logger.WithValues("podProfile", cronJob.Spec.Profile.PodProfileRef)

	var (
		updateStatus = func() error {
			err := r.Status().Update(ctx, cronJob)
			if err != nil {
				logger.Error(err, "unable to update status")
			}
			return err
		}
		setCondition = func(cond metav1.Condition) {
			meta.SetStatusCondition(&cronJob.Status.Conditions, cond)
		}
		setFailedConditions = func(message string) {
			setCondition(metav1.Condition{
				Type:    string(pixivnetv1.CronJobAvailable),
				Status:  metav1.ConditionFalse,
				Reason:  "Reconciling",
				Message: message, // Set the message to `Available`.
			})
			setCondition(metav1.Condition{
				Type:   string(pixivnetv1.CronJobDegraded),
				Status: metav1.ConditionTrue,
				Reason: "Reconciling",
			})
		}
	)
	// Set the default status to OK.
	setCondition(metav1.Condition{
		Type:   string(pixivnetv1.CronJobAvailable),
		Status: metav1.ConditionTrue,
		Reason: "OK",
	})
	setCondition(metav1.Condition{
		Type:   string(pixivnetv1.CronJobDegraded),
		Status: metav1.ConditionFalse,
		Reason: "OK",
	})

	podProfile, err := r.getPodProfile(ctx, cronJob.Namespace, cronJob.Spec.Profile.PodProfileRef)
	if err != nil {
		logger.Error(err, "unable to get PodProfile")
		setFailedConditions("PodProfile not found")
		_ = updateStatus()
		return ctrl.Result{}, err
	}

	batchCronJob, err := r.constructBatchCronJob(ctx, cronJob, podProfile)
	if err != nil {
		logger.Error(err, "unable to construct cronjob")
		setFailedConditions("Failed to construct batch CronJob manifest")
		_ = updateStatus()
		return ctrl.Result{}, err
	}

	if err := r.applyBatchCronJob(ctx, cronJob, batchCronJob); err != nil {
		logger.Error(err, "unable to apply cronjob")
		setFailedConditions("Failed to apply batch CronJob")
		_ = updateStatus()
		return ctrl.Result{}, err
	}

	logger.Info("reconcile CronJob successfully")
	return ctrl.Result{}, updateStatus()
}

const (
	cronJobFieldManager = "pixiv-job-controller"
)

func (r *CronJobReconciler) applyBatchCronJob(ctx context.Context, cronJob *pixivnetv1.CronJob, batchCronJob *batchv1.CronJob) error {
	logger := log.FromContext(ctx).WithValues("batchCronJob", batchCronJob.Name)

	existingCronJob, err := r.getBatchCronJob(ctx, cronJob)
	if apierrors.IsNotFound(err) {
		if err := r.Create(ctx, batchCronJob); err != nil {
			logger.Error(err, "failed to create batch CronJob")
			return err
		}
		logger.Info("batch CronJob created")
		return nil
	}
	if err != nil {
		logger.Error(err, "failed to get batch CronJob")
		return err
	}

	// https://kubernetes.io/docs/reference/using-api/server-side-apply/#using-server-side-apply-in-a-controller
	patch, err := util.IntoUnstructured(batchCronJob)
	if err != nil {
		logger.Error(err, "failed to convert batch CronJob into unstructured")
		return err
	}
	existingApplyConfig, err := applybatchv1.ExtractCronJob(existingCronJob, cronJobFieldManager)
	if err != nil {
		logger.Error(err, "failed to extract applyconfig from batch CronJob")
		return err
	}

	if equality.Semantic.DeepEqual(patch, existingApplyConfig) {
		logger.Info("batch CronJob manifest is not changed")
		return nil
	}

	if err := r.Patch(ctx, patch, client.Apply, &client.PatchOptions{
		FieldManager: cronJobFieldManager,
		Force:        ptr.To(true),
	}); err != nil {
		logger.Error(err, "failed to apply batch CronJob")
		return err
	}

	logger.Info("batch CronJob applied")
	return nil
}

func (r *CronJobReconciler) constructBatchCronJob(ctx context.Context, cronJob *pixivnetv1.CronJob, podProfile *pixivnetv1.PodProfile) (*batchv1.CronJob, error) {
	return construct.BatchCronJob(ctx, cronJob, podProfile, r.Patcher, r.Scheme)
}

func (r *CronJobReconciler) getBatchCronJob(ctx context.Context, cronJob *pixivnetv1.CronJob) (*batchv1.CronJob, error) {
	var batchCronJob batchv1.CronJob
	if err := r.Get(ctx, client.ObjectKey{
		Namespace: cronJob.Namespace,
		Name:      construct.BatchCronJobName(cronJob),
	}, &batchCronJob); err != nil {
		return nil, err
	}
	return &batchCronJob, nil
}

func (r *CronJobReconciler) getCronJob(ctx context.Context, req ctrl.Request) (*pixivnetv1.CronJob, error) {
	var cronJob pixivnetv1.CronJob
	if err := r.Get(ctx, client.ObjectKey{
		Namespace: req.Namespace,
		Name:      req.Name,
	}, &cronJob); err != nil {
		return nil, err
	}
	return &cronJob, nil
}

func (r *CronJobReconciler) getPodProfile(ctx context.Context, namespace, podProfileRef string) (*pixivnetv1.PodProfile, error) {
	var profile pixivnetv1.PodProfile
	if err := r.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      podProfileRef,
	}, &profile); err != nil {
		return nil, err
	}
	return &profile, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CronJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pixivnetv1.CronJob{}).
		Named("cronjob").
		Owns(&batchv1.CronJob{}).
		Owns(&pixivnetv1.PodProfile{}).
		Complete(r)
}
