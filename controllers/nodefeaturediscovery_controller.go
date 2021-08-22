/*
Copyright 2020-2021 The Kubernetes Authors.

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

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	nfdv1 "github.com/kubernetes-sigs/node-feature-discovery-operator/api/v1"
)

// log is used to set the logger with a name that describes the actions of
// functions and arguments in this file
var log = logf.Log.WithName("controller_nodefeaturediscovery")

// nfd is an NFD object that will be used to initialize the NFD operator
var nfd NFD

// NodeFeatureDiscoveryReconciler reconciles a NodeFeatureDiscovery object
type NodeFeatureDiscoveryReconciler struct {

	// client.Client reads and writes directly from/to the OCP API server. This
	// field needs to be added to the reconciler because it is responsible for
	// fetching objects from the server, which the NFD operator needs to do in
	// order to add its labels to each node in the cluster.
	client.Client

	// Log is used to log the reconciliation. Every controllers needs this.
	Log logr.Logger

	// Scheme is used by the kubebuilder library to set OwnerReferences. Every
	// controller needs this.
	Scheme *runtime.Scheme

	// Recorder defines interfaces for working with OCP event recorders. This
	// field is needed by NFD in order for NFD to write events.
	Recorder record.EventRecorder

	// AssetsDir defines the directory with assets under the operator image
	AssetsDir string
}

// SetupWithManager sets up the controller with the Manager to create the controller.
// The Manager serves the purpose of initializing shared dependencies (like caches
// and clients) from the 'client.Client' field in the NodeFeatureDiscoveryReconciler
// struct.
func (r *NodeFeatureDiscoveryReconciler) SetupWithManager(mgr ctrl.Manager) error {

	// The predicate package is used by the controller to filter events before
	// they are sent to event handlers. In this case, the "validateUpdateEvent"
	// function is used with UpdateFunc to initate the reconcile loop only on a
	// spec change of the runtime object. (Refer to the "validateUpdateEvent"
	// function in this file for more info.)
	p := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return validateUpdateEvent(&e)
		},
	}

	// To create a new controller, it must be managed by a manager that has
	// already been created. The "For" function is used to to define the type of
	// object being reconciled as "NFD". The "Owns" defines the types of objects
	// being generated: DaemonSet, a Service, a ServiceAccount, some Pods, and
	// some ConfigMaps. Finally, the "Complete" function builds the application
	// controller using the NFD reconciler object.
	return ctrl.NewControllerManagedBy(mgr).
		For(&nfdv1.NodeFeatureDiscovery{}).
		Owns(&appsv1.DaemonSet{}, builder.WithPredicates(p)).
		Owns(&corev1.Service{}, builder.WithPredicates(p)).
		Owns(&corev1.ServiceAccount{}, builder.WithPredicates(p)).
		Owns(&corev1.Pod{}, builder.WithPredicates(p)).
		Owns(&corev1.ConfigMap{}, builder.WithPredicates(p)).
		Complete(r)
}

// validateUpdateEvent looks at an update event and returns true or false
// depending on whether the update event has runtime objects to update.
func validateUpdateEvent(e *event.UpdateEvent) bool {
	if e.ObjectOld == nil {
		klog.Error("Update event has no old runtime object to update")
		return false
	}
	if e.ObjectNew == nil {
		klog.Error("Update event has no new runtime object for update")
		return false
	}

	return true
}

// +kubebuilder:rbac:groups=nfd.kubernetes.io,resources=nodefeaturediscoveries,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=nfd.kubernetes.io,resources=nodefeaturediscoveries/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=nfd.kubernetes.io,resources=nodefeaturediscoveries/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/log,verbs=get
// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=imagestreams/layers,verbs=get
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=events,verbs=list;watch;create;update;patch
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;update;
// +kubebuilder:rbac:groups=core,resources=persistentvolumes,verbs=get;list;watch;create;delete
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=storage.k8s.io,resources=csinodes,verbs=get;list;watch
// +kubebuilder:rbac:groups=storage.k8s.io,resources=storageclasses,verbs=watch
// +kubebuilder:rbac:groups=storage.k8s.io,resources=csidrivers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=endpoints,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.coreos.com,resources=prometheusrules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims
// to move the current state of the cluster closer to the desired state.
func (r *NodeFeatureDiscoveryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("nodefeaturediscovery", req.NamespacedName)

	// Fetch the NodeFeatureDiscovery instance on the cluster. ctx is
	// used to carry a deadline (timeout), a cancellation/abort signal,
	// and similar values across API boundaries so that the
	// reconciliation process can halt if it needs to.
	r.Log.Info("Fetch the NodeFeatureDiscovery instance")
	instance := &nfdv1.NodeFeatureDiscovery{}
	err := r.Get(ctx, req.NamespacedName, instance)

	// If an error occurs because "r.Get" cannot get the NFD instance
	// (e.g., due to timeouts, aborts, etc. defined by ctx), the
	// request likely needs to be requeued.
	if err != nil {
		// handle deletion of resource
		if k8serrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup
			// logic use finalizers. Return and don't requeue.
			r.Log.Info("resource has been deleted", "req", req.Name, "got", instance.Name)
			return ctrl.Result{Requeue: false}, nil
		}

		r.Log.Error(err, "requeueing event since there was an error reading object")
		return ctrl.Result{Requeue: true}, err
	}

	// Apply NFD components upon initializing NFD
	r.Log.Info("Ready to apply components")
	nfd.init(r, instance)

	// For each resource function stored in "nfd.controls", it is
	// necessary to determine if the relevant resource is Ready or
	// NotReady. This loop stops when either a resource is returned
	// as NotReady or when the loop has reached the last element of
	// the nfd.controls list -- determined by nfd.last()
	for {
		err := nfd.step()
		if err != nil {
			return reconcile.Result{}, err
		}
		if nfd.last() {
			break
		}
	}

	return ctrl.Result{}, nil
}
