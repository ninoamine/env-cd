# Env-CD

Env-CD is a Kubernetes controller project designed to automate the deployment of complete environments. This includes deploying services (via Helm charts) as well as their associated datasources such as PostgreSQL databases, MongoDB topics, Kafka topics, and more.

> **Note:** The PostgreSQL and Kafka controllers do not create actual servers or clusters; they manage resources within existing services.

## 🎯 Project Objective

The main goal of this project is to provide declarative environment management in Kubernetes:

* Deploy services through Helm charts.
* Manage datasources and infrastructure resources (PostgreSQL schemas, MongoDB topics, Kafka topics, etc.).
* Offer reusable controllers for various infrastructure components.

## 🛠 Features / Current Scope

* Environment controller for deploying complete environments.
* Controllers for managing PostgreSQL resources.
* Controllers for managing Kafka topics.
* Modular design to easily add new controllers for other resources in the future.

## 🚀 Roadmap

### Phase 1 – Basic Environment Controller

* Create `Environment` CRD and controller.
* Implement minimal Reconcile logic (logging creation/update of Environment resources).
* Set up project structure and manager in Kubebuilder.

### Phase 2 – Datasource Controllers

* PostgreSQL: manage databases, schemas, and users within an existing server.
* Kafka: manage topics and configurations within an existing cluster.
* MongoDB: manage collections and topics.

### Phase 3 – Service Deployment

* Helm chart integration to deploy services declaratively.
* Support for environment-specific values and configurations.

### Phase 4 – Advanced Features

* Status management and metrics for each environment.
* Automated cleanup and resource lifecycle management.
* Multi-environment support and resource grouping.

## 📚 How to Use

1. Install the CRDs in your cluster:
```bash
make install
```

2. Run the controller locally:
```bash
make run
```

3. Create an `Environment` resource:
```yaml
apiVersion: core.envcd.io/v1alpha1
kind: Environment
metadata:
  name: dev-env
spec: {}
```
```bash
kubectl apply -f config/samples/core_v1alpha1_environment.yaml
```

4. Check the logs to see the reconciliation in action.

## ⚡ Tech Stack

* **Kubernetes** and **Kubebuilder**
* **Go** for controller logic
* **Helm** for service deployment
* **PostgreSQL**, **Kafka**, **MongoDB** for datasources management