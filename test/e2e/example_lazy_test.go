// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	ctlconf "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/config"
	"os"
	"testing"
)

func TestExampleLazy(t *testing.T) {
	env := BuildEnv(t)
	vendir := Vendir{t, env.BinaryPath, Logger{}}

	osEnv := []string{"VENDIR_HELM_BINARY=" + env.Helm2Binary}

	dir := "examples/lazy"
	path := "../../" + dir

	_, err := vendir.RunWithOpts([]string{"sync", "-f=vendir-lazy.yml"}, RunOpts{Dir: path, Env: osEnv})
	if err != nil {
		t.Fatalf("Expected no err")
	}

	lockConf, err := ctlconf.NewLockConfigFromFile(path + "/vendir.lock.yml")
	if err != nil {
		t.Fatalf("Expected no err: %s", err)
	}

	// find content digest
	if len(lockConf.Directories[0].Contents[0].ConfigDigest) == 0 {
		t.Fatalf("Expected Config Digest in Lock File")
	}

	// remove some directory
	err = os.RemoveAll(path + "/vendor/dir")
	if err != nil {
		t.Fatalf("Expected no err")
	}

	// resync lazily
	_, err = vendir.RunWithOpts([]string{"sync", "-f=vendir-lazy.yml"}, RunOpts{Dir: path, Env: osEnv})
	if err != nil {
		t.Fatalf("Expected no err")
	}

	_, err = os.Stat(path + "/vendor/dir")
	if err == nil {
		t.Fatalf("Expected err")
	} else if !os.IsNotExist(err) {
		t.Fatalf("Expected IsNotExist err")
	}

	// resync with lazy override
	_, err = vendir.RunWithOpts([]string{"sync", "--lazy=false", "-f=vendir-lazy.yml"}, RunOpts{Dir: path, Env: osEnv})
	if err != nil {
		t.Fatalf("Expected no err")
	}

	stat, err := os.Stat(path + "/vendor/dir")
	if err != nil {
		t.Fatalf("Expected no err")
	}
	if !stat.IsDir() {
		t.Fatalf("Expected Directory")
	}

	// content digest is kept during lazy sync override
	lockConf, err = ctlconf.NewLockConfigFromFile(path + "/vendir.lock.yml")
	if err != nil {
		t.Fatalf("Expected no err")
	}
	if len(lockConf.Directories[0].Contents[0].ConfigDigest) == 0 {
		t.Fatalf("Expected Config Digest in Lock File")
	}

	// if synced without lazy flag in vendir.yml, no config digest is kept in lock file
	_, err = vendir.RunWithOpts([]string{"sync", "-f=vendir-nonlazy.yml"}, RunOpts{Dir: path, Env: osEnv})
	if err != nil {
		t.Fatalf("Expected no err")
	}

	lockConf, err = ctlconf.NewLockConfigFromFile(path + "/vendir.lock.yml")
	if err != nil {
		t.Fatalf("Expected no err")
	}

	// find content digest
	if len(lockConf.Directories[0].Contents[0].ConfigDigest) != 0 {
		t.Fatalf("Expected No Config Digest in Lock File")
	}

}
