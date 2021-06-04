/*
Copyright 2021.

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
	"log"

	"github.com/go-logr/logr"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	meridiov1alpha1 "github.com/nordix/meridio-operator/api/v1alpha1"
)

// TrenchReconciler reconciles a Trench object
type TrenchReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

const MeridioConfigKey = "meridio.conf"

type Config struct {
	VIPs []string `yaml:"vips"`
}

//+kubebuilder:rbac:groups=meridio.nordix.org,resources=trenches,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=meridio.nordix.org,resources=trenches/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=meridio.nordix.org,resources=trenches/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Trench object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *TrenchReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("trench", req.NamespacedName)

	trench := &meridiov1alpha1.Trench{}
	err := r.Get(ctx, req.NamespacedName, trench)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Check if the configmap already exists, if not create a new one
	configMap := &corev1.ConfigMap{}
	err = r.Client.Get(ctx, client.ObjectKey{Namespace: trench.Namespace, Name: trench.Spec.ConfigMapName}, configMap)
	if apierrors.IsNotFound(err) {
		configMap = r.buildConfigMap(trench)
		err := r.Client.Create(ctx, configMap)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	// Ensure the configmap contains the same data as the spec
	if !configMapValid(trench, configMap) {
		r.setConfigMapData(trench, configMap)
		err = r.Update(ctx, configMap)
		if err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (r *TrenchReconciler) setConfigMapData(trench *meridiov1alpha1.Trench, configMap *corev1.ConfigMap) {
	config := &Config{
		VIPs: trench.Spec.VIPs,
	}
	configYAML, err := yaml.Marshal(&config)
	if err != nil {
		log.Fatalf("error yaml.Marshal: %v", err)
	}
	configMap.Data = map[string]string{
		MeridioConfigKey: string(configYAML),
	}
	configMap.Data[MeridioConfigKey] = string(configYAML)
}

func (r *TrenchReconciler) buildConfigMap(trench *meridiov1alpha1.Trench) *corev1.ConfigMap {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      trench.Spec.ConfigMapName,
			Namespace: trench.Namespace,
		},
	}
	r.setConfigMapData(trench, configMap)

	controllerutil.SetControllerReference(trench, configMap, r.Scheme)
	return configMap
}

func configMapValid(trench *meridiov1alpha1.Trench, configMap *corev1.ConfigMap) bool {
	configuration, ok := configMap.Data[MeridioConfigKey]
	if !ok {
		return false
	}
	config := &Config{}
	err := yaml.Unmarshal([]byte(configuration), &config)
	if err != nil {
		return false
	}
	if !vipListsEqual(config.VIPs, trench.Spec.VIPs) {
		return false
	}
	return true
}

func vipListsEqual(vipListA []string, vipListB []string) bool {
	vipsA := make(map[string]struct{})
	vipsB := make(map[string]struct{})
	for _, vip := range vipListA {
		vipsA[vip] = struct{}{}
	}
	for _, vip := range vipListB {
		vipsB[vip] = struct{}{}
	}
	for _, vip := range vipListA {
		if _, ok := vipsB[vip]; !ok {
			return false
		}
	}
	for _, vip := range vipListB {
		if _, ok := vipsA[vip]; !ok {
			return false
		}
	}
	return true
}

// SetupWithManager sets up the controller with the Manager.
func (r *TrenchReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&meridiov1alpha1.Trench{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}
