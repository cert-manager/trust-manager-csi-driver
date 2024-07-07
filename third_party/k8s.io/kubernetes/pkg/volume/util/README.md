# k8s.io/kubernetes/pkg/volume/util

The files in this directory are adapted from the [Kubernetes codebase](https://github.com/kubernetes/kubernetes/blob/fea466ea7b50462a77042a5133377aedc86eab70/pkg/volume/util). This is because the Kubernetes codebase is a nightmare to import and would be a huge dependency for such a small package.

The following changes have also been applied:
- Support setting of GID of created files
- Get log from context instead of using the global logger (so we can use our logger instead of being locked into klog)