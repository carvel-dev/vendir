# vendir

- Slack: [#k14s in Kubernetes slack](https://slack.kubernetes.io)
- [Docs](docs/README.md) with example workflow and other details
- Install: Grab prebuilt binaries from the [Releases page](https://github.com/k14s/vendir/releases)

`vendir` allows to declaratively state what should be in a directory. It's could be used for vendoring software.

```bash
$ vendir sync # from a directory that contains vendir.yml
```

Example: [examples/git-and-manual](examples/git-and-manual).

Features:

- Various sources
  - Pull Git repositories at particular revision
  - Pull Github release at particular version
- Keep only particular portions of pulled content
- State which directories are manually managed
- Keep common legal files (LICENSE, etc.)

## Development

```bash
./hack/build.sh
./hack/test-all.sh
```
