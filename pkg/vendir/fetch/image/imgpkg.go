// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	ctlconf "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/config"
	ctlfetch "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch"
)

type Imgpkg struct {
	secretRef  *ctlconf.DirectoryContentsLocalRef
	refFetcher ctlfetch.RefFetcher
	cmdRunFunc func(*exec.Cmd) error
}

func NewImgpkg(secretRef *ctlconf.DirectoryContentsLocalRef,
	refFetcher ctlfetch.RefFetcher, cmdRunFunc func(*exec.Cmd) error) *Imgpkg {

	if cmdRunFunc == nil {
		cmdRunFunc = func(cmd *exec.Cmd) error { return cmd.Run() }
	}
	return &Imgpkg{secretRef, refFetcher, cmdRunFunc}
}

func (t *Imgpkg) Run(args []string) (string, error) {
	authEnv, err := t.authEnv()
	if err != nil {
		return "", err
	}

	var stdoutBs, stderrBs bytes.Buffer

	cmd := exec.Command("imgpkg", args...)
	cmd.Env = append(os.Environ(), authEnv...)
	cmd.Stdout = &stdoutBs
	cmd.Stderr = &stderrBs

	err = t.cmdRunFunc(cmd)
	if err != nil {
		return "", fmt.Errorf("Imgpkg: %s (stderr: %s)", err, stderrBs.String())
	}

	return stdoutBs.String(), nil
}

func (t *Imgpkg) authEnv() ([]string, error) {
	var authEnv []string

	if t.secretRef != nil {
		secret, err := t.refFetcher.GetSecret(t.secretRef.Name)
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

	if len(authEnv) == 0 {
		authEnv = []string{"IMGPKG_ANON=true"}
	}

	return authEnv, nil
}
