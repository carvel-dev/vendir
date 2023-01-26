// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package image_test

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"testing"

	regname "github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/registry"
	regrandom "github.com/google/go-containerregistry/pkg/v1/random"
	regremote "github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/require"
	ctlregistry "github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/registry"
	"github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/registry/auth"
	ctlconf "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/config"
	ctlfetch "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch"
	ctlcache "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch/cache"
	ctlimg "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch/image"
)

var localRegistryAddress string

func TestMain(m *testing.M) {
	port, err := freeport.GetFreePort()
	if err != nil {
		panic(err.Error())
	}
	localRegistryAddress = fmt.Sprintf("localhost:%d", port)
	s := &http.Server{
		Addr:    localRegistryAddress,
		Handler: registry.New(registry.Logger(log.New(bytes.NewBuffer(nil), "", 0))),
	}

	go func() {
		err := s.ListenAndServe()
		if err != nil {
			panic(err.Error())
		}
	}()

	os.Exit(m.Run())
}

func TestImgpkgAuth(t *testing.T) {
	t.Run("with empty plain secret", func(t *testing.T) {
		opts := createRegistryOptions(t, ctlconf.Secret{
			Data: map[string][]byte{},
		})

		requireImgpkgEnv(t, nil, opts.EnvironFunc())
	})

	t.Run("with filled plain secret", func(t *testing.T) {
		opts := createRegistryOptions(t, ctlconf.Secret{
			Data: map[string][]byte{
				"username": []byte("username"),
				"password": []byte("password"),
			},
		})

		requireImgpkgEnv(t, []string{
			"IMGPKG_USERNAME=username",
			"IMGPKG_PASSWORD=password",
		}, opts.EnvironFunc())
	})

	t.Run("with plain secret associated with hostname", func(t *testing.T) {
		opts := createRegistryOptions(t, ctlconf.Secret{
			Data: map[string][]byte{
				"hostname": []byte("hostname"),
				"username": []byte("username"),
				"password": []byte("password"),
			},
		})

		requireImgpkgEnv(t, []string{
			"IMGPKG_REGISTRY_HOSTNAME_0=hostname",
			"IMGPKG_REGISTRY_USERNAME_0=username",
			"IMGPKG_REGISTRY_PASSWORD_0=password",
		}, opts.EnvironFunc())
	})

	t.Run("with empty dockerconfigjson secret", func(t *testing.T) {
		opts := createRegistryOptions(t, ctlconf.Secret{
			Type: "kubernetes.io/dockerconfigjson",
			Data: map[string][]byte{
				".dockerconfigjson": []byte("{}"),
			},
		})

		requireImgpkgEnv(t, nil, opts.EnvironFunc())
	})

	t.Run("with filled dockerconfigjson secret", func(t *testing.T) {
		opts := createRegistryOptions(t, ctlconf.Secret{
			Type: "kubernetes.io/dockerconfigjson",
			Data: map[string][]byte{
				".dockerconfigjson": []byte(`{"auths":{
					"hostname1":{"username":"username1", "password":"password1"},
					"hostname2":{"username":"username2", "password":"password2"},
					"hostname3":{}
				}}`),
			},
		})

		requireImgpkgEnv(t, []string{
			"IMGPKG_REGISTRY_HOSTNAME_0=hostname1",
			"IMGPKG_REGISTRY_USERNAME_0=username1",
			"IMGPKG_REGISTRY_PASSWORD_0=password1",
			"IMGPKG_REGISTRY_HOSTNAME_1=hostname2",
			"IMGPKG_REGISTRY_USERNAME_1=username2",
			"IMGPKG_REGISTRY_PASSWORD_1=password2",
			"IMGPKG_REGISTRY_HOSTNAME_2=hostname3",
			"IMGPKG_REGISTRY_USERNAME_2=",
			"IMGPKG_REGISTRY_PASSWORD_2=",
		}, opts.EnvironFunc())
	})

	t.Run("without a secret", func(t *testing.T) {
		cache, err := ctlcache.NewCache("", "1Mi")
		require.NoError(t, err)

		imgpkg := ctlimg.NewImgpkg(
			ctlimg.ImgpkgOpts{},
			ctlfetch.SingleSecretRefFetcher{},
			cache,
		)

		opts, err := imgpkg.RegistryOpts()
		require.NoError(t, err)

		requireImgpkgEnv(t, nil, opts.EnvironFunc())
	})

	t.Run("enable keychain auth with list of keychains", func(t *testing.T) {
		cache, err := ctlcache.NewCache("", "1Mi")
		require.NoError(t, err)

		imgpkg := ctlimg.NewImgpkg(
			ctlimg.ImgpkgOpts{
				EnvironFunc: func() []string {
					return []string{"IMGPKG_ACTIVE_KEYCHAINS=gcr,ecr"}
				},
			},
			ctlfetch.SingleSecretRefFetcher{},
			cache,
		)

		opts, err := imgpkg.RegistryOpts()
		require.NoError(t, err)
		require.Equal(t, []auth.IAASKeychain{"gcr", "ecr"}, opts.ActiveKeychains)
	})

	t.Run("enable keychain auth with single keychain", func(t *testing.T) {
		cache, err := ctlcache.NewCache("", "1Mi")
		require.NoError(t, err)

		imgpkg := ctlimg.NewImgpkg(
			ctlimg.ImgpkgOpts{
				EnvironFunc: func() []string {
					return []string{"IMGPKG_ACTIVE_KEYCHAINS=single"}
				},
			},
			ctlfetch.SingleSecretRefFetcher{},
			cache,
		)

		opts, err := imgpkg.RegistryOpts()
		require.NoError(t, err)
		require.Equal(t, []auth.IAASKeychain{"single"}, opts.ActiveKeychains)
	})

	t.Run("no keychain enable when environment variable not set", func(t *testing.T) {
		cache, err := ctlcache.NewCache("", "1Mi")
		require.NoError(t, err)

		imgpkg := ctlimg.NewImgpkg(
			ctlimg.ImgpkgOpts{},
			ctlfetch.SingleSecretRefFetcher{},
			cache,
		)

		opts, err := imgpkg.RegistryOpts()
		require.NoError(t, err)
		require.Nil(t, opts.ActiveKeychains)
	})
}

func TestImgpkgCache(t *testing.T) {
	b, err := regrandom.Image(500, 5)
	require.NoError(t, err)
	d, err := b.Digest()
	require.NoError(t, err)
	ref, err := regname.ParseReference(fmt.Sprintf("%s/img1:test-img", localRegistryAddress))
	require.NoError(t, err)
	err = regremote.Write(ref, b)
	require.NoError(t, err)

	t.Run("uses cache when fetching a cacheable image", func(t *testing.T) {
		localCache := &fakeCache{cache: map[string]map[string]string{}}
		imgpkg := ctlimg.NewImgpkg(
			ctlimg.ImgpkgOpts{EnvironFunc: func() []string { return []string{} }},
			nil,
			localCache,
		)

		temp, err := os.MkdirTemp("", "vendir-fetch-image")
		require.NoError(t, err)
		defer os.RemoveAll(temp)
		digest := ref.Context().Digest(d.String())
		oRef, err := imgpkg.FetchImage(digest.String(), temp)
		require.NoError(t, err)
		fmt.Println(oRef)
		require.Equal(t, 1, localCache.numCallHit, "Called Hit 1 time")
		require.Equal(t, 1, localCache.numCallSave, "Called Save 1 time")
		require.Equal(t, 0, localCache.numCallCopyFrom, "Called CopyFrom 0 time")

		oRef1, err := imgpkg.FetchImage(digest.String(), temp)
		require.NoError(t, err)
		fmt.Println(oRef1)
		require.Equal(t, 2, localCache.numCallHit, "Called Hit 2 time")
		require.Equal(t, 1, localCache.numCallSave, "Called Save 1 time")
		require.Equal(t, 1, localCache.numCallCopyFrom, "Called CopyFrom 1 time")
	})

	t.Run("does not use cache when fetching is a Not cacheable image", func(t *testing.T) {
		localCache := &fakeCache{cache: map[string]map[string]string{}}
		imgpkg := ctlimg.NewImgpkg(
			ctlimg.ImgpkgOpts{EnvironFunc: func() []string { return []string{} }},
			nil,
			localCache,
		)

		temp, err := os.MkdirTemp("", "vendir-fetch-image")
		require.NoError(t, err)
		defer os.RemoveAll(temp)
		oRef, err := imgpkg.FetchImage(ref.String(), temp)
		require.NoError(t, err)
		fmt.Println(oRef)
		require.Equal(t, 1, localCache.numCallHit, "Called Hit 1 time")
		require.Equal(t, 0, localCache.numCallSave, "Called Save 1 time")
		require.Equal(t, 0, localCache.numCallCopyFrom, "Called CopyFrom 0 time")

		oRef1, err := imgpkg.FetchImage(ref.String(), temp)
		require.NoError(t, err)
		fmt.Println(oRef1)
		require.Equal(t, 2, localCache.numCallHit, "Called Hit 2 time")
		require.Equal(t, 0, localCache.numCallSave, "Called Save 1 time")
		require.Equal(t, 0, localCache.numCallCopyFrom, "Called CopyFrom 1 time")
	})
}

func createRegistryOptions(t *testing.T, secret ctlconf.Secret) ctlregistry.Opts {
	secret.Metadata = ctlconf.GenericMetadata{Name: "secret"}

	cache, err := ctlcache.NewCache("", "10Mi")
	require.NoError(t, err)

	imgpkg := ctlimg.NewImgpkg(
		ctlimg.ImgpkgOpts{
			SecretRef: &ctlconf.DirectoryContentsLocalRef{Name: "secret"},
		},
		ctlfetch.SingleSecretRefFetcher{Secret: &secret},
		cache,
	)

	opts, err := imgpkg.RegistryOpts()
	require.NoError(t, err)

	return opts
}

func requireImgpkgEnv(t *testing.T, expectedEnv, actualEnv []string) {
	var filteredActualEnv []string
	for _, kv := range actualEnv {
		if strings.HasPrefix(kv, "IMGPKG_") {
			filteredActualEnv = append(filteredActualEnv, kv)
		}
	}

	sort.Strings(filteredActualEnv)
	sort.Strings(expectedEnv)

	require.Equal(t, expectedEnv, filteredActualEnv)
}

type fakeCache struct {
	cache           map[string]map[string]string
	numCallHit      int
	numCallSave     int
	numCallCopyFrom int
}

func (d *fakeCache) Has(artifactType string, id string) (string, bool) {
	d.numCallHit++
	path, hit := d.cache[artifactType][id]
	return path, hit
}

func (d *fakeCache) Save(artifactType string, id string, src string) error {
	d.numCallSave++
	if _, found := d.cache[artifactType]; !found {
		d.cache[artifactType] = map[string]string{}
	}
	d.cache[artifactType][id] = src
	return nil
}

func (d *fakeCache) CopyFrom(_, _, _ string) error {
	d.numCallCopyFrom++
	return nil
}
