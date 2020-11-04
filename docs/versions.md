## Versions

Available in v0.12.0+.

Version selection is available for:

- git source type for selection of `ref` based on Git tags

---
### Semver

Vendir relies on [github.com/blang/semver/v4 package](https://github.com/blang/semver) for parsing "semver" versions.

For valid semver syntax refer to <https://semver.org/#backusnaur-form-grammar-for-valid-semver-versions>. (Vendir will ignore commonly-used `v` prefix during parsing)

For constraints syntax refer to [blang/semver's Ranges section](https://github.com/blang/semver#ranges).

By default prerelease versions are not included in selection. See examples for details.

#### Examples

Any version greater than 0.4.0 _without_ prereleases.

```
semver:
  constraints: ">0.4.0"
```

Any version greater than 0.4.0 _with_ all prereleases.

```
semver:
  constraints: ">0.4.0"
  prereleases: {}
```

Any version greater than 0.4.0 _with_ only beta or rc prereleases.

```
semver:
  constraints: ">0.4.0"
  prereleases:
    identifiers: [beta, rc]
```

#### sort-semver command

`vendir tools sort-semver` command is included to showcase how vendir parses versions.

- `--version` (`-v`) specifies one or more versions
- `--constraint` (`-c`) specifies zero or more constraints
- `--prerelease` specifies to include prereleases
- `--prerelease-identifier` specifies zero or more identifiers to match prereleases

```
$ vendir tools sort-semver -v "v0.0.1 v0.1.0 v0.2.0-pre.20 v0.2.0+build.1 v0.2.1 v0.2.0 v0.3.0"
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

Note that by default prerelease versions are not included. Use configuration or flag to include them.

```
$ vendir tools sort-semver -v "v0.0.1 v0.1.0 v0.2.0-pre.20 v0.2.0+build.1 v0.2.0 v0.3.0" -c ">=0.1.0"
Versions

Version
v0.1.0
v0.2.0+build.1
v0.2.0
v0.3.0

Highest version: v0.3.0

Succeeded
```
