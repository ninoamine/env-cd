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
	"sigs.k8s.io/controller-runtime/pkg/log"

	"fmt"
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

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Environment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *EnvironmentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var environment corev1alpha1.Environment
	if err := r.Get(ctx, req.NamespacedName, &environment); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Environment deleted", "name", req.Name, "namespace", req.Namespace)
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get Environment")
		return ctrl.Result{}, err
	}

	logger.Info("Reconciling Environment",
		"name", environment.Name,
		"namespace", environment.Namespace,
		"status", environment.Status.Status,
	)

	// Réinitialise les erreurs avant reconciliation
	environment.Status.Errors = []string{}

	hasError := false

	for _, pg := range environment.Spec.Databases.Postgresql {
		logger.Info("Postgresql Database to be created",
			"database_name", pg.Name,
		)

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

		err := r.Create(ctx, db)
		if err != nil {
			if apierrors.IsAlreadyExists(err) {
				logger.Info("PostgresqlDatabase already exists", "database_name", pg.Name)
				environment.Status.Errors = append(environment.Status.Errors, fmt.Sprintf("PostgresqlDatabase %s already exists", pg.Name))
			} else {
				logger.Error(err, "Failed to create PostgresqlDatabase", "database_name", pg.Name)
				environment.Status.Errors = append(environment.Status.Errors, fmt.Sprintf("Failed to create PostgresqlDatabase %s: %v", pg.Name, err))
			}
			hasError = true
			continue
		}

		logger.Info("PostgresqlDatabase created successfully", "database_name", pg.Name)
	}

	// Mise à jour du status globale
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

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EnvironmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.Environment{}).
		Named("environment").
		Complete(r)
}
