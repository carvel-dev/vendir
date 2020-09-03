# k8s-simple-app-example

Example repo shows how to use tools from k14s org: [ytt](https://get-ytt.io), [kbld](https://get-kbld.io), [kapp](https://get-kapp.io) and [kwt](https://github.com/k14s/kapp) to work with a simple Go app on Kubernetes.

Associated blog post: [Introducing k14s (Kubernetes Tools): Simple and Composable Tools for Application Deployment](https://content.pivotal.io/blog/introducing-k14s-kubernetes-tools-simple-and-composable-tools-for-application-deployment).

## Install k14s Tools

Head over to [k14s.io](https://k14s.io/) for installation instructions.

## Deploying Application

Each top level step has an associated `config-step-*` directory. Refer to [Directory Layout](#directory-layout) for details about files.

### Step 1: Deploying application

Introduces [kapp](https://get-kapp.io) for deploying k8s resources.

```bash
kapp deploy -a simple-app -f config-step-1-minimal/
kapp inspect -a simple-app --tree
kapp logs -f -a simple-app
```

### Step 1a: Viewing application

Once deployed successfully, you can access frontend service at `127.0.0.1:8080` in your browser via `kubectl port-forward` command:

```bash
kubectl port-forward svc/simple-app 8080:80
```

You will have to restart port forward command after making any changes as pods are recreated. Alternatively consider using [k14s' kwt tool](https://github.com/k14s/kwt) which exposes cluser IP subnets and cluster DNS to your machine and does not require any restarts:

```bash
sudo -E kwt net start
```

and open [`http://simple-app.default.svc.cluster.local/`](http://simple-app.default.svc.cluster.local/).

### Step 1b: Modifying application configuration

Modify `HELLO_MSG` environment value from `stranger` to something else in `config-step-1-minimal/config.yml`, and run:

```bash
kapp deploy -a simple-app -f config-step-1-minimal/ --diff-changes
```

### Step 2: Configuration templating

Introduces [ytt](https://get-ytt.io) templating for more flexible configuration.

```bash
ytt -f config-step-2-template/ | kapp deploy -a simple-app -f- --diff-changes -y
```

ytt provides a way to configure data values from command line as well:

```bash
ytt -f config-step-2-template/ -v hello_msg=another-stranger | kapp deploy -a simple-app -f- --diff-changes -y
```

New message should be returned from the app in the browser.

### Step 2a: Configuration patching

Introduces [ytt overlays](https://github.com/k14s/ytt/blob/master/docs/lang-ref-ytt-overlay.md) to patch configuration without modifying original `config.yml`.

```bash
ytt -f config-step-2-template/ -f config-step-2a-overlays/custom-scale.yml | kapp deploy -a simple-app -f- --diff-changes -y
```

### Step 2b: Customizing configuration data values per environment

Requires ytt v0.13.0+.

Introduces [use of multiple data values](https://github.com/k14s/ytt/blob/master/docs/ytt-data-values.md) to show layering of configuration for different environment without modifying default `values.yml`.

```bash
ytt -f config-step-2-template/ -f config-step-2b-multiple-data-values/ | kapp deploy -a simple-app -f- --diff-changes -y
```

### Step 3: Building container images locally

Introduces [kbld](https://get-kbld.io) functionality for building images from source code. This step requires Minikube. If Minikube is not available, skip to the next step.

```bash
eval $(minikube docker-env)
ytt -f config-step-3-build-local/ | kbld -f- | kapp deploy -a simple-app -f- --diff-changes -y
```

Note that rerunning above command again should be a noop, given that nothing has changed.

### Step 3a: Modifying application source code

Uncomment `fmt.Fprintf(w, "<p>local change</p>")` line in `app.go`, and re-run above command:

```bash
ytt -f config-step-3-build-local/ | kbld -f- | kapp deploy -a simple-app -f- --diff-changes -y
```

Observe that new container was built, and deployed. This change should be returned from the app in the browser.

### Step 4: Building and pushing container images to registry

Introduces [kbld](https://get-kbld.io) functionality to push to remote registries. This step can works against Minikube or remote cluster.

```bash
docker login -u dkalinin -p ...
ytt -f config-step-4-build-and-push/ -v push_images=true -v push_images_repo=docker.io/dkalinin/k8s-simple-app | kbld -f- | kapp deploy -a simple-app -f- --diff-changes -y
```

### Step 5: Clean up cluster resources

```bash
kapp delete -a simple-app
```

There is currently no functionality in kbld to remove pushed images from registry.

## Directory Layout

- [`app.go`](app.go): simple Go HTTP server
- [`Dockerfile`](Dockerfile): Dockerfile to build Go app
- `config-step-1-minimal/`
  - [`config.yml`](config-step-1-minimal/config.yml): basic k8s Service and Deployment configuration for the app
- `config-step-2-template/`
  - [`config.yml`](config-step-2-template/config.yml): slightly modified configuration to use `ytt` features, such as data module and functions
  - [`values.yml`](config-step-2-template/values.yml): defines extracted data values used in `config.yml`
- `config-step-2a-overlays/`
  - [`custom-scale.yml`](config-step-2a-overlays/custom-scale.yml): ytt overlay to set number of deployment replicas to 3
- `config-step-3-build-local/`
  - [`build.yml`](config-step-3-build-local/build.yml): tells `kbld` about how to build container image from source (app.go + Dockerfile)
  - [`config.yml`](config-step-3-build-local/config.yml): _same as prev step_
  - [`values.yml`](config-step-3-build-local/values.yml): _same as prev step_
- `config-step-4-build-and-push/`
  - [`build.yml`](config-step-4-build-and-push/build.yml): _same as prev step_
  - [`push.yml`](config-step-4-build-and-push/push.yml): tells `kbld` about how to push container image to remote registry
  - [`config.yml`](config-step-4-build-and-push/config.yml): _same as prev step_
  - [`values.yml`](config-step-4-build-and-push/values.yml): defines shared configuration, including configuration for pushing container images
