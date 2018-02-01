package authtest

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/heptio/sonobuoy/pkg/backplane/ca"
)

// Server is an extension of httptest.Server that uses our own CA
type Server struct {
	*httptest.Server
	auth *ca.Authority
	t    *testing.T
}

// NewServer is a passthrough wrapper for httptest.NewServer. It does not
// use the CA at all. This provided only for debugging purposes.
func NewServer(handle http.Handler, t *testing.T) *Server {
	return &Server{
		Server: httptest.NewServer(handle),
		t:      t,
	}
}

// NewTLSServer Wraps httptest.NewTLSServer, injecting our CA and TLS config
func NewTLSServer(handle http.Handler, t *testing.T) *Server {
	auth, err := ca.NewAuthority()
	if err != nil {
		t.Fatalf("Couldn't create certificate authority")
	}
	cfg, err := auth.MakeServerConfig("127.0.0.1")
	if err != nil {
		t.Fatalf("Couldn't get server config %v", err)
	}
	srv := httptest.NewUnstartedServer(handle)
	srv.TLS = cfg
	srv.StartTLS()
	return &Server{
		Server: srv,
		auth:   auth,
		t:      t,
	}
}

// Client wraps httptest.Server.Client(), injecting our CA and client cert
func (s *Server) Client() *http.Client {
	if s.auth == nil {
		return s.Server.Client()
	}
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
