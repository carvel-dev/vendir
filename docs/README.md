## Docs

`vendir` allows to declaratively state what should be in a directory. It was designed to easily manage libraries for [ytt](https://get-ytt.io); however, it is a generic tool and does not care how files within managed directories are used.

`vendir sync` command looks for `vendir.yml` file for its configuration. `vendir.yml` specifies source of files for each managed directory. Currently there are only two file source types: `git` and `manual`.

See example within [examples/git-and-manual](../examples/git-and-manual) directory.

- [`vendir.yml` spec](vendir-spec.md)
- [`vendir.lock.yml` spec](vendir-lock-spec.md)
