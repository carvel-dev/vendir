![logo](docs/CarvelLogo.png)

[![OpenSSF Best Practices](https://bestpractices.coreinfrastructure.org/projects/7746/badge)](https://bestpractices.coreinfrastructure.org/projects/7746)

# vendir

- Website: [https://carvel.dev/vendir](https://carvel.dev/vendir)
- Slack: [#carvel in Kubernetes slack](https://slack.kubernetes.io)
- [Docs](https://carvel.dev/vendir/docs/latest/) with example workflow and other details
- Install: Grab prebuilt binaries from the [Releases page](https://github.com/carvel-dev/vendir/releases) or [Homebrew Carvel tap](https://github.com/carvel-dev/homebrew)
- Backlog: [See what we're up to](https://github.com/orgs/carvel-dev/projects/1/views/1?filterQuery=repo%3A%22carvel-dev%2Fvendir%22).

`vendir` allows you to declaratively state what should be in a directory. It can be used for vendoring software.

```bash
$ vendir sync # from a directory that contains vendir.yml
```

Features:

- Various sources
  - pull Git repositories ([examples/git](examples/git), semver tag resolution in [examples/versionselection](examples/versionselection))
    - including tag semver selection, GPG verification
  - pull Github releases ([examples/github-release](examples/github-release), semver tag resolution in [examples/versionselection](examples/versionselection))
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
Please join us during our online community meetings. Details can be found on our [Carvel website](https://carvel.dev/community/).

You can chat with us on Kubernetes Slack in the #carvel channel and follow us on Twitter at @carvel_dev.

Check out which organizations are using and contributing to Carvel: [Adopter's list](https://github.com/carvel-dev/carvel/blob/master/ADOPTERS.md)
