// Copyright 2020 VMware, Inc.
// SPDX-License-Identifier: Apache-2.0

package directory

import (
	"fmt"
	"reflect"

	ctlconf "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/config"
	ctlfetch "github.com/vmware-tanzu/carvel-vendir/pkg/vendir/fetch"
)

type NamedRefFetcher struct {
	secrets    []ctlconf.Secret
	configMaps []ctlconf.ConfigMap
}

var _ ctlfetch.RefFetcher = NamedRefFetcher{}

func NewNamedRefFetcher(secrets []ctlconf.Secret, configMaps []ctlconf.ConfigMap) NamedRefFetcher {
	return NamedRefFetcher{secrets, configMaps}
}

func (f NamedRefFetcher) GetSecret(name string) (ctlconf.Secret, error) {
	var found []ctlconf.Secret
	for _, secret := range f.secrets {
		if secret.Metadata.Name == name {
			if len(found) == 1 && reflect.DeepEqual(found[0].Data, secret.Data) {
				continue
			}
			found = append(found, secret)
		}
	}

	if len(found) == 0 {
		return ctlconf.Secret{}, fmt.Errorf(
			"Expected to find one secret '%s', but found none", name)
	}
	if len(found) > 1 {
		return ctlconf.Secret{}, fmt.Errorf(
			"Expected to find one secret '%s', but found multiple", name)
	}

	return found[0], nil
}

func (f NamedRefFetcher) GetConfigMap(name string) (ctlconf.ConfigMap, error) {
	var found []ctlconf.ConfigMap
	for _, configMap := range f.configMaps {
		if configMap.Metadata.Name == name {
			found = append(found, configMap)
		}
	}

	if len(found) == 0 {
		return ctlconf.ConfigMap{}, fmt.Errorf(
			"Expected to find one config map '%s', but found none", name)
	}
	if len(found) > 1 {
		return ctlconf.ConfigMap{}, fmt.Errorf(
			"Expected to find one config map '%s', but found multiple", name)
	}

	return found[0], nil
}
