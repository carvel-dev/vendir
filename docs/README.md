## Docs

`vendir` allows to declaratively state what should be in a directory. It was designed to easily manage libraries for [ytt](https://get-ytt.io); however, it is a generic tool and does not care how files within managed directories are used.

### Sync command

`vendir sync` command looks for `vendir.yml` file for its configuration. `vendir.yml` specifies source of files for each managed directory. There are four source types: `git`, `githubRelease`, `directory` and `manual`.

```
# Run to sync directory contents as specified by vendir.yml
$ vendir sync
```

Further documentation:

- [`vendir.yml` spec](vendir-spec.md)
- [`vendir.lock.yml` spec](vendir-lock-spec.md)
- [Github release details](github-release.md)

Examples could be found in [examples/](../examples/) directory.

### Sync with local changes override

As of v0.7.0 you can use `--directory` flag to override contents of particular directories by pointing them to local directories. When this flag is specified other directories will not be synced (hence lock config is not going to be updated).

```
$ vendir sync --directory vendor/local-dir=local-dir-dev
```

### Sync with locks

`vendir sync` writes `vendir.lock.yml` (next to `vendir.yml`) that contains resolved references:

- for `git`, resolved SHAs are recorded
- for `githubRelease`, permanent links are recorded
- for `directory`, nothing is returned
- for `manual`, nothing is returned

To use these resolved references on top of `vendir.yml`, use `vendir sync -l`.
