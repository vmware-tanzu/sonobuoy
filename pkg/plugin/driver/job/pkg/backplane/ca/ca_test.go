package ca

import (
	"math/big"
	"testing"

	"crypto/x509"
)

const dnsName = "test.authority.local"

func TestSerial(t *testing.T) {
	auth, err := NewAuthority()
	if err != nil {
		t.Fatalf("Couldn't create certificate authority")
	}
	s1 := auth.nextSerial()
	if s1.Cmp(big.NewInt(2)) != 0 {
		t.Errorf("expected 2, got %d", s1.Int64())
	}
	s2 := auth.nextSerial()
	if s2.Cmp(big.NewInt(3)) != 0 {
		t.Errorf("expected 3, got %d", s2.Int64())
	}
}

func TestCA(t *testing.T) {
	auth, err := NewAuthority()
	if err != nil {
		t.Fatalf("Couldn't create certificate authority")
	}

	capool := x509.NewCertPool()
	capool.AddCert(auth.CACert())

	srvName := "master.sonobuoy.local"
	srvCert, err := auth.ServerKey(srvName)
	if err != nil {
		t.Errorf("couldn't get server cert")
	} else {
		_, err = srvCert.Leaf.Verify(x509.VerifyOptions{
			Roots:     capool,
			DNSName:   srvName,
			KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		})

		if err != nil {
			t.Errorf("Expected server key to verify, got error %v", err)
		}
	}

	clientName := "worker1.sonobuoy.local"
	clientCert, err := auth.ClientKey(clientName)

	if err != nil {
		t.Errorf("couldn't get server cert")
	} else {
		_, err = clientCert.Leaf.Verify(x509.VerifyOptions{
			Roots:     capool,
			DNSName:   clientName,
			KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		})

		if err != nil {
			t.Errorf("Expected client key to verify, got error %v", err)
		}
	}

}
