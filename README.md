# Kubernetes Cluster API Provider for Proxmox - CAPMOX

[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=ionos-cloud_cluster-api-provider-proxmox&metric=alert_status&token=fb1b4c0a87d83a780c76c21be0f89dc13efc2ca0)](https://sonarcloud.io/summary/new_code?id=ionos-cloud_cluster-api-provider-proxmox)

## Overview

The [Cluster API](https://github.com/kubernetes-sigs/cluster-api) brings declarative, Kubernetes-style APIs to cluster creation, configuration and management.
Cluster API Provider for Proxmox is a concrete implementation of Cluster API for Proxmox VE.

## Launching a Kubernetes cluster on Proxmox

Check out the [quickstart guide](./docs/Usage.md#quick-start) for launching a cluster on Proxmox.

## Compatibility with Cluster API and Kubernetes Versions
This provider's versions are compatible with the following versions of Cluster API:

|                        | Cluster API v1beta1 (v1.3) | Cluster API v1beta1 (v1.4) | Cluster API v1beta1 (v1.5) | Cluster API v1beta1 (v1.6) |
|------------------------|:--------------------------:|:--------------------------:|:--------------------------:|:--------------------------:|
| CAPMOX v1alpha1 (v0.1) |             ✓              |             ✓              |             ✓              |             ☓              |

(See [Kubernetes support matrix](https://cluster-api.sigs.k8s.io/reference/versions.html) of Cluster API versions).

## Documentation

Further documentation is available in the `/docs` directory.

## Security

We take security seriously.
Please read our [security policy](SECURITY.md) for information on how to report security issues.
