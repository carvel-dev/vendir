To vendir with default `vendir.yml` contents:

```
$ vendir sync
```

While iterating on code you may want to add/remove/update pulled in content, hence, you can use `ytt` to build temporary copy and pass it to vendir.

```
$ vendir sync --file <(ytt -f vendir.yml -f local-override.yml)
```
