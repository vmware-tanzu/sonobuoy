package ca

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"crypto/tls"
	"crypto/x509"
)

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

	capool := auth.CACertPool()

	srvName := "master.sonobuoy.local"
	srvCert, err := auth.ServerKeyPair(srvName)
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
	clientCert, err := auth.ClientKeyPair(clientName)

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

func TestServer(t *testing.T) {
	auth, err := NewAuthority()
	if err != nil {
		t.Fatalf("Couldn't create certificate authority")
	}

	testString := "Whose woods these are, I think I know.\n"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, testString)
	})

	cfg, err := auth.MakeServerConfig("127.0.0.1")
	if err != nil {
		t.Fatalf("Couldn't get server config %v", err)

	}
	srv := httptest.NewUnstartedServer(handler)
	srv.TLS = cfg
	srv.StartTLS()
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/test")
	if err == nil {
		defer resp.Body.Close()
		t.Fatalf("made request without cert, should've gotten error")
	}

	clientCert, err := auth.ClientKeyPair("client1.local")
	if err != nil {
		t.Fatalf("couldn't get client cert %v", err)
	}

	client := srv.Client()
	client.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates: []tls.Certificate{*clientCert},
			RootCAs:      auth.CACertPool(),
		},
	}

	resp, err = srv.Client().Get(srv.URL + "/test")
	if err != nil {
		t.Fatalf("expected client error to be null, got %v", err)
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("couldn't read body: %v", err)
	}

	if string(respBody) != testString {
		t.Errorf("expected %s, got %s", testString, respBody)
	}

}
