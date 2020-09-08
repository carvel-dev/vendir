package directory

import (
	"fmt"

	ctlconf "github.com/k14s/vendir/pkg/vendir/config"
)

type RefFetcher interface {
	GetSecret(string) (ctlconf.Secret, error)
}

type NoopRefFetcher struct{}

var _ RefFetcher = NoopRefFetcher{}

func (f NoopRefFetcher) GetSecret(name string) (ctlconf.Secret, error) {
	return ctlconf.Secret{}, fmt.Errorf("Not found")
}

type NamedRefFetcher struct {
	secrets []ctlconf.Secret
}

var _ RefFetcher = NamedRefFetcher{}

func NewNamedRefFetcher(secrets []ctlconf.Secret) NamedRefFetcher {
	return NamedRefFetcher{secrets}
}

func (f NamedRefFetcher) GetSecret(name string) (ctlconf.Secret, error) {
	var foundSecrets []ctlconf.Secret
	for _, secret := range f.secrets {
		if secret.Metadata.Name == name {
			foundSecrets = append(foundSecrets, secret)
		}
	}

	if len(foundSecrets) == 0 {
		return ctlconf.Secret{}, fmt.Errorf(
			"Expected to find one secret '%s', but found none", name)
	}
	if len(foundSecrets) > 1 {
		return ctlconf.Secret{}, fmt.Errorf(
			"Expected to find one secret '%s', but found multiple", name)
	}

	return foundSecrets[0], nil
}
