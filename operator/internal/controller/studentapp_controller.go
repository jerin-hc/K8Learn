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
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	studentappv1 "ij3rry.com/student-operator/api/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// StudentAppReconciler reconciles a StudentApp object
type StudentAppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=app.ij3rry.com,resources=studentapps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=app.ij3rry.com,resources=studentapps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=app.ij3rry.com,resources=studentapps/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the StudentApp object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.24.1/pkg/reconcile
func (r *StudentAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	app := &studentappv1.StudentApp{}
	if err := r.Get(ctx, req.NamespacedName, app); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	dbService := &corev1.Service{}
	ready, err := r.recouncilerPostgres(ctx, app, dbService, log)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !ready {
		log.Info("Postgres not ready yet, requeuing...")
		return ctrl.Result{
			RequeueAfter: 2 * time.Second,
		}, nil
	}
	ready, err = r.reconcilerStudentApp(ctx, app, dbService, log)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !ready {
		log.Info("StudentApp not ready yet, requeuing...")
		return ctrl.Result{
			RequeueAfter: 2 * time.Second,
		}, nil
	}
	log.Info("StudentApp successfuly deployed...")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *StudentAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&studentappv1.StudentApp{}).
		Named("studentapp").
		Complete(r)
}

func generateMeta(kind string, app *studentappv1.StudentApp) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      getResourceName(kind, app.Name),
		Namespace: app.Namespace,
		OwnerReferences: []metav1.OwnerReference{
			*metav1.NewControllerRef(app, app.GroupVersionKind()),
		},
	}
}

func getResourceName(kind string, name string) string {
	return fmt.Sprintf("%s-%s", name, kind)
}
