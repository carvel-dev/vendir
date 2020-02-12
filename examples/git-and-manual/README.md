To vendir with default `vendir.yml` contents:

```
$ vendir sync
```

While iterating on code you may want to add/remove/update pulled in content, hence, you can use `ytt` to build temporary copy and pass it to vendir.

```
$ vendir sync --file <(ytt -f vendir.yml -f local-override.yml)
```

Alternatively as of v0.5.0 you can use `--use-directory` flag to override contents of particular directories by pointing them to local directories.

```
$ vendir sync --use-directory vendor/local-dir=local-dir-dev
```
