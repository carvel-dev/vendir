## `vendir.yml` spec

```yaml
apiVersion: vendir.k14s.io/v1alpha1
kind: Config

# one or more directories to manage with vendir
directories:
- # path is relative to vendir.yml location
  path: config/_ytt_lib

  contents:
  - # path lives relative to directory path # (required)
    path: github.com/cloudfoundry/cf-k8s-networking

    # states that directory specified by above path
    # is managed by hand; nothing to do for vendir (optional)
    manual: {}

    # uses git to clone repository (optional)
    git:
      # http or ssh urls are supported (required)
      url: https://github.com/cloudfoundry/cf-k8s-networking
      # branch, tag, commit; origin is the name of the remote (required)
      ref: origin/master

    # fetches assets from a github release
    githubRelease:
      # slug for repository (org/repo) (required)
      slug: k14s/kapp-controller
      # release tag (required)
      tag: v0.1.0
      # disables checking checksums for downloaded assets (optional)
      # checksums are found within release's body in following format
      # `<sha256>  <filename>`
      disableChecksumValidation: true
      # specifies which archive to unpack for contents (optional)
      unpackArchive:
        path: release.tgz

    # copy contents from local directory (optional)
    directory:
      # local file system path relative to vendir.yml
      path: some-path

    # includes paths specify what should be included. by default
    # all paths are included (optional)
    includePaths:
    - cfroutesync/crds/**/*
    - install/ytt/networking/**/*

    # exclude paths are "placed" on top of include paths (optional)
    excludePaths: []

    # specifies paths to files that need to be includes for
    # legal reasons such as LICENSE file. Defaults to few 
    # LICENSE, NOTICE and COPYRIGHT variations (optional)
    legalPaths: []
```
