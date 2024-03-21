// Copyright 2024 The Carvel Authors.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"io"
	"log"
	"math/rand"
	"os"
	"time"

	"carvel.dev/vendir/pkg/vendir/cmd"
	uierrs "github.com/cppforlife/go-cli-ui/errors"
	"github.com/cppforlife/go-cli-ui/ui"
)

func main() {
	rand.New(rand.NewSource(time.Now().UTC().UnixNano()))

	log.SetOutput(io.Discard)

	// TODO logs
	// TODO log flags used

	confUI := ui.NewConfUI(ui.NewNoopLogger())
	defer confUI.Flush()

	command := cmd.NewDefaultVendirCmd(confUI)

	err := command.Execute()
	if err != nil {
		confUI.ErrorLinef("vendir: Error: %v", uierrs.NewMultiLineError(err))
		os.Exit(1)
	}

	confUI.PrintLinef("Succeeded")
}
