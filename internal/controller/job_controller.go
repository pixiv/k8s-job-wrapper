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
	"encoding/json"
	"errors"
	"slices"
	"time"

	pixivnetv1 "github.com/pixiv/k8s-job-wrapper/api/v1"
	pixivnetv2 "github.com/pixiv/k8s-job-wrapper/api/v2"
	"github.com/pixiv/k8s-job-wrapper/internal/construct"
	"github.com/pixiv/k8s-job-wrapper/internal/kustomize"
	"github.com/pixiv/k8s-job-wrapper/internal/util"
	batchv1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// JobReconciler reconciles a Job object
type JobReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Patcher kustomize.Patcher
}

// +kubebuilder:rbac:groups=pixiv.net,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=pixiv.net,resources=jobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=pixiv.net,resources=jobs/finalizers,verbs=update
// +kubebuilder:rbac:groups=pixiv.net,resources=podProfiles,verbs=get;list;watch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs/status,verbs=get;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Job object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.2/pkg/reconcile
func (r *JobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	job, err := r.getJob(ctx, req)
	if apierrors.IsNotFound(err) {
		return r.onDeleted(ctx, req)
	}
	if err != nil {
		logger.Error(err, "unable to get Job")
		return ctrl.Result{}, err
	}
	if !job.DeletionTimestamp.IsZero() {
		// Finalizers are deleting the Job.
		// https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers/
		return ctrl.Result{}, nil
	}

	logger = logger.WithValues("podProfile", job.Spec.PodProfile.Ref, "jobProfile", job.Spec.JobProfile.Ref)

	var (
		updateStatus = func() error {
			err := r.Status().Update(ctx, job)
			if err != nil {
				logger.Error(err, "unable to update status")
			}
			return err
		}
		setCondition = func(cond metav1.Condition) {
			meta.SetStatusCondition(&job.Status.Conditions, cond)
		}
		setFailedConditions = func(message string) {
			setCondition(metav1.Condition{
				Type:    string(pixivnetv1.JobAvailable),
				Status:  metav1.ConditionFalse,
				Reason:  "Reconciling",
				Message: message, // Set the message to `Available`.
			})
			setCondition(metav1.Condition{
				Type:   string(pixivnetv1.JobDegraded),
				Status: metav1.ConditionTrue,
				Reason: "Reconciling",
			})
		}
	)
	// Set the default status to OK.
	setCondition(metav1.Condition{
		Type:   string(pixivnetv1.JobAvailable),
		Status: metav1.ConditionTrue,
		Reason: "OK",
	})
	setCondition(metav1.Condition{
		Type:   string(pixivnetv1.JobDegraded),
		Status: metav1.ConditionFalse,
		Reason: "OK",
	})

	podProfile, err := r.getPodProfile(ctx, job.Namespace, job.Spec.PodProfile.Ref)
	if err != nil {
		logger.Error(err, "unable to get PodProfile")
		setFailedConditions("PodProfile not found")
		_ = updateStatus()
		return ctrl.Result{}, err
	}

	jobProfile, err := r.getJobProfile(ctx, job.Namespace, job.Spec.JobProfile.Ref)
	if err != nil {
		logger.Error(err, "unable to get JobProfile")
		setFailedConditions("JobProfile not found")
		_ = updateStatus()
		return ctrl.Result{}, err
	}

	nextJob, err := r.constructBatchJob(ctx, job, podProfile, jobProfile)
	if err != nil {
		logger.Error(err, "unable to construct job")
		setFailedConditions("Failed to construct batch Job manifest")
		_ = updateStatus()
		return ctrl.Result{}, err
	}

	result, err := r.processBatchJobsTTL(ctx, time.Now(), job)
	if err != nil {
		logger.Error(err, "unable to delete expired jobs")
	}

	if *&jobProfile.Spec.Template.JobsHistoryLimit != nil {
		if err := r.deleteOldBatchJobs(ctx, job, *jobProfile.Spec.Template.JobsHistoryLimit); err != nil {
			logger.Error(err, "unable to delete old jobs")
		}
	}

	existJobs, err := r.listBatchJobsOrderByCreationTimestampDesc(ctx, job)
	if err != nil {
		logger.Error(err, "unable to list existing batch jobs")
		setFailedConditions("Failed to list existing batch Jobs")
		_ = updateStatus()
		return result, err
	}

	var latestJob *batchv1.Job
	if idx := slices.IndexFunc(existJobs, func(x batchv1.Job) bool {
		return x.DeletionTimestamp == nil
	}); idx >= 0 {
		latestJob = &existJobs[idx]
	}

	creationStatus := r.shouldCreateBatchJob(latestJob, nextJob)
	if !creationStatus.shouldCreate {
		logger.Info("skip to create batch job", "reason", creationStatus.reason)
		return result, updateStatus()
	}

	logger.Info("creating batch job", "reason", creationStatus.reason)

	if err := r.createBatchJob(ctx, job, nextJob); err != nil {
		logger.Error(err, "unable to create job")
		setFailedConditions("Failed to create batch Job")
		_ = updateStatus()
		return result, err
	}

	logger.Info("reconcile Job successfully")
	return result, updateStatus()
}

func (r *JobReconciler) onDeleted(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Job deleted")
	return ctrl.Result{}, nil
}

func (r *JobReconciler) createBatchJob(ctx context.Context, job *pixivnetv2.Job, nextJob *batchv1.Job) error {
	logger := log.FromContext(ctx).WithValues("batchJob", nextJob.Name)

	err := r.Create(ctx, nextJob)
	switch {
	case err == nil:
		j, _ := json.Marshal(nextJob.Spec)
		logger.Info("batch job created")
		logger.V(2).Info("batch job created", "spec", string(j))
		return nil
	case apierrors.IsAlreadyExists(err): // The name (hash) of a past batch job conflicted.
		if job.Status.CollisionCount == nil {
			job.Status.CollisionCount = new(int32)
		}
		preCollisionCount := *job.Status.CollisionCount
		*job.Status.CollisionCount++
		if sErr := r.Status().Update(ctx, job); sErr == nil {
			logger.Info("found a hash collision for batch job - bumping collisionCount to resolve it",
				"oldCollisionCount", preCollisionCount,
				"newCollisionCount", *job.Status.CollisionCount,
			)
		}
		return err
	default:
		j, _ := json.Marshal(nextJob.Spec)
		logger.Error(err, "failed to create batch job", "spec", string(j))
		return err
	}
}

// Prune old batch jobs that exceed the history limit.
func (r *JobReconciler) deleteOldBatchJobs(ctx context.Context, job *pixivnetv2.Job, jobsHistoryLimit int) error {
	history, err := r.listBatchJobsOrderByCreationTimestampDesc(ctx, job)
	if err != nil {
		return err
	}

	logger := log.FromContext(ctx)

	var (
		kept int
		errs []error
	)
	for _, x := range history {
		// Skip items that have not yet finished or are already pending deletion.
		if !r.isBatchJobTerminated(&x) || x.DeletionTimestamp != nil {
			continue
		}
		// Otherwise, skip it if it falls within the history limit.
		if kept < jobsHistoryLimit {
			kept++
			continue
		}

		uid := x.GetUID()
		logger.Info("deleting batch job because too old to keep", "batchJob", x.Name, "uid", uid)
		if err := r.Delete(ctx, &x, &client.DeleteOptions{
			PropagationPolicy: ptr.To(metav1.DeletePropagationBackground),
			Preconditions: &metav1.Preconditions{
				UID: &uid,
			},
		}); err != nil {
			logger.Error(err, "failed to delete batch job from history", "batchJob", x.Name, "uid", uid)
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// Delete batch Jobs where TTLSecondsAfterFinished has passed since their completion time.
// If there are any batch Jobs with a TTL set that have not yet expired, schedule a future [JobReconciler.Reconcile].
func (r *JobReconciler) processBatchJobsTTL(ctx context.Context, now time.Time, job *pixivnetv2.Job) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var defaultResult ctrl.Result
	history, err := r.listBatchJobsOrderByCreationTimestampDesc(ctx, job)
	if err != nil {
		return defaultResult, err
	}

	results := make([]processBatchJobTTL, len(history))
	for i, x := range history {
		results[i] = r.processBatchJobTTL(ctx, now, &x)
	}

	var (
		errs        = []error{}
		delays      = []time.Duration{}
		deleted     int
		allowDelete = len(results) - 1 // If all of the batch jobs are deleted, it will trigger JobReconciler.Reconcile to regenerate them.
	)
	// Check from oldest to newest.
	slices.Reverse(results)
	for _, x := range results {
		if x.shouldDelete && deleted < allowDelete {
			deleted++
			uid := x.job.GetUID()
			logger.Info("deleting batch job due to TTL", "uid", uid, "batchJob", x.job.GetName())
			if err := r.Delete(ctx, x.job, &client.DeleteOptions{
				PropagationPolicy: ptr.To(metav1.DeletePropagationBackground),
				Preconditions: &metav1.Preconditions{
					UID: &uid,
				},
			}); err != nil {
				logger.Error(err, "batch job TTL expired but failed to delete", "uid", uid, "batchJob", x.job.GetName())
				errs = append(errs, err)
			}
		}

		if delay := x.delay; delay != nil {
			delays = append(delays, *delay)
		}

		if err := x.err; err != nil {
			errs = append(errs, err)
		}
	}

	err = errors.Join(errs...)

	if len(delays) > 0 {
		if minDelay := slices.Min(delays); minDelay > 0 {
			logger.Info("trying deletion of expired batch Jobs later", "delay", minDelay)
			return ctrl.Result{
				RequeueAfter: minDelay,
			}, err
		}
	}
	return defaultResult, err
}

const (
	// An annotation that indicates the batch job was deleted due to its TTL.
	// We use this to distinguish this case, as we don't want to call [JobReconciler.Reconcile] on a TTL-based deletion.
	BatchJobAnnotationTTLExpired = "jobs.pixiv.net/ttl-expired"
)

func hasTTLExpiredAnnotation(obj client.Object) bool {
	if v, ok := obj.GetAnnotations()[BatchJobAnnotationTTLExpired]; ok && v == "true" {
		return true
	}
	return false
}

func (r *JobReconciler) addTTLExpiredAnnotationToBatchJob(ctx context.Context, job *batchv1.Job) error {
	annotations := job.GetAnnotations()
	annotations[BatchJobAnnotationTTLExpired] = "true"
	job.Annotations = annotations
	return r.Update(ctx, job)
}

type processBatchJobTTL struct {
	job          *batchv1.Job
	shouldDelete bool           // true indicates ttl expired
	delay        *time.Duration // not nil indicates that reconcile should be executed
	err          error
}

// If the batch Job has finished and its TTL has expired, delete it.
// If the TTL has not yet expired, return the remaining time.
func (r *JobReconciler) processBatchJobTTL(ctx context.Context, now time.Time, job *batchv1.Job) processBatchJobTTL {
	logger := log.FromContext(ctx).WithValues("batchJob", job.Name)

	if hasTTLExpiredAnnotation(job) {
		return processBatchJobTTL{
			job:          job,
			shouldDelete: true,
		}
	}

	expireAt, ok := r.batchJobExpireAt(ctx, job)
	if !ok {
		return processBatchJobTTL{
			job: job,
		}
	}
	logger = logger.WithValues("expireAt", expireAt)

	if delay := expireAt.Sub(now); delay > 0 {
		// Delete this in a later reconcile, not now.
		logger.Info("found finished batch Job with remaining TTL", "remaining", delay)
		delay += 3 * time.Second // Reconciling at the last minute might result in delay > 0 again, so we'll add a 3-second delay.
		return processBatchJobTTL{
			job:   job,
			delay: &delay,
		}
	}

	logger.Info("add TTL expired annotation to batch job")
	if err := r.addTTLExpiredAnnotationToBatchJob(ctx, job); err != nil {
		logger.Error(err, "failed to add TTL expired annotation to batch job")
		return processBatchJobTTL{
			job: job,
			err: err,
		}
	}
	return processBatchJobTTL{
		job: job,
	}
}

// Calculate when a batch job with a TTL will expire and be deleted.
func (r *JobReconciler) batchJobExpireAt(_ context.Context, job *batchv1.Job) (*time.Time, bool) {
	info := util.GetFinishedBatchJobInfo(job)
	// Skip items that have not finished, have an unknown completion time, or have already been deleted.
	if !info.Finished || info.FinishedAt.IsZero() || job.DeletionTimestamp != nil {
		return nil, false
	}
	ttl, ok := construct.BatchJobTTLSecondsAfterFinishedFromAnnotations(job)
	if !ok {
		return nil, false
	}
	expireAt := info.FinishedAt.Add(time.Duration(ttl) * time.Second)
	return &expireAt, true
}

type jobCreationStatus struct {
	shouldCreate bool
	reason       string
}

func (r *JobReconciler) shouldCreateBatchJob(latestJob, nextJob *batchv1.Job) jobCreationStatus {
	switch {
	case latestJob == nil:
		// Create a new job if not exists.
		return jobCreationStatus{
			shouldCreate: true,
			reason:       "no existing job",
		}
	case !r.isBatchJobTerminated(latestJob):
		// Do not create a new job while an existing one is still active.
		return jobCreationStatus{
			shouldCreate: false,
			reason:       "existing job is running",
		}
	case r.getBatchJobSpecHash(latestJob) == r.getBatchJobSpecHash(nextJob):
		// Do not create a new job without changes.
		return jobCreationStatus{
			shouldCreate: false,
			reason:       "job spec is not changed",
		}
	default:
		// Create a new job with changes.
		return jobCreationStatus{
			shouldCreate: true,
			reason:       "job spec is changed",
		}
	}
}

func (*JobReconciler) isBatchJobTerminated(job *batchv1.Job) bool {
	return util.IsBatchJobFinished(job)
}

// Get the hash of batchJob.spec.
func (*JobReconciler) getBatchJobSpecHash(job *batchv1.Job) string {
	return job.GetLabels()[construct.BatchJobLabelSpecHashKey]
}

// A label used to retrieve a list of the generated batch Jobs.
func (*JobReconciler) batchJobLabels(job *pixivnetv2.Job) map[string]string {
	return construct.BatchJobLabelsForList(job)
}

// Create a batchJob object.
func (r *JobReconciler) constructBatchJob(ctx context.Context, job *pixivnetv2.Job, podProfile *pixivnetv1.PodProfile, jobProfile *pixivnetv2.JobProfile) (*batchv1.Job, error) {
	logger := log.FromContext(ctx).WithValues("podProfile", podProfile.Name, "jobProfile", jobProfile.Name)
	nextJob, err := construct.BatchJob(ctx, job, podProfile, jobProfile, r.Patcher, r.Scheme)
	if err != nil {
		logger.Error(err, "failed to construct batch job manifest")
		return nil, err
	}
	logger.Info("new batch job manifest constructed successfully")
	return nextJob, nil
}

func (r *JobReconciler) listBatchJobsOrderByCreationTimestampDesc(ctx context.Context, job *pixivnetv2.Job) ([]batchv1.Job, error) {
	var batchJobs batchv1.JobList
	if err := r.List(ctx, &batchJobs, &client.ListOptions{
		Namespace:     job.Namespace,
		LabelSelector: labels.SelectorFromSet(r.batchJobLabels(job)),
	}); err != nil {
		return nil, err
	}

	// Sort by creation time desc.
	items := batchJobs.Items
	slices.SortFunc(items, func(x, y batchv1.Job) int {
		if x.CreationTimestamp.Before(&y.CreationTimestamp) {
			return 1
		}
		if x.CreationTimestamp == y.CreationTimestamp {
			return 0
		}
		return -1
	})
	return items, nil
}

func (r *JobReconciler) getJob(ctx context.Context, req ctrl.Request) (*pixivnetv2.Job, error) {
	var job pixivnetv2.Job
	if err := r.Get(ctx, client.ObjectKey{
		Namespace: req.Namespace,
		Name:      req.Name,
	}, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *JobReconciler) getPodProfile(ctx context.Context, namespace, podProfileRef string) (*pixivnetv1.PodProfile, error) {
	var profile pixivnetv1.PodProfile
	if err := r.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      podProfileRef,
	}, &profile); err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *JobReconciler) getJobProfile(ctx context.Context, namespace, jobProfileRef string) (*pixivnetv2.JobProfile, error) {
	var profile pixivnetv2.JobProfile
	if err := r.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      jobProfileRef,
	}, &profile); err != nil {
		return nil, err
	}
	return &profile, nil
}

/*
Run [JobReconciler.Reconcile] when a batch Job meets the following conditions.

  - holds jobs.pixiv.net/v1 as owner
*/
func (r *JobReconciler) watchBatchJobHandler() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
		name, ok := r.getOwnerName(obj.GetOwnerReferences())
		if !ok {
			return []reconcile.Request{}
		}

		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Namespace: obj.GetNamespace(),
					Name:      name,
				},
			},
		}
	})
}

// If an ownerReference with `jobs.pixiv.net/v1` exists, return its name.
func (*JobReconciler) getOwnerName(ownerReferences []metav1.OwnerReference) (string, bool) {
	for _, x := range ownerReferences {
		if v := x.Controller; v == nil || !*v {
			continue
		}
		if x.APIVersion == pixivnetv1.GroupVersion.String() && x.Kind == "Job" {
			return x.Name, true
		}
	}
	return "", false
}

// Decides whether to pass a "batch Job changed" event to the JobReconciler.watchBatchJobHandler.
func (r *JobReconciler) watchBatchJobPredicates() builder.Predicates {
	return builder.WithPredicates(predicate.Funcs{
		UpdateFunc: func(_ event.UpdateEvent) bool {
			return true
		},
		CreateFunc: func(_ event.CreateEvent) bool {
			return true
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Ignore if it was deleted due to TTL expiration.
			return !hasTTLExpiredAnnotation(e.Object)
		},
		GenericFunc: func(_ event.GenericEvent) bool {
			return true
		},
	})
}

func (r *JobReconciler) indexByPodProfileRef() client.IndexerFunc {
	return func(obj client.Object) []string {
		job := obj.(*pixivnetv1.Job)
		return []string{job.Spec.Profile.PodProfileRef}
	}
}

const jobPodProfileRefKey = ".spec.profile.podProfileRef"

func (r *JobReconciler) watchPodProfileHandler() handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
		logger := log.FromContext(ctx).WithValues("podProfileRef", obj.GetName())

		logger.Info("find jobs by podProfileRef")
		var jobs pixivnetv1.JobList
		if err := r.List(ctx, &jobs, client.InNamespace(obj.GetNamespace()), client.MatchingFields{
			jobPodProfileRefKey: obj.GetName(),
		}); err != nil {
			logger.Error(err, "failed to list jobs by podProfileRef")
			return []reconcile.Request{}
		}

		requests := make([]reconcile.Request, len(jobs.Items))
		for i, x := range jobs.Items {
			logger.V(2).Info("found job by podProfileRef", "job", x.GetName())
			requests[i] = reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: x.GetNamespace(),
					Name:      x.GetName(),
				},
			}
		}
		return requests
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *JobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&pixivnetv1.Job{},
		jobPodProfileRefKey,
		r.indexByPodProfileRef()); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&pixivnetv1.Job{}).
		Named("job").
		Watches(
			&pixivnetv1.PodProfile{},
			r.watchPodProfileHandler(),
		).
		Watches(
			&batchv1.Job{},
			r.watchBatchJobHandler(),
			r.watchBatchJobPredicates(),
		).
		Complete(r)
}
