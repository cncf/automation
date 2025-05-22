package remote

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	cryptorand "crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"golang.org/x/crypto/ssh"
)

type SSHKeyPair struct {
	SSHAuth   ssh.AuthMethod
	PublicKey string
}

func CreateSSHKeyPair() (*SSHKeyPair, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P384(), cryptorand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating ssh key: %w", err)
	}

	var privateKeyBytes bytes.Buffer
	{
		b, err := x509.MarshalECPrivateKey(privateKey)
		if err != nil {
			return nil, fmt.Errorf("marshaling private key: %w", err)
		}
		privateKeyPEM := &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
		if err := pem.Encode(&privateKeyBytes, privateKeyPEM); err != nil {
			return nil, fmt.Errorf("encoding private key: %w", err)
		}
	}

	// generate and write public key
	sshPublicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("building ssh public key: %w", err)
	}

	// Create signer
	signer, err := ssh.ParsePrivateKey(privateKeyBytes.Bytes())
	if err != nil {
		return nil, fmt.Errorf("parsing ssh key: %w", err)
	}

	sshAuth := ssh.PublicKeys(signer)

	return &SSHKeyPair{
		SSHAuth:   sshAuth,
		PublicKey: string(ssh.MarshalAuthorizedKey(sshPublicKey)),
	}, nil
}
