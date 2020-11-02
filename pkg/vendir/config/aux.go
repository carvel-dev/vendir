package config

const (
	SecretK8sCorev1BasicAuthUsernameKey = "username"
	SecretK8sCorev1BasicAuthPasswordKey = "password"

	SecretK8sCoreV1SSHAuthPrivateKey = "ssh-privatekey"
	SecretSSHAuthKnownHosts          = "ssh-knownhosts" // not part of k8s

	SecretToken = "token"
)

// There structs have minimal used set of fields from their K8s representations.

type GenericMetadata struct {
	Name string
}

type Secret struct {
	APIVersion string
	Kind       string

	Metadata GenericMetadata
	Data     map[string][]byte
}

type ConfigMap struct {
	APIVersion string
	Kind       string

	Metadata GenericMetadata
	Data     map[string]string
}
