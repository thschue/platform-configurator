package helpers

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"golang.org/x/crypto/ssh"
)

func GenerateSSHKeyPair() (privateKey, publicKey string, err error) {
	// Generate a new private key.
	privateKeyObj, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate private key: %w", err)
	}

	// Generate the corresponding public key.
	publicKeyBytes, err := ssh.NewPublicKey(&privateKeyObj.PublicKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate public key: %w", err)
	}
	publicKey = string(ssh.MarshalAuthorizedKey(publicKeyBytes))

	// Encode the private key to PEM format.
	privateKeyPEM := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKeyObj),
		},
	)

	privateKey = string(privateKeyPEM)

	return privateKey, publicKey, nil
}
