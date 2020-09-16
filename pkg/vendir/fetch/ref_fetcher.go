package fetch

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
