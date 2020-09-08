package config

const (
	SecretK8sCorev1BasicAuthUsernameKey = "username"
	SecretK8sCorev1BasicAuthPasswordKey = "password"
)

type Secret struct {
	APIVersion string
	Kind       string

	Metadata SecretMetadata
	Data     map[string]string
}

type SecretMetadata struct {
	Name string
}
