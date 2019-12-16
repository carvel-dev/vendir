# vendir

- Slack: [#k14s in Kubernetes slack](https://slack.kubernetes.io)
- [Docs](docs/README.md) with example workflow and other details
- Install: Grab prebuilt binaries from the [Releases page](https://github.com/k14s/vendir/releases)

`vendir` allows to declaratively state what should be in a directory. It's could be used for vendoring software.

```bash
$ vendir sync # from a directory that contains vendir.yml
```

Features:

- Pull Git repositories at particular revision
- Keep only particular portions of a pulled repository
- State which directories are manually managed
- Keep common legal files (LICENSE, etc.)

## Development

```bash
./hack/build.sh
./hack/test-all.sh
```
