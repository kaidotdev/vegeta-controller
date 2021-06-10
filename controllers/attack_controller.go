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

const (
	ownerKey           = ".metadata.controller"
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

	var job batchV1.Job
	if err := r.Client.Get(
		ctx,
		client.ObjectKey{
			Name:      req.Name + "-attack",
			Namespace: req.Namespace,
		},
		&job,
	); errors.IsNotFound(err) {
		job = *r.buildJob(attack)
		if err := controllerutil.SetControllerReference(attack, &job, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.Create(ctx, &job); err != nil {
			return ctrl.Result{}, err
		}
		r.Recorder.Eventf(attack, coreV1.EventTypeNormal, "SuccessfulCreated", "Created job: %q", job.Name)
		logger.V(1).Info("create", "job", job)
	} else if err != nil {
		return ctrl.Result{}, err
	}

	var scenarioConfigMap v1.ConfigMap
	if err := r.Client.Get(
		ctx,
		client.ObjectKey{
			Name:      req.Name + "-scenario",
			Namespace: req.Namespace,
		},
		&scenarioConfigMap,
	); errors.IsNotFound(err) {
		scenarioConfigMap = *r.buildScenarioConfigMap(attack)
		if err := controllerutil.SetControllerReference(attack, &scenarioConfigMap, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.Create(ctx, &scenarioConfigMap); err != nil {
			return ctrl.Result{}, err
		}
		r.Recorder.Eventf(attack, coreV1.EventTypeNormal, "SuccessfulCreated", "Created scenario config map: %q", scenarioConfigMap.Name)
		logger.V(1).Info("create", "scenario config map", scenarioConfigMap)
	} else if err != nil {
		return ctrl.Result{}, err
	}

	var nsswitchConfigMap v1.ConfigMap
	if err := r.Client.Get(
		ctx,
		client.ObjectKey{
			Name:      req.Name + "-nsswitch",
			Namespace: req.Namespace,
		},
		&nsswitchConfigMap,
	); errors.IsNotFound(err) {
		nsswitchConfigMap = *r.buildNSSwitchConfigMap(attack)
		if err := controllerutil.SetControllerReference(attack, &nsswitchConfigMap, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.Create(ctx, &nsswitchConfigMap); err != nil {
			return ctrl.Result{}, err
		}
		r.Recorder.Eventf(attack, coreV1.EventTypeNormal, "SuccessfulCreated", "Created nsswitch config map: %q", nsswitchConfigMap.Name)
		logger.V(1).Info("create", "nsswitch config map", nsswitchConfigMap)
	} else if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *AttackReconciler) buildScenarioConfigMap(attack *vegetaV1.Attack) *v1.ConfigMap {
	return &v1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      attack.Name + "-scenario",
			Namespace: attack.Namespace,
		},
		Data: map[string]string{
			"scenario": attack.Spec.Scenario + "\n", // vegeta needs line break
		},
	}
}

func (r *AttackReconciler) buildNSSwitchConfigMap(attack *vegetaV1.Attack) *v1.ConfigMap {
	return &v1.ConfigMap{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      attack.Name + "-nsswitch",
			Namespace: attack.Namespace,
		},
		Data: map[string]string{
			"nsswitch.conf": "hosts: files dns",
		},
	}
}

func (r *AttackReconciler) buildJob(attack *vegetaV1.Attack) *batchV1.Job {
	fmt.Printf("%+v", attack.Spec)
	appLabel := attack.Name + "-attack"

	labels := map[string]string{
		"app": appLabel,
	}
	for k, v := range attack.Spec.Template.ObjectMeta.Labels {
		labels[k] = v
	}
	attack.Spec.Template.ObjectMeta.Labels = labels

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
	if attack.Spec.Option.Format != "" {
		options = append(options, fmt.Sprintf("-format %s", attack.Spec.Option.Format))
	}
	if !attack.Spec.Option.Keepalive {
		options = append(options, "-keepalive false")
	}

	var vegetaImage string
	if r.VegetaImage == "" {
		vegetaImage = defaultVegetaImage
	} else {
		vegetaImage = r.VegetaImage
	}

	return &batchV1.Job{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      attack.Name + "-attack",
			Namespace: attack.Namespace,
		},
		Spec: batchV1.JobSpec{
			Parallelism: &attack.Spec.Parallelism,
			Template: v1.PodTemplateSpec{
				ObjectMeta: attack.Spec.Template.ObjectMeta,
				Spec: v1.PodSpec{
					Affinity: &v1.Affinity{
						PodAntiAffinity: &v1.PodAntiAffinity{
							PreferredDuringSchedulingIgnoredDuringExecution: []v1.WeightedPodAffinityTerm{
								{
									Weight: 100,
									PodAffinityTerm: v1.PodAffinityTerm{
										LabelSelector: &metaV1.LabelSelector{
											MatchLabels: map[string]string{
												"app": appLabel,
											},
										},
										TopologyKey: "kubernetes.io/hostname",
									},
								},
							},
						},
					},
					HostAliases: attack.Spec.Template.Spec.HostAliases,
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
							Resources:       attack.Spec.AttackContainerSpec.Resources,
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "scenario",
									MountPath: "/var/lib/vegeta",
								},
								{
									Name:      "nsswitch",
									MountPath: "/etc/nsswitch.conf",
									SubPath:   "nsswitch.conf",
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
						{
							Name: "nsswitch",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: attack.Name + "-nsswitch",
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
		client.MatchingFields{ownerKey: attack.Name},
	); err != nil {
		return err
	}

	for _, job := range jobs.Items {
		job := job

		if job.Name == attack.Name+"-attack" {
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
		client.MatchingFields{ownerKey: attack.Name},
	); err != nil {
		return err
	}

	for _, configMap := range configMaps.Items {
		configMap := configMap

		if configMap.Name == attack.Name+"-scenario" || configMap.Name == attack.Name+"-nsswitch" {
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
	if err := mgr.GetFieldIndexer().IndexField(&batchV1.Job{}, ownerKey, func(rawObj runtime.Object) []string {
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

	if err := mgr.GetFieldIndexer().IndexField(&v1.ConfigMap{}, ownerKey, func(rawObj runtime.Object) []string {
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
		Owns(&v1.ConfigMap{}).
		Complete(r)
}
