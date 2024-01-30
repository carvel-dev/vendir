// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package image

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"carvel.dev/imgpkg/pkg/imgpkg/registry"
	v1 "carvel.dev/imgpkg/pkg/imgpkg/v1"
	ctlconf "carvel.dev/vendir/pkg/vendir/config"
	ctlfetch "carvel.dev/vendir/pkg/vendir/fetch"
	ctlcache "carvel.dev/vendir/pkg/vendir/fetch/cache"
	"github.com/google/go-containerregistry/pkg/name"
)

const (
	ImgpkgBundleArtifactType = "imgpkgBundle"
	ImageArtifactType        = "image"
)

type ImgpkgOpts struct {
	SecretRef              *ctlconf.DirectoryContentsLocalRef
	DangerousSkipTLSVerify bool
	ResponseHeaderTimeout  int

	EnvironFunc func() []string
}

type Imgpkg struct {
	opts       ImgpkgOpts
	refFetcher ctlfetch.RefFetcher
	cache      ctlcache.Cache
}

func NewImgpkg(opts ImgpkgOpts, refFetcher ctlfetch.RefFetcher, c ctlcache.Cache) *Imgpkg {
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

	if _, hit := t.cache.Has(ImgpkgBundleArtifactType, ref.Identifier()); hit {
		return imageRef, t.cache.CopyFrom(ImgpkgBundleArtifactType, ref.Identifier(), destination)
	}

	opts, err := t.RegistryOpts()
	if err != nil {
		return "", err
	}

	status, err := v1.PullRecursive(imageRef, destination, v1.PullOpts{
		Logger:   &Logger{buf: bytes.NewBufferString("")},
		AsImage:  false,
		IsBundle: true,
	}, opts)

	if err != nil {
		return "", err
	}

	if status.Cacheable {
		err := t.cache.Save(ImgpkgBundleArtifactType, ref.Identifier(), destination)
		if err != nil {
			return "", err
		}
	}

	return status.ImageRef, nil
}

func (t *Imgpkg) fetch(imageRef, destination string, isBundle bool) (string, error) {
	artifactType := ImageArtifactType
	if isBundle {
		artifactType = ImgpkgBundleArtifactType
	}

	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return "", err
	}

	if _, hit := t.cache.Has(artifactType, ref.Identifier()); hit {
		return imageRef, t.cache.CopyFrom(artifactType, ref.Identifier(), destination)
	}

	opts, err := t.RegistryOpts()
	if err != nil {
		return "", err
	}

	status, err := v1.Pull(imageRef, destination, v1.PullOpts{
		Logger:   &Logger{buf: bytes.NewBufferString("")},
		AsImage:  !isBundle,
		IsBundle: isBundle,
	}, opts)

	if err != nil {
		return "", err
	}

	if status.Cacheable {
		err := t.cache.Save(artifactType, ref.Identifier(), destination)
		if err != nil {
			return "", err
		}
	}

	return status.ImageRef, nil
}

func (t *Imgpkg) Tags(repo string) ([]string, error) {
	opts, err := t.RegistryOpts()
	if err != nil {
		return nil, err
	}

	tagsInfo, err := v1.TagList(repo, false, opts)
	if err != nil {
		return nil, err
	}

	var tags []string
	for _, tag := range tagsInfo.Tags {
		tags = append(tags, tag.Tag)
	}

	return tags, nil
}

func (t *Imgpkg) RegistryOpts() (registry.Opts, error) {
	envVariables, err := t.authEnv()
	if err != nil {
		return registry.Opts{}, err
	}

	opts := registry.Opts{
		VerifyCerts:           !t.opts.DangerousSkipTLSVerify,
		Insecure:              false,
		ResponseHeaderTimeout: time.Duration(t.opts.ResponseHeaderTimeout|30) * time.Second,
		RetryCount:            5,
		EnvironFunc: func() []string {
			return append(envVariables, t.opts.EnvironFunc()...)
		},
	}
	envVars := map[string]string{}
	for _, envVar := range append(envVariables, t.opts.EnvironFunc()...) {
		envVarSplit := strings.SplitN(envVar, "=", 2)
		if len(envVarSplit) != 2 {
			return registry.Opts{}, fmt.Errorf("Value '%s' does not look like an environment variable", envVar)
		}
		envVars[envVarSplit[0]] = envVarSplit[1]
	}

	envLookup := func(key string) (string, bool) {
		value, found := envVars[key]
		return value, found
	}
	return v1.OptsFromEnv(opts, envLookup), nil
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

// Logger provided to the imgpkg API calls
// This logger does write to the buffer debug and trace message.
// If we want to provide such a mechanism we should provide a way to define what is the level of messages that
// we want to have present in the buffer
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
