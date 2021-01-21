// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package fetch

import (
	"fmt"

	ctlconf "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/config"
)

type RefFetcher interface {
	GetSecret(string) (ctlconf.Secret, error)
	GetConfigMap(string) (ctlconf.ConfigMap, error)
}

type NoopRefFetcher struct{}

var _ RefFetcher = NoopRefFetcher{}

func (f NoopRefFetcher) GetSecret(name string) (ctlconf.Secret, error) {
	return ctlconf.Secret{}, fmt.Errorf("Not found")
}

func (f NoopRefFetcher) GetConfigMap(name string) (ctlconf.ConfigMap, error) {
	return ctlconf.ConfigMap{}, fmt.Errorf("Not found")
}
