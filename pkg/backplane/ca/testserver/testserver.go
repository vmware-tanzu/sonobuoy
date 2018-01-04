package testserver

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/heptio/sonobuoy/pkg/backplane/ca"
)

// TestServer is an extension of httptest.Server that uses our own CA
type TestServer struct {
	*httptest.Server
	auth *ca.Authority
	t    *testing.T
}

// NewTLSServer Wraps httptest.NewTLSServer, injecting our CA and TLS config
func NewTLSServer(handle http.Handler, t *testing.T) *TestServer {
	auth, err := ca.NewAuthority()
	if err != nil {
		t.Fatalf("Couldn't create certificate authority")
	}
	cfg, err := auth.MakeServerConfig("")
	if err != nil {
		t.Fatalf("Couldn't get server config %v", err)
	}
	srv := httptest.NewUnstartedServer(handle)
	srv.TLS = cfg
	srv.StartTLS()
	return &TestServer{
		Server: srv,
		auth:   auth,
		t:      t,
	}
}

// Client wraps httptest.Server.Client(), injecting our CA and client cert
func (s *TestServer) Client() *http.Client {
	clientCert, err := s.auth.ClientKeyPair("client1.local")
	if err != nil {
		s.t.Fatalf("couldn't get client cert %v", err)
	}
	client := s.Server.Client()
	client.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates: []tls.Certificate{*clientCert},
			RootCAs:      s.auth.CACertPool(),
		},
	}
	return client
}
