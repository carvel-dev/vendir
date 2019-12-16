## Docs

`vendir.yml` example:

```yaml
apiVersion: vendir.k14s.io/v1alpha1
kind: Config
directories:
- path: config/_ytt_lib
  contents:
  - path: github.com/cloudfoundry/cf-k8s-networking
    git:
      url: https://github.com/cloudfoundry/cf-k8s-networking
      ref: origin/master
    includePaths:
    - cfroutesync/crds/**/*
    - install/ytt/networking/**/*
    excludePaths: []
  - path: github.com/cloudfoundry/cc
    manual: {}
  - path: github.com/cloudfoundry/uaa
    manual: {}
  - path: github.com/cloudfoundry/db
    manual: {}
  - path: github.com/GoogleCloudPlatform/metacontroller
    manual: {}
```

generated `vendir.lock.yml` example:

```yaml
apiVersion: vendir.k14s.io/v1alpha1
kind: LockConfig
directories:
- path: config/_ytt_lib
  contents:
  - path: github.com/cloudfoundry/cf-k8s-networking
    git:
      sha: 2b009b61fa8afb330a4302c694ee61b11104c54c
      commit_title: "feat: add /metrics prometheus scrapable endpoint "
  - path: github.com/cloudfoundry/cc
    manual: {}
  - path: github.com/cloudfoundry/uaa
    manual: {}
  - path: github.com/cloudfoundry/db
    manual: {}
  - path: github.com/GoogleCloudPlatform/metacontroller
    manual: {}
```

Try out example in [examples/git-and-manual](../examples/git-and-manual) directory.
