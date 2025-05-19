# 🧠 Cluster Inspector Operator

A Kubernetes Operator built with Kubebuilder that uses an AI model (via [Ollama](https://ollama.com)) to run periodic inspections on your cluster. It evaluates system status, pod behavior, and more to provide intelligent summaries and recommendations.

## 📦 Features

- Custom resource `InspectionReport` to trigger inspections
- Periodic re-evaluation using controller requeue
- AI-powered analysis using a local Ollama model (e.g., `llama3`)
- Extendable to include logs, events, and metrics

---

## 🚀 Getting Started

### 🧱 Prerequisites

- [Go](https://golang.org/) >= 1.20
- [Docker](https://www.docker.com/)
- [Kubebuilder](https://book.kubebuilder.io/quick-start.html)
- [K3D](https://k3d.io/) or a Kubernetes cluster
- [Ollama](https://ollama.com) running locally or in the cluster

### 🧰 Clone and Build

```bash
git clone https://github.com/your-org/cluster-inspector-operator.git
cd cluster-inspector-operator

# Build and push the Docker image
make docker-build docker-push IMG=cluster-inspector-operator:dev

# Load into K3D
k3d image import cluster-inspector-operator:dev -c <your-cluster-name>

# Deploy the operator
make deploy IMG=cluster-inspector-operator:dev
