# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Crane Autoscaler is a Kubernetes operator that coordinates HPA and VPA on the same workload. It prevents conflicts by enabling only one autoscaler at a time, switching between them based on resource utilization thresholds.

Built with Kubebuilder v4, Go 1.23, and controller-runtime v0.19.

## Common Commands

```bash
# Development environment (Nix)
nix develop

# Build
make build                    # Build binary to bin/manager
make docker-build IMG=<img>   # Build container image

# Code generation (run after modifying API types in api/v1alpha1/)
make manifests generate fmt

# Testing
make test                                     # Unit tests (uses envtest, excludes e2e)
make test-e2e                                 # E2E tests (requires Kind cluster)
make up                                       # Create Kind cluster for e2e
make down                                     # Destroy Kind cluster
go test ./internal/controller/ -run TestName  # Run a single test

# Linting
make lint                     # pre-commit hooks + golangci-lint
make lint-fix                 # Auto-fix lint issues
make fmt                      # Format Go + YAML files
```

## Architecture

### CRD: `CranePodAutoscaler`

Defined in `api/v1alpha1/cranepodautoscaler_types.go`. Wraps both an HPA spec and a VPA spec plus a `Behavior` section with `VPACapacityThresholdPercent` (default 80%).

Supporting files in the same package:
- `validation.go` - Validation logic (HPA and VPA must target same workload, `minReplicas` required)
- `autoscaler_generators.go` - Generates enabled/disabled variants of HPA and VPA resources
- `cranepodautoscaler_webhook.go` - Defaulting and validating admission webhooks

### Controller: `CranePodAutoscalerReconciler`

Single controller in `internal/controller/cranepodautoscaler_controller.go`. Owns both HPA and VPA resources (cascade delete). The scaling decision logic:

1. **Initial state / just created**: Defaults to HPA active (safer for availability)
2. **VPA active**: Switches to HPA when VPA target/upperBound ratio exceeds threshold
3. **HPA active**: Switches to VPA when HPA is at min replicas AND VPA recommendation is below threshold

Disabling VPA = set `UpdateMode` to `Off`. Disabling HPA = set `maxReplicas` = `minReplicas`.

### Testing

Tests use Ginkgo v2 / Gomega. Unit tests in `internal/controller/` use envtest (embedded kube-apiserver). E2E tests in `test/e2e/` run against a Kind cluster. CI tests against K8s versions 1.28 through 1.32.

### Code Generation

Kubebuilder markers in source files drive generation of CRDs, RBAC rules, webhook configs, and DeepCopy methods. Always run `make manifests generate` after changing API types or RBAC markers.
