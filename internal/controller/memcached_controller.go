/*
Copyright 2026.

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
	"reflect"

	cachev1alpha1 "github.com/plabiak/memcached-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// MemcachedReconciler reconciles a Memcached object
type MemcachedReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=cache.memcached.dev.local,resources=memcacheds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cache.memcached.dev.local,resources=memcacheds/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cache.memcached.dev.local,resources=memcacheds/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Memcached object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.23.3/pkg/reconcile
func (r *MemcachedReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the Memcached instance
	memcached := &cachev1alpha1.Memcached{}
	err := r.Get(ctx, req.NamespacedName, memcached)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info(fmt.Sprintf("Memcached resource %s not found. Ignoring since object must be deleted", req.NamespacedName))
			return ctrl.Result{}, nil
		}
		log.Error(err, fmt.Sprintf("Failed to get Memcached resource %s", req.NamespacedName))
		return ctrl.Result{}, err
	}

	svc := r.serviceForMemcached(memcached)
	log.Info("Applying Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
	if err := r.Patch(ctx, svc, client.Apply, client.ForceOwnership, client.FieldOwner("memcached-controller")); err != nil {
		log.Error(err, "Failed to apply Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
		return ctrl.Result{}, err
	}

	sts := r.statefulSetForMemcached(memcached)
	log.Info("Applying StatefulSet", "StatefulSet.Namespace", sts.Namespace, "StatefulSet.Name", sts.Name)
	if err := r.Patch(ctx, sts, client.Apply, client.ForceOwnership, client.FieldOwner("memcached-controller")); err != nil {
		log.Error(err, "Failed to apply StatefulSet", "StatefulSet.Namespace", sts.Namespace, "StatefulSet.Name", sts.Name)
		return ctrl.Result{}, err
	}

	//Pod list reconciliation
	podlist := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(memcached.Namespace),
		client.MatchingLabels(r.labelsForMemcached(memcached.Name)),
	}
	if err := r.List(ctx, podlist, listOpts...); err != nil {
		log.Error(err, fmt.Sprintf("Failed to list pods for Memcached %s", memcached.Name))
		return ctrl.Result{}, err
	}
	podNames := getPodNames(podlist.Items)
	if !reflect.DeepEqual(podNames, memcached.Status.Nodes) {
		memcached.Status.Nodes = podNames
		if err := r.Status().Update(ctx, memcached); err != nil {
			log.Error(err, fmt.Sprintf("Failed to update Memcached status for %s", memcached.Name))
			return ctrl.Result{}, err
		}
	}

	// Update the Memcached status with the pod names
	if !reflect.DeepEqual(podNames, memcached.Status.Nodes) {
		memcached.Status.Nodes = podNames
		if err := r.Status().Update(ctx, memcached); err != nil {
			log.Error(err, fmt.Sprintf("Failed to update Memcached status for %s", memcached.Name))
			return ctrl.Result{}, err
		}
	}
	log.Info(fmt.Sprintf("Reconciling Memcached resource %s", req.NamespacedName))
	return ctrl.Result{}, nil
}

// Function helpers generated services
func (r *MemcachedReconciler) serviceForMemcached(m *cachev1alpha1.Memcached) *corev1.Service {
	ls := r.labelsForMemcached(m.Name)
	svc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(m, cachev1alpha1.GroupVersion.WithKind("Memcached")),
			},
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Selector:  ls,
			Ports: []corev1.ServicePort{{
				Port: m.Spec.ContainerPort,
				Name: "memcached",
			}},
		},
	}
	_ = controllerutil.SetControllerReference(m, svc, r.Scheme)
	return svc
}

// Function helpers generated statefulsets
func (r *MemcachedReconciler) statefulSetForMemcached(m *cachev1alpha1.Memcached) *appsv1.StatefulSet {
	ls := r.labelsForMemcached(m.Name)
	replicas := m.Spec.Size

	image := "memcached:1.6.9-alpine"
	if m.Spec.Image != "" {
		image = m.Spec.Image
	}

	sts := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(m, cachev1alpha1.GroupVersion.WithKind("Memcached")),
			},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &replicas,
			ServiceName: m.Name,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "memcached",
						Image: image,
						Ports: []corev1.ContainerPort{{
							ContainerPort: m.Spec.ContainerPort,
							Name:          "memcached",
						}},
					}},
				},
			},
		},
	}
	return sts
}

// Function helpers generated labels
func (r *MemcachedReconciler) labelsForMemcached(name string) map[string]string {
	return map[string]string{"app": "memcached", "memcached_cr": name}
}

// Function helpers generated pod names
func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}

// SetupWithManager sets up the controller with the Manager.
func (r *MemcachedReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1alpha1.Memcached{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
