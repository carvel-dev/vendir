## Versions

Available in v0.12.0+.

Vendir uses version selection in following places:

- git source type for selection of `ref` based on Git tags

---
### VersionSelection type

`VersionSelection` type may be used by other projects (such as kbld) for selection of versions in different contexts. All usage follows same spec:

```yaml
# interpret versions according to semantic version spec.
# see semver section below for further details (required)
semver:
  # list of semver constraints (optional)
  constraints: ">0.4.0"
  # by default prerelease versions are not included (optional)
  prereleases:
    # select prerelease versions that include given identifiers (optional)
    identifiers: [beta, rc]
```

---
### Semver

[github.com/blang/semver/v4 package](https://github.com/blang/semver) is used for parsing "semver" versions.

For valid semver syntax refer to <https://semver.org/#backusnaur-form-grammar-for-valid-semver-versions>. (Commonly-used `v` prefix will be ignored during parsing)

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
