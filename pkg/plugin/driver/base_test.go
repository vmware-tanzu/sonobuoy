/*
Copyright 2018 Heptio Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package driver

import (
	"crypto/ecdsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/heptio/sonobuoy/pkg/backplane/ca"
	"github.com/heptio/sonobuoy/pkg/plugin"
)

func TestMakeTLSSecret(t *testing.T) {
	auth, err := ca.NewAuthority()
	if err != nil {
		t.Fatalf("unexpected error %v making authority", err)
	}
	expectedNamespace := "test-namespace"
	expectedName := "test-name"
	sessionID := "aaaaaa11111"

	cert, err := auth.ClientKeyPair("")
	if err != nil {
		t.Fatalf("unexpected error %v making client pair", err)
	}

	driver := &Base{
		Namespace: expectedNamespace,
		Definition: plugin.Definition{
			Name: expectedName,
		},
		SessionID: sessionID,
	}

	secret, err := driver.MakeTLSSecret(cert)
	if err != nil {
		t.Fatalf("unexpected error %v making TLS Secret", err)
	}

	if secret.ObjectMeta.Name != driver.GetSecretName() {
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
