## Development & Deploy

### Prerequisites

You will need the following tools installed locally:
- git
- hg
- helm2
- helm3
- imgpkg

### Testing

To run all the tests:
```bash
./hack/build.sh
./hack/test-all.sh
```

#### Unit Testing

```
./hack/test.sh
```

#### e2e Testing
To avoid github rate-limiting generate a PAT and set it to the `VENDIR_GITHUB_API_TOKEN` env var:

```bash
export VENDIR_GITHUB_API_TOKEN=<pat-token-here>
./hack/test-e2e.sh
```

### Continuous Integration/Jobs

vendir uses GitHub Actions for all continuous integration for the project.
You can find these CI processes under the [`.github/workflows`](../.github/workflows)
folder.

#### Pull Requests

On each pull request, the following CI processes run:
* [`test-gh`](../.github/workflows/test-gh.yml) - Runs unit tests, runs e2e tests.
* [`golangci-lint`](../.github/workflows/golangci-lint.yml) - Runs project linter. Configuration for linter is in [`.golangci.yml`](../.golangci.yml) file.

#### Daily Jobs

Each day, the following processes run:
* [`Trivy CVE Dependency Scanner`](../.github/workflows/trivy-scan.yml) - This job runs a [`trivy`](https://aquasecurity.github.io/trivy/) scan on
  vendir code base and latest release to identify CVEs.
* [`Mark issues stale and close stale issues`](../.github/workflows/stale-issues-action.yml) - This job marks any issues without a comment for 40
  days as a stale issue. If no comment is made in the issue, the issue will then be closed in the next 5 days.

#### Jobs Based on Events

The actions below are carried out when a certain event occurs:
* [`Remove label on close`](../.github/workflows/closed-issue.yml) - This job runs whenever an issue is closed. It removes the `carvel-triage`
  label from the closed issue to signal no further attention is needed on the issue.
* [`Closed issue comment labeling`](../.github/workflows/closed-issue-comment.yml) - This job runs whenever a comment is posted to a closed
  issue to signal maintainers should take a look.
* [`vendir release`](../.github/workflows/release.yml) - This job carries out the vendir release.
