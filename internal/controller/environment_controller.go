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
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"fmt"
	"time"

	corev1alpha1 "github.com/ninoamine/env-cd/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EnvironmentReconciler reconciles a Environment object
type EnvironmentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=core.envcd.io,resources=environments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core.envcd.io,resources=environments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core.envcd.io,resources=environments/finalizers,verbs=update

const environmentFinalizer = "finalizer.environment.core.envcd.io"

func (r *EnvironmentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var environment corev1alpha1.Environment
	if err := r.Get(ctx, req.NamespacedName, &environment); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !environment.ObjectMeta.DeletionTimestamp.IsZero() {
		logger.Info("Environment is being deleted", "name", environment.Name)

		if controllerutil.ContainsFinalizer(&environment, environmentFinalizer) {
			controllerutil.RemoveFinalizer(&environment, environmentFinalizer)
			if err := r.Update(ctx, &environment); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(&environment, environmentFinalizer) {
		controllerutil.AddFinalizer(&environment, environmentFinalizer)
		if err := r.Update(ctx, &environment); err != nil {
			return ctrl.Result{}, err
		}
	}

	environment.Status.Errors = []string{}
	hasError := false

	for _, pg := range environment.Spec.Databases.Postgresql {
		logger.Info("Postgresql Database to be created", "database_name", pg.Name)

		found := &corev1alpha1.PostgresqlDatabase{}
		err := r.Get(ctx, client.ObjectKey{Name: pg.Name, Namespace: req.Namespace}, found)

		if err != nil && apierrors.IsNotFound(err) {
			// N'existe pas, on crée
			db := &corev1alpha1.PostgresqlDatabase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pg.Name,
					Namespace: req.Namespace,
					OwnerReferences: []metav1.OwnerReference{
						*metav1.NewControllerRef(&environment, corev1alpha1.GroupVersion.WithKind("Environment")),
					},
				},
				Spec: corev1alpha1.PostgresqlDatabaseSpec{
					Name: pg.Name,
				},
			}

			err = r.Create(ctx, db)
			if err != nil {
				logger.Error(err, "Failed to create PostgresqlDatabase", "database_name", pg.Name)
				environment.Status.Errors = append(environment.Status.Errors, fmt.Sprintf("Failed to create PostgresqlDatabase %s: %v", pg.Name, err))
				hasError = true
				continue
			}
			logger.Info("PostgresqlDatabase created successfully", "database_name", pg.Name)

		} else if err != nil {
			logger.Error(err, "Failed to get PostgresqlDatabase", "database_name", pg.Name)
			environment.Status.Errors = append(environment.Status.Errors, fmt.Sprintf("Failed to get PostgresqlDatabase %s: %v", pg.Name, err))
			hasError = true
			continue

		} else {
			logger.Info("PostgresqlDatabase already exists", "database_name", pg.Name)
		}
	}

	if hasError {
		environment.Status.Status = "Error"
		environment.Status.Message = "One or more errors occurred during reconciliation"
	} else {
		environment.Status.Status = "Success"
		environment.Status.Message = "Reconciliation completed successfully"
	}

	if err := r.Status().Update(ctx, &environment); err != nil {
		logger.Error(err, "unable to update Environment status")
	}

	if hasError {
		return ctrl.Result{RequeueAfter: time.Second * 30}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EnvironmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.Environment{}).
		Named("environment").
		Complete(r)
}
