// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package helmchart

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	ctlconf "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/config"
	ctlfetch "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch"
)

/*

apiVersion: vendir.k14s.io/v1alpha1
kind: Config
directories:
- path: config
  contents:
  - path: .
    helmChart:
      name: grafana
      version: 5.2.10
      repository:
        url: oci://registry.corp.com/projects/charts

...will result in...

$ export HELM_EXPERIMENTAL_OCI=1
$ helm registry login registry.corp.com/projects/charts
$ helm chart pull     registry.corp.com/projects/charts/grafana:5.2.10
$ helm chart export   registry.corp.com/projects/charts/grafana:5.2.10

Handy command to run registry with certs:

$ docker run -d -p 5000:5000 --name registry -v /Users/dk/workspace/cert:/certs \
	-e "REGISTRY_AUTH=htpasswd" -e "REGISTRY_AUTH_HTPASSWD_REALM=Registry Realm" \
	-e REGISTRY_AUTH_HTPASSWD_PATH=/auth/htpasswd -e REGISTRY_HTTP_TLS_CERTIFICATE=/certs/domain.crt \
	-e REGISTRY_HTTP_TLS_KEY=/certs/domain.key registry:2
$ docker stop registry && docker rm registry

*/

type OCISource struct {
	opts       ctlconf.DirectoryContentsHelmChart
	helmBinary string
	refFetcher ctlfetch.RefFetcher
}

func NewOCISource(opts ctlconf.DirectoryContentsHelmChart,
	helmBinary string, refFetcher ctlfetch.RefFetcher) *OCISource {

	return &OCISource{opts, helmBinary, refFetcher}
}

func (t *OCISource) Fetch(dstPath string, tempArea ctlfetch.TempArea) error {
	if len(t.opts.Name) == 0 {
		return fmt.Errorf("Expected non-empty name")
	}
	if len(t.opts.Version) == 0 {
		return fmt.Errorf("Expected non-empty version")
	}
	if t.opts.Repository == nil || len(t.opts.Repository.URL) == 0 {
		return fmt.Errorf("Expected non-empty repository URL")
	}

	helmHomeDir, err := tempArea.NewTempDir("helm-home")
	if err != nil {
		return err
	}

	defer os.RemoveAll(helmHomeDir)

	repo := strings.TrimPrefix(t.opts.Repository.URL, "oci://")

	// TODO authenticate against multiple repos since dependencies might be else where?
	err = t.login(repo, helmHomeDir)
	if err != nil {
		return err
	}

	ref := fmt.Sprintf("%s/%s:%s", repo, t.opts.Name, t.opts.Version)

	err = t.pull(ref, helmHomeDir)
	if err != nil {
		return err
	}

	err = t.export(ref, dstPath, helmHomeDir)
	if err != nil {
		return err
	}

	// TODO might need to run "helm dependency update"

	return nil
}

func (t *OCISource) login(repo, helmHomeDir string) error {
	authArgs, cmdStdin, err := t.addAuthArgs([]string{})
	if err != nil {
		return fmt.Errorf("Adding helm auth info: %s", err)
	}

	if len(authArgs) == 0 {
		// Skipping authentication
		return nil
	}

	args := append([]string{"registry", "login", repo}, authArgs...)

	var stdoutBs, stderrBs bytes.Buffer

	cmd := exec.Command(t.helmBinary, args...)
	cmd.Env = []string{"HOME=" + helmHomeDir, "HELM_EXPERIMENTAL_OCI=1"}
	cmd.Stdin = cmdStdin
	cmd.Stdout = &stdoutBs
	cmd.Stderr = &stderrBs

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("Helm registry login: %s (stderr: %s)", err, stderrBs.String())
	}

	return nil
}

func (t *OCISource) pull(ref, helmHomeDir string) error {
	args := []string{"chart", "pull", ref}

	var stdoutBs, stderrBs bytes.Buffer

	cmd := exec.Command(t.helmBinary, args...)
	cmd.Env = []string{"HOME=" + helmHomeDir, "HELM_EXPERIMENTAL_OCI=1"}
	cmd.Stdout = &stdoutBs
	cmd.Stderr = &stderrBs

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("Helm chart pull: %s (stderr: %s)", err, stderrBs.String())
	}

	return nil
}

func (t *OCISource) export(ref, dstPath, helmHomeDir string) error {
	args := []string{"chart", "export", ref, "--destination", dstPath}

	var stdoutBs, stderrBs bytes.Buffer

	cmd := exec.Command(t.helmBinary, args...)
	cmd.Env = []string{"HOME=" + helmHomeDir, "HELM_EXPERIMENTAL_OCI=1"}
	cmd.Stdout = &stdoutBs
	cmd.Stderr = &stderrBs

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("Helm chart export: %s (stderr: %s)", err, stderrBs.String())
	}

	return nil
}

func (t *OCISource) addAuthArgs(args []string) ([]string, io.Reader, error) {
	var authArgs []string
	var passwordStdin io.Reader

	if t.opts.Repository != nil && t.opts.Repository.SecretRef != nil {
		secret, err := t.refFetcher.GetSecret(t.opts.Repository.SecretRef.Name)
		if err != nil {
			return nil, nil, err
		}

		secrets, err := secret.ToRegistryAuthSecrets()
		if err != nil {
			return nil, nil, err
		}

		for _, secret := range secrets {
			for name, val := range secret.Data {
				switch name {
				case ctlconf.SecretRegistryHostnameKey:
					// do nothing for now
					// TODO match secret by hostname?
				case ctlconf.SecretK8sCorev1BasicAuthUsernameKey:
					authArgs = append(authArgs, []string{"--username", string(val)}...)
				case ctlconf.SecretK8sCorev1BasicAuthPasswordKey:
					authArgs = append(authArgs, []string{"--password-stdin"}...)
					passwordStdin = strings.NewReader(string(val))
				default:
					return nil, nil, fmt.Errorf("Unknown secret field '%s' in secret '%s'", name, secret.Metadata.Name)
				}
			}
			// Only single set of credentials is supported
			break
		}
	}

	return append(args, authArgs...), passwordStdin, nil
}
