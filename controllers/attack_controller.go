package controllers

import (
	"context"
	"fmt"
	"strings"

	vegetaV1 "vegeta-controller/api/v1"

	"github.com/go-logr/logr"
	batchV1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var (
	jobOwnerKey        = ".metadata.controller"
	defaultParallelism = 1
	defaultOutput      = "text"
	defaultDuration    = "10s"
	defaultVegetaImage = "peterevans/vegeta:6.7"
)

type AttackReconciler struct {
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	Recorder    record.EventRecorder
	VegetaImage string
}

func (r *AttackReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	attack := &vegetaV1.Attack{}
	ctx := context.Background()
	logger := r.Log.WithValues("attack", req.NamespacedName)
	if err := r.Get(ctx, req.NamespacedName, attack); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if err := r.cleanupOwnedResources(ctx, attack); err != nil {
		return ctrl.Result{}, err
	}

	job := r.buildJob(attack)

	var foundJob batchV1.Job
	if err := r.Client.Get(
		ctx,
		client.ObjectKey{
			Namespace: req.Namespace,
			Name:      req.Name + "-job",
		},
		&foundJob,
	); errors.IsNotFound(err) {
		if err := controllerutil.SetControllerReference(attack, job, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.Create(ctx, job); err != nil {
			return ctrl.Result{}, err
		}
		r.Recorder.Eventf(attack, coreV1.EventTypeNormal, "SuccessfulCreated", "Created job: %q", job.Name)
		logger.V(1).Info("create", "job", job)
	} else if err != nil {
		return ctrl.Result{}, err
	}

	configMap := r.buildConfigMap(attack)

	var foundConfigMap v1.ConfigMap
	if err := r.Client.Get(
		ctx,
		client.ObjectKey{
			Namespace: req.Namespace,
			Name:      req.Name + "-scenario",
		},
		&foundConfigMap,
	); errors.IsNotFound(err) {
		if err := controllerutil.SetControllerReference(attack, configMap, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.Create(ctx, configMap); err != nil {
			return ctrl.Result{}, err
		}
		r.Recorder.Eventf(attack, coreV1.EventTypeNormal, "SuccessfulCreated", "Created config map: %q", configMap.Name)
		logger.V(1).Info("create", "config map", configMap)
	} else if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *AttackReconciler) buildConfigMap(attack *vegetaV1.Attack) *v1.ConfigMap {
	return &v1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      attack.Name + "-scenario",
			Namespace: attack.Namespace,
		},
		Data: map[string]string{
			"scenario": attack.Spec.Scenario,
		},
	}
}

func (r *AttackReconciler) buildJob(attack *vegetaV1.Attack) *batchV1.Job {
	if attack.Spec.Parallelism == 0 {
		attack.Spec.Parallelism = int32(defaultParallelism)
	}
	if attack.Spec.Output == "" {
		attack.Spec.Output = defaultOutput
	}
	if attack.Spec.Option.Duration == "" {
		attack.Spec.Option.Duration = defaultDuration
	}

	labels := map[string]string{
		"app": attack.Name + "-job",
	}
	for k, v := range attack.Spec.Template.Metadata.Labels {
		labels[k] = v
	}

	var options []string
	if attack.Spec.Option.Duration != "" {
		options = append(options, fmt.Sprintf("-duration %s", attack.Spec.Option.Duration))
	}
	if attack.Spec.Option.Rate != 0 {
		options = append(options, fmt.Sprintf("-rate %d", attack.Spec.Option.Rate))
	}
	if attack.Spec.Option.Connections != 0 {
		options = append(options, fmt.Sprintf("-connections %d", attack.Spec.Option.Connections))
	}
	if attack.Spec.Option.Timeout != "" {
		options = append(options, fmt.Sprintf("-timeout %s", attack.Spec.Option.Timeout))
	}
	if attack.Spec.Option.Workers != 0 {
		options = append(options, fmt.Sprintf("-workers %d", attack.Spec.Option.Workers))
	}

	var vegetaImage string
	if r.VegetaImage == "" {
		vegetaImage = defaultVegetaImage
	} else {
		vegetaImage = r.VegetaImage
	}

	return &batchV1.Job{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      attack.Name + "-job",
			Namespace: attack.Namespace,
		},
		Spec: batchV1.JobSpec{
			Parallelism: &attack.Spec.Parallelism,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metaV1.ObjectMeta{
					Annotations: attack.Spec.Template.Metadata.Annotations,
					Labels:      labels,
				},
				Spec: v1.PodSpec{
					Affinity: &v1.Affinity{
						PodAntiAffinity: &v1.PodAntiAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{
								{
									Weight: 100,
									PodAffinityTerm: v1.PodAffinityTerm{
										LabelSelector: &metaV1.LabelSelector{
											MatchLabels: map[string]string{
												"app": attack.Name + "-job",
											},
										},
										TopologyKey: "kubernetes.io/hostname",
									},
								},
							},
						},
					},
					Containers: []v1.Container{
						{
							Name:    "vegeta",
							Image:   vegetaImage,
							Command: []string{"sh"},
							Args: []string{"-c", fmt.Sprintf(
								"vegeta attack %s -targets /var/lib/vegeta/scenario | vegeta report -type %s",
								strings.Join(options, " "),
								attack.Spec.Output,
							)},
							ImagePullPolicy: v1.PullIfNotPresent,
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "scenario",
									MountPath: "/var/lib/vegeta",
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "scenario",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: attack.Name + "-scenario",
									},
								},
							},
						},
					},
					RestartPolicy: v1.RestartPolicyNever,
				},
			},
		},
	}
}

func (r *AttackReconciler) cleanupOwnedResources(ctx context.Context, attack *vegetaV1.Attack) error {
	var jobs batchV1.JobList
	if err := r.List(
		ctx,
		&jobs,
		client.InNamespace(attack.Namespace),
		client.MatchingFields{jobOwnerKey: attack.Name},
	); err != nil {
		return err
	}

	for _, job := range jobs.Items {
		job := job

		if job.Name == attack.Name+"-job" {
			continue
		}

		if err := r.Client.Delete(ctx, &job); err != nil {
			return err
		}
		r.Recorder.Eventf(attack, coreV1.EventTypeNormal, "SuccessfulDeleted", "Deleted job: %q", job.Name)
	}

	var configMaps v1.ConfigMapList
	if err := r.List(
		ctx,
		&configMaps,
		client.InNamespace(attack.Namespace),
		client.MatchingFields{jobOwnerKey: attack.Name},
	); err != nil {
		return err
	}

	for _, configMap := range configMaps.Items {
		configMap := configMap

		if configMap.Name == attack.Name+"-scenario" {
			continue
		}

		if err := r.Client.Delete(ctx, &configMap); err != nil {
			return err
		}
		r.Recorder.Eventf(attack, coreV1.EventTypeNormal, "SuccessfulDeleted", "Deleted config map: %q", configMap.Name)
	}

	return nil
}

func (r *AttackReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(&batchV1.Job{}, jobOwnerKey, func(rawObj runtime.Object) []string {
		job := rawObj.(*batchV1.Job)
		owner := metaV1.GetControllerOf(job)
		if owner == nil {
			return nil
		}
		if owner.Kind != "Attack" {
			return nil
		}

		return []string{owner.Name}
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(&v1.ConfigMap{}, jobOwnerKey, func(rawObj runtime.Object) []string {
		configMap := rawObj.(*v1.ConfigMap)
		owner := metaV1.GetControllerOf(configMap)
		if owner == nil {
			return nil
		}
		if owner.Kind != "Attack" {
			return nil
		}

		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&vegetaV1.Attack{}).
		Owns(&batchV1.Job{}).
		Complete(r)
}
