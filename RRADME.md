# GitLab Runner External Scaler for KEDA

This project provides a **KEDA External Scaler** to dynamically scale **GitLab Runners** running on Kubernetes based on the number of pending jobs in GitLab.

## Features
- Monitors pending jobs via the **GitLab API**.
- Scales runners based on a configurable **pending jobs per runner** ratio.
- Deployable via **Helm**.
- Designed for **distroless nonroot** containers.
- Fully compatible with **KEDA’s external scaler interface**.

## Configuration

| Env Var                  | Description                                        |
|--------------------------|----------------------------------------------------|
| `GITLAB_TOKEN`            | GitLab Personal Access Token (read_api scope)     |
| `GITLAB_RUNNER_ID`        | Target GitLab Runner ID                           |
| `PENDING_JOBS_PER_RUNNER` | Number of pending jobs each runner can handle     |

## Quick Start

1. **Build & Push Image:**
    ```sh
    docker build -t your-registry/gitlab-runner-scaler:latest .
    docker push your-registry/gitlab-runner-scaler:latest
    ```

2. **Install Helm Chart:**
    ```sh
    helm install gitlab-runner-scaler ./helm-chart \
        --set image.repository=your-registry/gitlab-runner-scaler \
        --set gitlab.token=$(echo -n "your-token" | base64) \
        --set gitlab.runnerID="1234" \
        --set gitlab.pendingJobsPerRunner=10
    ```

3. **Monitor Scaling:**
    ```sh
    kubectl get pods
    kubectl describe scaledobject gitlab-runner-scaler
    ```

