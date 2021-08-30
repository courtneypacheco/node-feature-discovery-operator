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
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	secv1 "github.com/openshift/api/security/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type assetsFromFile []byte

// Resources holds objects owned by NFD
type Resources struct {
	Namespace                  corev1.Namespace
	ServiceAccount             corev1.ServiceAccount
	Role                       rbacv1.Role
	RoleBinding                rbacv1.RoleBinding
	ClusterRole                rbacv1.ClusterRole
	ClusterRoleBinding         rbacv1.ClusterRoleBinding
	ConfigMap                  corev1.ConfigMap
	DaemonSet                  appsv1.DaemonSet
	Pod                        corev1.Pod
	Service                    corev1.Service
	SecurityContextConstraints secv1.SecurityContextConstraints
}

// Add3dpartyResourcesToScheme Adds 3rd party resources To the operator
func Add3dpartyResourcesToScheme(scheme *runtime.Scheme) error {

	if err := secv1.AddToScheme(scheme); err != nil {
		return err
	}
	return nil
}

func filePathWalkDir(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func getAssetsFrom(path string) []assetsFromFile {

	manifests := []assetsFromFile{}
	assets := path
	files, err := filePathWalkDir(assets)
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		buffer, err := ioutil.ReadFile(file)
		if err != nil {
			panic(err)
		}
		manifests = append(manifests, buffer)
	}
	return manifests
}

func addResourcesControls(path string) (Resources, controlFunc) {
	res := Resources{}
	ctrl := controlFunc{}

	manifests := getAssetsFrom(path)

	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme,
		scheme.Scheme)
	reg, _ := regexp.Compile(`\b(\w*kind:\w*)\B.*\b`)

	for _, m := range manifests {
		kind := reg.FindString(string(m))
		slce := strings.Split(kind, ":")
		kind = strings.TrimSpace(slce[1])

		switch kind {
		case "Namespace":
			_, _, err := s.Decode(m, nil, &res.Namespace)
			panicIfError(err)
			ctrl = append(ctrl, Namespace)
		case "ServiceAccount":
			_, _, err := s.Decode(m, nil, &res.ServiceAccount)
			panicIfError(err)
			ctrl = append(ctrl, ServiceAccount)
		case "ClusterRole":
			_, _, err := s.Decode(m, nil, &res.ClusterRole)
			panicIfError(err)
			ctrl = append(ctrl, ClusterRole)
		case "ClusterRoleBinding":
			_, _, err := s.Decode(m, nil, &res.ClusterRoleBinding)
			panicIfError(err)
			ctrl = append(ctrl, ClusterRoleBinding)
		case "Role":
			_, _, err := s.Decode(m, nil, &res.Role)
			panicIfError(err)
			ctrl = append(ctrl, Role)
		case "RoleBinding":
			_, _, err := s.Decode(m, nil, &res.RoleBinding)
			panicIfError(err)
			ctrl = append(ctrl, RoleBinding)
		case "ConfigMap":
			_, _, err := s.Decode(m, nil, &res.ConfigMap)
			panicIfError(err)
			ctrl = append(ctrl, ConfigMap)
		case "DaemonSet":
			_, _, err := s.Decode(m, nil, &res.DaemonSet)
			panicIfError(err)
			ctrl = append(ctrl, DaemonSet)
		case "Service":
			_, _, err := s.Decode(m, nil, &res.Service)
			panicIfError(err)
			ctrl = append(ctrl, Service)
		case "SecurityContextConstraints":
			_, _, err := s.Decode(m, nil, &res.SecurityContextConstraints)
			panicIfError(err)
			ctrl = append(ctrl, SecurityContextConstraints)

		default:
			log.Info("Unknown Resource: ", "Kind", kind)
		}

	}

	return res, ctrl
}

func panicIfError(err error) {
	if err != nil {
		panic(err)
	}
}

// getServiceAccount gets one of the NFD Operator's ServiceAccounts
func (r *NodeFeatureDiscoveryReconciler) getServiceAccount(ctx context.Context, namespace string, name string) (*corev1.ServiceAccount, error) {
	sa := &corev1.ServiceAccount{}
	err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, sa)
	return sa, err
}

// getDaemonSet gets one of the NFD Operator's DaemonSets
func (r *NodeFeatureDiscoveryReconciler) getDaemonSet(ctx context.Context, namespace string, name string) (*appsv1.DaemonSet, error) {
	ds := &appsv1.DaemonSet{}
	err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, ds)
	return ds, err
}

// getService gets one of the NFD Operator's Services
func (r *NodeFeatureDiscoveryReconciler) getService(ctx context.Context, namespace string, name string) (*corev1.Service, error) {
	svc := &corev1.Service{}
	err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, svc)
	return svc, err
}

// getRole gets one of the NFD Operator's Roles
func (r *NodeFeatureDiscoveryReconciler) getRole(ctx context.Context, namespace string, name string) (*rbacv1.Role, error) {
	role := &rbacv1.Role{}
	err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, role)
	return role, err
}

// getRoleBinding gets one of the NFD Operator's RoleBindings
func (r *NodeFeatureDiscoveryReconciler) getRoleBinding(ctx context.Context, namespace string, name string) (*rbacv1.RoleBinding, error) {
	rb := &rbacv1.RoleBinding{}
	err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, rb)
	return rb, err
}

// getClusterRole gets one of the NFD Operator's ClusterRoles
func (r *NodeFeatureDiscoveryReconciler) getClusterRole(ctx context.Context, namespace string, name string) (*rbacv1.ClusterRole, error) {
	cr := &rbacv1.ClusterRole{}
	err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, cr)
	return cr, err
}

// getClusterRoleBinding gets one of the NFD Operator's ClusterRoleBindings
func (r *NodeFeatureDiscoveryReconciler) getClusterRoleBinding(ctx context.Context, namespace string, name string) (*rbacv1.ClusterRoleBinding, error) {
	crb := &rbacv1.ClusterRoleBinding{}
	err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, crb)
	return crb, err
}

// getSecurityContextConstraints gets one of the NFD Operator's SecurityContextConstraints
func (r *NodeFeatureDiscoveryReconciler) getSecurityContextConstraints(ctx context.Context, namespace string, name string) (*secv1.SecurityContextConstraints, error) {
	scc := &secv1.SecurityContextConstraints{}
	err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, scc)
	return scc, err
}
