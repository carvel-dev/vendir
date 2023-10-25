// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"os"
	"testing"

	ctlconf "carvel.dev/vendir/pkg/vendir/config"
	"github.com/stretchr/testify/require"
)

func TestExampleLazy(t *testing.T) {
	env := BuildEnv(t)
	vendir := Vendir{t, env.BinaryPath, Logger{}}

	osEnv := []string{"VENDIR_HELM_BINARY=" + env.Helm2Binary}

	dir := "examples/lazy"
	path := "../../" + dir

	// run lazy sync
	_, err := vendir.RunWithOpts([]string{"sync", "-f=vendir-lazy.yml"}, RunOpts{Dir: path, Env: osEnv})
	require.NoError(t, err)

	// check that the lock file has config digest
	lockConf, err := ctlconf.NewLockConfigFromFile(path + "/vendir.lock.yml")
	require.NoError(t, err)
	require.NotEmpty(t, lockConf.Directories[0].Contents[0].ConfigDigest, "Expected Config Digest in Lock File")

	// remove some directory
	err = os.RemoveAll(path + "/vendor/dir")
	require.NoError(t, err)

	// resync lazily, should not sync. Removed dir has not been reinstated
	_, err = vendir.RunWithOpts([]string{"sync", "-f=vendir-lazy.yml"}, RunOpts{Dir: path, Env: osEnv})
	require.NoError(t, err)
	require.NoDirExists(t, path+"/vendor/dir")

	// resync with lazy override, should not affect config digest
	_, err = vendir.RunWithOpts([]string{"sync", "--lazy=false", "-f=vendir-lazy.yml"}, RunOpts{Dir: path, Env: osEnv})
	require.NoError(t, err)
	require.DirExists(t, path+"/vendor/dir")

	// content digest is kept during lazy sync override
	lockConf, err = ctlconf.NewLockConfigFromFile(path + "/vendir.lock.yml")
	require.NoError(t, err)
	require.NotEmpty(t, lockConf.Directories[0].Contents[0].ConfigDigest, "Expected Config Digest in Lock File")

	// if synced without lazy flag in vendir.yml, no config digest should be kept in lock file
	_, err = vendir.RunWithOpts([]string{"sync", "-f=vendir-nonlazy.yml"}, RunOpts{Dir: path, Env: osEnv})
	require.NoError(t, err)
	lockConf, err = ctlconf.NewLockConfigFromFile(path + "/vendir.lock.yml")
	require.NoError(t, err)
	require.Empty(t, lockConf.Directories[0].Contents[0].ConfigDigest, "Expected No Config Digest in Lock File")
}
