## Versions

Available in v0.10.0+.

Vendir relies on [hashicorp/go-version](https://github.com/hashicorp/go-version) for parsing versions (based on SemVer specification). 

`vendir tools sort-versions` command is included to showcase this behaviour.

- `--version` (`-v`) specifies one or more versions
- `--constraint` (`-c`) specified zero or more constraints

Examples:

```
$ vendir tools sort-versions -v "v0.0.1 v0.1.0 v0.2.0-pre.20 v0.2.0+build.1 v0.2.1 v0.2.0 v0.3.0"
Versions

Version
v0.0.1
v0.1.0
v0.2.0-pre.20
v0.2.0+build.1
v0.2.0
v0.2.1
v0.3.0

Highest version: v0.3.0

Succeeded
```

Note that constraints without prerelease segment do not match versions that include prerelease segment. For example `>=0.1.0` is not going to match `v0.2.0-pre.20`.

```
$ vendir tools sort-versions -v "v0.0.1 v0.1.0 v0.2.0-pre.20 v0.2.0+build.1 v0.2.0 v0.3.0" -c ">=0.1.0"
Versions

Version
v0.1.0
v0.2.0+build.1
v0.2.0
v0.3.0

Highest version: v0.3.0

Succeeded
```
