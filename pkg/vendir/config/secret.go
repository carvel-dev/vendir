package config

const (
	SecretK8sCorev1BasicAuthUsernameKey = "username"
	SecretK8sCorev1BasicAuthPasswordKey = "password"

	SecretK8sCoreV1SSHAuthPrivateKey = "ssh-privatekey"
	SecretSSHAuthKnownHosts          = "ssh-knownhosts" // not part of k8s

	SecretToken = "token"
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
