// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/registry"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/v1"
	ctlconf "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/config"
	ctlfetch "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch"
	ctlcache "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch/cache"
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
	cache      ctlcache.Cache
}

func NewImgpkg(opts ImgpkgOpts, refFetcher ctlfetch.RefFetcher, c ctlcache.Cache) *Imgpkg {
	if opts.CmdRunFunc == nil {
		opts.CmdRunFunc = func(cmd *exec.Cmd) error { return cmd.Run() }
	}

	if opts.EnvironFunc == nil {
		opts.EnvironFunc = os.Environ
	}

	return &Imgpkg{opts, refFetcher, c}
}

// FetchImage Downloads the OCI Image to the provided destination
func (t *Imgpkg) FetchImage(imageRef, destination string) (string, error) {
	return t.fetch(imageRef, destination, false)
}

// FetchBundle Downloads the Bundle to the provided destination
func (t *Imgpkg) FetchBundle(imageRef, destination string) (string, error) {
	return t.fetch(imageRef, destination, true)
}

// FetchBundleRecursively Download the Bundle and all the nested Bundles to the provided destination
func (t *Imgpkg) FetchBundleRecursively(imageRef, destination string) (string, error) {
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return "", err
	}

	if _, hit := t.cache.Hit(ref.Identifier()); hit {
		return imageRef, t.cache.CopyFrom(ref.Identifier(), destination)
	}

	envVariables, err := t.authEnv()
	if err != nil {
		return "", err
	}

	status, err := v1.PullRecursive(imageRef, destination, v1.PullOpts{
		Logger:   &Logger{buf: bytes.NewBufferString("")},
		AsImage:  false,
		IsBundle: true,
	}, registry.Opts{
		VerifyCerts:           !t.opts.DangerousSkipTLSVerify,
		Insecure:              false,
		ResponseHeaderTimeout: 30 * time.Second,
		RetryCount:            5,
		EnvironFunc: func() []string {
			return envVariables
		},
	})

	if err != nil {
		return "", err
	}

	if status.Cacheable {
		err := t.cache.Save(ref.Identifier(), destination)
		if err != nil {
			return "", err
		}
	}

	return status.ImageRef, nil
}

func (t *Imgpkg) fetch(imageRef, destination string, isBundle bool) (string, error) {
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return "", err
	}

	if _, hit := t.cache.Hit(ref.Identifier()); hit {
		return imageRef, t.cache.CopyFrom(ref.Identifier(), destination)
	}

	envVariables, err := t.authEnv()
	if err != nil {
		return "", err
	}

	status, err := v1.Pull(imageRef, destination, v1.PullOpts{
		Logger:   &Logger{buf: bytes.NewBufferString("")},
		AsImage:  !isBundle,
		IsBundle: isBundle,
	}, registry.Opts{
		VerifyCerts:           !t.opts.DangerousSkipTLSVerify,
		Insecure:              false,
		ResponseHeaderTimeout: 30 * time.Second,
		RetryCount:            5,
		EnvironFunc: func() []string {
			return envVariables
		},
	})

	if err != nil {
		return "", err
	}

	if status.Cacheable {
		err := t.cache.Save(ref.Identifier(), destination)
		if err != nil {
			return "", err
		}
	}

	return status.ImageRef, nil
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

// Logger provided to the imgpkg API calls
type Logger struct {
	buf *bytes.Buffer
}

// Errorf Writes error messages to the buffer
func (l *Logger) Errorf(msg string, args ...interface{}) {
	l.buf.Write([]byte(fmt.Sprintf(msg, args...)))
}

// Warnf Writes warning messages to the buffer
func (l *Logger) Warnf(msg string, args ...interface{}) {
	l.buf.Write([]byte(fmt.Sprintf(msg, args...)))
}

// Logf Writes messages to the buffer
func (l *Logger) Logf(msg string, args ...interface{}) {
	l.buf.Write([]byte(fmt.Sprintf(msg, args...)))
}

// Debugf does nothing
func (l *Logger) Debugf(string, ...interface{}) {}

// Tracef does nothing
func (l *Logger) Tracef(string, ...interface{}) {}
