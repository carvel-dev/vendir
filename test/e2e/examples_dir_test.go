// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"bytes"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-containerregistry/pkg/registry"
	"github.com/phayes/freeport"
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

	err = filepath.WalkDir("./assets/helmcharts/", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		_, _, err = execHelm3([]string{"push", path, fmt.Sprintf("oci://%s/%s", localRegistryAddress, "helmcharts")})
		return err
	})

	if err != nil {
		panic(err.Error())
	}

	os.Exit(m.Run())
}

func TestExamplesDir(t *testing.T) {
	env := BuildEnv(t)

	// Useful when commenting out examples
	_ = env.Helm3Binary

	tests := []example{
		{Name: "git"},
		{Name: "hg"},
		{Name: "http"},
		{Name: "image"},
		{Name: "imgpkgBundle"},
		{Name: "helm-chart", Env: []string{"VENDIR_HELM_BINARY=" + env.Helm2Binary}},
		{Name: "helm-chart", Env: []string{"VENDIR_HELM_BINARY=" + env.Helm3Binary}},
		{Name: "helm-chart-oci", Env: []string{"VENDIR_HELM_BINARY=" + env.Helm3Binary}, YttVendirYamlArgs: []string{"-v", fmt.Sprintf("registryURL=%s", localRegistryAddress)}},
		{Name: "github-release"},
		{Name: "entire-dir"},
		{Name: "inline"},
		{Name: "locked", OnlyLocked: true, Env: []string{"VENDIR_HELM_BINARY=" + env.Helm3Binary}},
		{Name: "new-root-path"},
		{Name: "versionselection"},
		{Name: "ignore", SkipRemove: true},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			test.Check(t)
		})
	}
}
