## Docs

`vendir` allows to declaratively state what should be in a directory. It was designed to easily manage libraries for [ytt](https://get-ytt.io); however, it is a generic tool and does not care how files within managed directories are used.

### Sync command

`vendir sync` command looks for `vendir.yml` file for its configuration. `vendir.yml` specifies source of files for each managed directory. Currently there are only two file source types: `git` and `manual`.

```
# Run to sync directory contents as specified by vendir.yml
$ vendir sync
```

Configuration specs:

- [`vendir.yml` spec](vendir-spec.md)
- [`vendir.lock.yml` spec](vendir-lock-spec.md)

Source specific details:

- [Github release](github-release.md)

Examples could be found in [examples/](../examples/) directory.

### Sync with local changes override

As of v0.7.0 you can use `--directory` flag to override contents of particular directories by pointing them to local directories. When this flag is specified other directories will not be synced (hence lock config is not going to be updated).

```
$ vendir sync --directory vendor/local-dir=local-dir-dev
```
