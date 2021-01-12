![logo](docs/CarvelLogo.png)

# vendir

- Website: [https://carvel.dev/vendir](https://carvel.dev/vendir)
- Slack: [#carvel in Kubernetes slack](https://slack.kubernetes.io)
- [Docs](https://carvel.dev/vendir/docs/latest/) with example workflow and other details
- Install: Grab prebuilt binaries from the [Releases page](https://github.com/vmware-tanzu/carvel-vendir/releases) or [Homebrew Carvel tap](https://github.com/vmware-tanzu/homebrew-carvel)

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

### Join the Community and Make Carvel Better
Carvel is better because of our contributors and maintainers. It is because of you that we can bring great software to the community.
Please join us during our online community meetings ([Zoom link](http://community.klt.rip/)) every other Wednesday at 12PM ET / 9AM PT and catch up with past meetings on the [VMware YouTube Channel](https://www.youtube.com/playlist?list=PL7bmigfV0EqQ_cDNKVTIcZt-dAM-hpClS).
Join [Google Group](https://groups.google.com/g/carvel-dev) to get updates on the project and invites to community meetings.
You can chat with us on Kubernetes Slack in the #carvel channel and follow us on Twitter at @carvel_dev.

Check out which organizations are using and contributing to Carvel: [Adopter's list](https://github.com/vmware-tanzu/carvel/ADOPTERS.md)

## Development

```bash
./hack/build.sh
./hack/test-all.sh
```
