// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package image_test

import (
	"os/exec"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	ctlconf "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/config"
	ctlfetch "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch"
	ctlimg "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch/image"
)

func TestImgpkgAuth(t *testing.T) {
	t.Run("with empty plain secret", func(t *testing.T) {
		ranCmd := runImgpkgWithSecret(t, ctlconf.Secret{
			Data: map[string][]byte{},
		})

		requireImgpkgEnv(t, nil, ranCmd.Env)
	})

	t.Run("with filled plain secret", func(t *testing.T) {
		ranCmd := runImgpkgWithSecret(t, ctlconf.Secret{
			Data: map[string][]byte{
				"username": []byte("username"),
				"password": []byte("password"),
			},
		})

		requireImgpkgEnv(t, []string{
			"IMGPKG_USERNAME=username",
			"IMGPKG_PASSWORD=password",
		}, ranCmd.Env)
	})

	t.Run("with plain secret associated with hostname", func(t *testing.T) {
		ranCmd := runImgpkgWithSecret(t, ctlconf.Secret{
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
		}, ranCmd.Env)
	})

	t.Run("with empty dockerconfigjson secret", func(t *testing.T) {
		ranCmd := runImgpkgWithSecret(t, ctlconf.Secret{
			Type: "kubernetes.io/dockerconfigjson",
			Data: map[string][]byte{
				".dockerconfigjson": []byte("{}"),
			},
		})

		requireImgpkgEnv(t, nil, ranCmd.Env)
	})

	t.Run("with filled dockerconfigjson secret", func(t *testing.T) {
		ranCmd := runImgpkgWithSecret(t, ctlconf.Secret{
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
		}, ranCmd.Env)
	})

	t.Run("without a secret", func(t *testing.T) {
		var ranCmd *exec.Cmd

		imgpkg := ctlimg.NewImgpkg(
			ctlimg.ImgpkgOpts{
				CmdRunFunc:  func(cmd *exec.Cmd) error { ranCmd = cmd; return nil },
				EnvironFunc: func() []string { return []string{} },
			},
			ctlfetch.SingleSecretRefFetcher{},
		)

		_, err := imgpkg.Run([]string{})
		require.NoError(t, err)

		requireImgpkgEnv(t, nil, ranCmd.Env)
	})
}

func runImgpkgWithSecret(t *testing.T, secret ctlconf.Secret) *exec.Cmd {
	secret.Metadata = ctlconf.GenericMetadata{Name: "secret"}

	var ranCmd *exec.Cmd

	imgpkg := ctlimg.NewImgpkg(
		ctlimg.ImgpkgOpts{
			SecretRef:   &ctlconf.DirectoryContentsLocalRef{Name: "secret"},
			CmdRunFunc:  func(cmd *exec.Cmd) error { ranCmd = cmd; return nil },
			EnvironFunc: func() []string { return []string{} },
		},
		ctlfetch.SingleSecretRefFetcher{&secret},
	)

	_, err := imgpkg.Run([]string{})
	require.NoError(t, err)

	return ranCmd
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
