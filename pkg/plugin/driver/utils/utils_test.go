package utils

import (
	"crypto/ecdsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/heptio/sonobuoy/pkg/backplane/ca"
)

func TestMakeTLSSecret(t *testing.T) {
	auth, err := ca.NewAuthority()
	if err != nil {
		t.Fatalf("unexpected error %v making authority", err)
	}
	expectedNamespace := "test-namespace"
	expectedName := "test-name"

	cert, err := auth.ClientKeyPair("")
	if err != nil {
		t.Fatalf("unexpected error %v making client pair", err)
	}

	secret, err := MakeTLSSecret(cert, expectedNamespace, expectedName)
	if err != nil {
		t.Fatalf("unexpected error %v making TLS Secret", err)
	}

	if secret.ObjectMeta.Name != expectedName {
		t.Errorf("expected name %v, got %v", expectedName, secret.ObjectMeta.Name)
	}
	if secret.ObjectMeta.Namespace != expectedNamespace {
		t.Errorf("expected namespace %v, got %v", expectedNamespace, secret.ObjectMeta.Namespace)
	}

	expectedKeyBytes, err := x509.MarshalECPrivateKey(cert.PrivateKey.(*ecdsa.PrivateKey))
	if err != nil {
		t.Fatalf("unexpected error %v marshalling EC private key", err)
	}
	keyPEM, _ := pem.Decode(secret.Data["tls.key"])
	if keyPEM == nil {
		t.Fatal("couldn't decode tls.key")
	}

	if sha1.Sum(expectedKeyBytes) != sha1.Sum(keyPEM.Bytes) {
		t.Error("key fingerprint didn't match")
	}

	certPEM, _ := pem.Decode(secret.Data["tls.crt"])
	if certPEM == nil {
		t.Fatal("couldn't decode tls.crt")
	}

	if sha1.Sum(cert.Leaf.Raw) != sha1.Sum(certPEM.Bytes) {
		t.Error("cert fingerprint didn't match")
	}
}
