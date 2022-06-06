// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	ctlconf "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/config"
	ctlfetch "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch"
)

type ImgpkgOpts struct {
	SecretRef              *ctlconf.DirectoryContentsLocalRef
	DangerousSkipTLSVerify bool

	CmdRunFunc  func(*exec.Cmd) error
	EnvironFunc func() []string
}

type Imgpkg struct {
	opts       ImgpkgOpts
	refFetcher ctlfetch.RefFetcher
}

func NewImgpkg(opts ImgpkgOpts, refFetcher ctlfetch.RefFetcher) *Imgpkg {
	if opts.CmdRunFunc == nil {
		opts.CmdRunFunc = func(cmd *exec.Cmd) error { return cmd.Run() }
	}

	if opts.EnvironFunc == nil {
		opts.EnvironFunc = os.Environ
	}

	return &Imgpkg{opts, refFetcher}
}

func (t *Imgpkg) Run(args []string) (string, error) {
	args = append([]string{}, args...) // copy
	args = t.addDangerousArgs(args)

	authEnv, err := t.authEnv()
	if err != nil {
		return "", err
	}

	var stdoutBs, stderrBs bytes.Buffer

	cmd := exec.Command("imgpkg", args...)
	cmd.Env = append(t.opts.EnvironFunc(), authEnv...)
	cmd.Stdout = &stdoutBs
	cmd.Stderr = &stderrBs

	err = t.opts.CmdRunFunc(cmd)
	if err != nil {
		return "", fmt.Errorf("Imgpkg: %s (stderr: %s)", err, stderrBs.String())
	}

	return stdoutBs.String(), nil
}

func (t *Imgpkg) Tags(repo string) ([]string, error) {
	out, err := t.Run([]string{"tag", "list", "-i", repo, "--column=name", "--digests=false"})
	if err != nil {
		return nil, fmt.Errorf("Fetching image tags: %s", err)
	}

	out = strings.TrimSpace(out)
	// not sure why there are tabs; remove in older versions
	out = strings.Replace(out, "\t", "", -1)

	return strings.Split(out, "\n"), nil
}

func (t *Imgpkg) authEnv() ([]string, error) {
	var authEnv []string

	if t.opts.SecretRef != nil {
		secret, err := t.refFetcher.GetSecret(t.opts.SecretRef.Name)
		if err != nil {
			return nil, err
		}

		secrets, err := secret.ToRegistryAuthSecrets()
		if err != nil {
			return nil, err
		}

		for i, secret := range secrets {
			// In case there is no registry hostname specified, set general fallback creds
			if _, found := secret.Data[ctlconf.SecretRegistryHostnameKey]; found {
				for name, val := range secret.Data {
					switch name {
					case ctlconf.SecretRegistryHostnameKey:
						authEnv = append(authEnv, fmt.Sprintf("IMGPKG_REGISTRY_HOSTNAME_%d=%s", i, val))
					case ctlconf.SecretK8sCorev1BasicAuthUsernameKey:
						authEnv = append(authEnv, fmt.Sprintf("IMGPKG_REGISTRY_USERNAME_%d=%s", i, val))
					case ctlconf.SecretK8sCorev1BasicAuthPasswordKey:
						authEnv = append(authEnv, fmt.Sprintf("IMGPKG_REGISTRY_PASSWORD_%d=%s", i, val))
					case ctlconf.SecretRegistryIdentityToken:
						authEnv = append(authEnv, fmt.Sprintf("IMGPKG_REGISTRY_IDENTITY_TOKEN_%d=%s", i, val))
					case ctlconf.SecretRegistryBearerToken:
						authEnv = append(authEnv, fmt.Sprintf("IMGPKG_REGISTRY_REGISTRY_TOKEN_%d=%s", i, val))
					default:
						return nil, fmt.Errorf("Unknown secret field '%s' in secret '%s'", name, secret.Metadata.Name)
					}
				}
			} else {
				for name, val := range secret.Data {
					switch name {
					case ctlconf.SecretK8sCorev1BasicAuthUsernameKey:
						authEnv = append(authEnv, fmt.Sprintf("IMGPKG_USERNAME=%s", val))
					case ctlconf.SecretK8sCorev1BasicAuthPasswordKey:
						authEnv = append(authEnv, fmt.Sprintf("IMGPKG_PASSWORD=%s", val))
					case ctlconf.SecretRegistryBearerToken:
						authEnv = append(authEnv, fmt.Sprintf("IMGPKG_TOKEN=%s", val))
					default:
						return nil, fmt.Errorf("Unknown secret field '%s' in secret '%s'", name, secret.Metadata.Name)
					}
				}
			}
		}
	}

	return authEnv, nil
}

func (t *Imgpkg) addDangerousArgs(args []string) []string {
	if t.opts.DangerousSkipTLSVerify {
		args = append(args, "--registry-verify-certs=false")
	}
	return args
}
