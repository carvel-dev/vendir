package directory

import (
	"fmt"
)

const (
	k8s_corev1_BasicAuthUsernameKey = "username"
	k8s_corev1_BasicAuthPasswordKey = "password"
)

type RefFetcher interface {
	GetSecret(string) (Secret, error)
}

type Secret struct {
	Name string
	Data map[string][]byte
}

type NoopRefFetcher struct{}

var _ RefFetcher = NoopRefFetcher{}

func (f NoopRefFetcher) GetSecret(name string) (Secret, error) {
	return Secret{}, fmt.Errorf("Not found")
}
