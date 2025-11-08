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
	"github.com/jackc/pgx/v5/pgxpool"
	corev1alpha1 "github.com/ninoamine/env-cd/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"regexp"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"net/url"
)

// PostgresqlDatabaseReconciler reconciles a PostgresqlDatabase object
type PostgresqlDatabaseReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	pgConfig *PostgresqlConfiguration
	pgPool   *pgxpool.Pool
}

// +kubebuilder:rbac:groups=core.envcd.io,resources=postgresqldatabases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core.envcd.io,resources=postgresqldatabases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core.envcd.io,resources=postgresqldatabases/finalizers,verbs=update

const databaseFinalizer = "finalizer.postgresqldatabase.core.envcd.io"

func (r *PostgresqlDatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var dbRes corev1alpha1.PostgresqlDatabase
	if err := r.Get(ctx, req.NamespacedName, &dbRes); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil // CR déjà supprimé
		}
		return ctrl.Result{}, err
	}

	// --- Cas 1 : suppression ---
	if !dbRes.ObjectMeta.DeletionTimestamp.IsZero() {
		logger.Info("PostgresqlDatabase is being deleted", "name", dbRes.Name)

		if controllerutil.ContainsFinalizer(&dbRes, databaseFinalizer) {
			// Supprime la DB PostgreSQL
			if err := r.deleteDatabase(ctx, dbRes.Name); err != nil {
				return ctrl.Result{}, err
			}

			// Retire le finalizer pour autoriser la suppression du CR
			controllerutil.RemoveFinalizer(&dbRes, databaseFinalizer)
			if err := r.Update(ctx, &dbRes); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// --- Cas 2 : création ou mise à jour ---
	if !controllerutil.ContainsFinalizer(&dbRes, databaseFinalizer) {
		controllerutil.AddFinalizer(&dbRes, databaseFinalizer)
		if err := r.Update(ctx, &dbRes); err != nil {
			return ctrl.Result{}, err
		}
	}

	logger.Info("Ensuring PostgreSQL database exists", "name", dbRes.Name)
	if err := r.createDatabase(ctx, dbRes.Name); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *PostgresqlDatabaseReconciler) deleteDatabase(ctx context.Context, dbName string) error {
	if !validIdentifier.MatchString(dbName) {
		return fmt.Errorf("invalid database name: %s", dbName)
	}

	query := fmt.Sprintf(`DROP DATABASE IF EXISTS "%s";`, dbName)
	_, err := r.pgPool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to delete PostgreSQL database %s: %v", dbName, err)
	}

	return nil
}

var validIdentifier = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func (r *PostgresqlDatabaseReconciler) createDatabase(ctx context.Context, dbName string) error {
	if !validIdentifier.MatchString(dbName) {
		return fmt.Errorf("invalid database name: %s", dbName)
	}

	// Vérifie si la DB existe déjà
	checkQuery := `SELECT 1 FROM pg_database WHERE datname = $1;`
	var exists int
	err := r.pgPool.QueryRow(ctx, checkQuery, dbName).Scan(&exists)
	if err == nil {
		// La DB existe déjà
		return nil
	}

	// Crée la DB
	createQuery := fmt.Sprintf(`CREATE DATABASE "%s";`, dbName)
	_, err = r.pgPool.Exec(ctx, createQuery)
	if err != nil {
		return fmt.Errorf("failed to create PostgreSQL database %s: %v", dbName, err)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PostgresqlDatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	cfg, err := LoadPostgresqlConfiguration()
	if err != nil {
		return err
	}
	r.pgConfig = cfg

	escaped_password := url.QueryEscape(r.pgConfig.Password)

	dsn := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s",
		r.pgConfig.User,
		escaped_password,
		r.pgConfig.Host,
		r.pgConfig.Port,
		r.pgConfig.Database,
	)

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return fmt.Errorf("unable to create PostgreSQL connection pool: %v", err)
	}
	r.pgPool = pool

	if err := mgr.Add(manager.RunnableFunc(func(ctx context.Context) error {
		<-ctx.Done() // Attendre le signal de shutdown
		r.pgPool.Close()
		return nil
	})); err != nil {
		return fmt.Errorf("unable to add PostgreSQL cleanup runnable: %v", err)
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.PostgresqlDatabase{}).
		Named("postgresqldatabase").
		Complete(r)
}
