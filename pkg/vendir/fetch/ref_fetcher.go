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

type SingleSecretRefFetcher struct {
	Secret *ctlconf.Secret
}

var _ RefFetcher = SingleSecretRefFetcher{}

func (f SingleSecretRefFetcher) GetSecret(name string) (ctlconf.Secret, error) {
	if f.Secret != nil && f.Secret.Metadata.Name == name {
		return *f.Secret, nil
	}
	return ctlconf.Secret{}, fmt.Errorf("Not found")
}

func (f SingleSecretRefFetcher) GetConfigMap(name string) (ctlconf.ConfigMap, error) {
	return ctlconf.ConfigMap{}, fmt.Errorf("Not found")
}
