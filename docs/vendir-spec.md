## `vendir.yml` spec

```yaml
apiVersion: vendir.k14s.io/v1alpha1
kind: Config

# declaration of minimum required vendir binary version (optional)
minimumRequiredVersion: 0.8.0

# one or more directories to manage with vendir
directories:
- # path is relative to vendir.yml location
  path: config/_ytt_lib

  contents:
  - # path lives relative to directory path # (required)
    path: github.com/cloudfoundry/cf-k8s-networking

    # uses git to clone repository (optional)
    git:
      # http or ssh urls are supported (required)
      url: https://github.com/cloudfoundry/cf-k8s-networking
      # branch, tag, commit; origin is the name of the remote (required)
      # optional if refSelection is specified (available in v0.11.0+)
      ref: origin/master
      # specifies a strategy to resolve to an explicit ref (optional; v0.11.0+)
      refSelection:
        semver:
          # list of semver constraints (see versions.md for details) (required)
          constraints: ">0.4.0"
      # skip downloading lfs files (optional)
      lfsSkipSmudge: false
      # specifies name of a secret with auth details;
      # secret may include 'ssh-privatekey', 'ssh-knownhosts',
      # 'username', 'password' keys (optional)
      secretRef:
        # (required)
        name: my-git-auth

    # fetches asset over HTTP (optional)
    http:
      # asset URL (required)
      url: 
      # verification checksum (optional)
      sha256: ""
      # specifies name of a secret with basic auth details;
      # secret may include 'username', 'password' keys (optional)
      secretRef:
        # (required)
        name: my-http-auth

    # fetches asset from an image registry (optional; v0.11.0+)
    image:
      # image URL; could be plain, tagged or digest reference (required)
      url: gcr.io/repo/image:v1.0.0
      # specifies name of a secret with registry auth details;
      # secret may include 'username', 'password' and/or 'token' keys (optional)
      secretRef:
        # (required)
        name: my-image-auth

    # fetches assets from a github release (optional)
    githubRelease:
      # slug for repository (org/repo) (required)
      slug: k14s/kapp-controller
      # use release tag (optional)
      tag: v0.1.0
      # use latest published version (optional)
      latest: true
      # use exact release URL (optional)
      url: https://api.github.com/repos/k14s/kapp-controller/releases/21912613
      # checksums for downloaded files (optional)
      # (if release text body contains checksums, it's not necessary
      # to manually specify them here)
      checkums:
        release.yml: 26bf09c42d72ae448af3d1ee9f6a933c87c4ec81d04d37b30e1b6a339f5983a7
      # disables checking auto-found checksums for downloaded files (optional)
      # (checksums are extracted from release's text body
      # based on following format `<sha256>  <filename>`)
      disableAutoChecksumValidation: true
      # specifies which archive to unpack for contents (optional)
      unpackArchive:
        # (required)
        path: release.tgz
      # specifies name of a secret with github auth details;
      # secret may include 'token' key (optional)
      secretRef:
        # (required)
        name: my-gh-auth

    # fetch Helm chart contents (optional; v0.11.0+)
    helmChart:
      # chart name (required)
      name: stable/redis
      # use specific chart version (string; optional)
      version: "1.2.1"
      # specifies Helm repository to fetch from (optional)
      repository:
        # repository url (required)
        url: https://...
        # specifies name of a secret with helm repo auth details;
        # secret may include 'username', 'password' (optional)
        secretRef:
          # (required)
          name: my-helm-auth
      # specify helm binary version to use;
      # '3' means binary 'helm3' needs to be on the path (optional)
      helmVersion: "3"

    # copy contents from local directory (optional)
    directory:
      # local file system path relative to vendir.yml
      path: some-path

    # states that directory specified by above path
    # is managed by hand; nothing to do for vendir (optional)
    manual: {}

    # specify contents inline within this file (optional; v0.11.0+)
    inline:
      # specifies mapping of paths to their content (optional)
      paths:
        dir/file.ext: file-content
      # specifies content via secrets and config maps (optional)
      pathsFrom:
      - secretRef:
          # (required)
          name: secret-name
          # specifies where to place files found in secret (optional)
          directoryPath: dir
      - configMapRef:
          # (required)
          name: cfgmap-name
          # specifies where to place files found in config map (optional)
          directoryPath: dir

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

    # make subdirectory to be new root path within this asset (optional; v0.11.0+)
    newRootPath: cfroutesync
```
