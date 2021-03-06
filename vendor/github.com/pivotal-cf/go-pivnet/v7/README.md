[![GoDoc](https://godoc.org/github.com/pivotal-cf/go-pivnet?status.svg)](https://godoc.org/github.com/pivotal-cf/go-pivnet)


# go-pivnet API library

A golang library for [Pivotal Network](https://network.tanzu.vmware.com).

**NOTE**
This library is intended to be used by pivnet-cli and pivnet-resource. If you are using this library on its own, be aware of possible breaking changes following new releases of pivnet-cli/resource.

See also: [pivnet-cli](https://github.com/pivotal-cf/pivnet-cli)
and [pivnet-resource](https://github.com/pivotal-cf/pivnet-resource).

## Usage

See [example](https://github.com/pivotal-cf/go-pivnet/blob/master/example/main.go)
for an executable example.

```go
import pivnet github.com/pivotal-cf/go-pivnet/v7

[...]

config := pivnet.ClientConfig{
  Host:      pivnet.DefaultHost,
  Token:     "token-from-pivnet",
  UserAgent: "user-agent",
}

stdoutLogger := log.New(os.Stdout, "", log.LstdFlags)
stderrLogger := log.New(os.Stderr, "", log.LstdFlags)

verbose := false
logger := logshim.NewLogShim(stdoutLogger, stderrLogger, verbose)

client := pivnet.NewClient(config, logger)

products, _ := client.Products.List()

fmt.Printf("products: %v", products)
```

### Running the tests

Install the ginkgo executable with:

```
go get -u github.com/onsi/ginkgo/ginkgo
```

The tests require a valid Pivotal Network API token and host.

Refer to the
[official docs](https://network.tanzu.vmware.com/docs/api#how-to-authenticate)
for more details on obtaining a Pivotal Network API token.

It is advised to run the acceptance tests against the Pivotal Network integration
environment endpoint i.e. `HOST='https://pivnet-integration.cfapps.io'`.

Run the tests with the following command:

```
API_TOKEN=my-token \
HOST='https://pivnet-integration.cfapps.io' \
./bin/test_all
```

**NOTE**
 - If dependencies are not correct, the tests may not run correctly. You will have to run `GO111MODULE=on go mod vendor` first in this case.

### Contributing

Please make all pull requests to the `develop` branch, and
[ensure the tests pass locally](https://github.com/pivotal-cf/go-pivnet#running-the-tests).

### Project management

The CI for this project can be found
[here](https://p-concourse.wings.cf-app.com/teams/system-team-pivnet-resource-pivnet-resource-657d)
and the scripts can be found in the
[pivnet-resource-ci repo](https://github.com/pivotal-cf/pivnet-resource-ci).

The roadmap is captured in [Pivotal Tracker](https://www.pivotaltracker.com/projects/1474244).
