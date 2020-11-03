# vendir

- Slack: [#carvel in Kubernetes slack](https://slack.kubernetes.io)
- [Docs](docs/README.md) with example workflow and other details
- Install: Grab prebuilt binaries from the [Releases page](https://github.com/k14s/vendir/releases) or [Homebrew k14s tap](https://github.com/k14s/homebrew-tap)

`vendir` allows to declaratively state what should be in a directory. It's could be used for vendoring software.

```bash
$ vendir sync # from a directory that contains vendir.yml
```

Features:

- Various sources
  - pull Git repositories ([examples/git](examples/git), semver tag resolution in [examples/versionselection](examples/versionselection))
    - including tag semver selection, GPG verification
  - pull Github releases ([examples/github-release](examples/github-release))
  - pull HTTP asset ([examples/http](examples/http))
  - pull Docker image contents ([examples/image](examples/image))
  - pull Helm chart contents ([examples/helm-chart](examples/helm-chart))
  - ...let us know sources you'd like to see
- Keep only particular portions of pulled content via includePaths/excludePaths or newRootPath
- Override specific directory with a local directory source for quick development
- State which directories are manually managed
- Specify inline content for a directory
- Generates lock file
- Keep common legal files (LICENSE, etc.)

See [all examples](examples/).

## Development

```bash
./hack/build.sh
./hack/test-all.sh
```
