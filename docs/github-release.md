## Github Release

vendir supports downloading software stored as a Github release. See [`vendir.yml` spec](vendir-spec.md) for how to configure.

### Github API Rate Limiting

If your public IP address is shared by multiple machines (e.g. workstations in an office), you may run into [Github rate limiting errors](https://developer.github.com/v3/#rate-limiting). vendir as of v0.8.0 supports providing "Personal access token" to increase Github API rate limits. You can specify it via an environment variable:

```bash
$ export VENDIR_GITHUB_API_TOKEN=8c0a3...
$ vendir sync
```

To obtain personal access token go to [Github.com: Settings / Developer Settings / Personal access tokens](https://github.com/settings/tokens). During token creation, you will be prompted for selection of scopes, and in most cases there is no need to select any scopes because this token only used to identify API usage. For organizations that enable SSO, you will need to "Enable SSO" for created token.
