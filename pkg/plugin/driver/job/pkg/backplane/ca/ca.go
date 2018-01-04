package ca

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"time"

	"github.com/pkg/errors"
)

const (
	rsaBits  = 4096
	validFor = 48 * time.Hour
	caName   = "sonobuoy-ca"
)

var pkixName = pkix.Name{
	Organization:       []string{"Heptio"},
	OrganizationalUnit: []string{"sonobuoy"},
	Country:            []string{"USA"},
	Locality:           []string{"Seattle"},
}

// Authority represents a root certificate authority that can issues
// certificates to be used for Client certs.
// Sonobuoy issues every worker a client certificate
type Authority struct {
	privKey    *rsa.PrivateKey
	cert       *x509.Certificate
	lastSerial *big.Int
}

// NewAuthority creates a new certificate authority. A new private key and root certificate will
// be generated but not returned.
func NewAuthority() (*Authority, error) {
	privKey, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't generate private key")
	}
	auth := &Authority{
		privKey: privKey,
	}
	cert, err := auth.makeCert(privKey.Public(), func(cert *x509.Certificate) {
		cert.IsCA = true
		cert.KeyUsage = x509.KeyUsageCertSign
		cert.Subject.CommonName = caName
	})
	if err != nil {
		return nil, err
	}
	auth.cert = cert
	return auth, nil
}

func (a *Authority) makeCert(pub crypto.PublicKey, mut func(*x509.Certificate)) (*x509.Certificate, error) {
	serialNumber := big.NewInt(1)
	validFrom := time.Now()
	tmpl := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkixName,
		NotBefore:             validFrom,
		NotAfter:              validFrom.Add(validFor),
		KeyUsage:              0,
		ExtKeyUsage:           []x509.ExtKeyUsage{},
		BasicConstraintsValid: true,
		// For testing and local access
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
	}
	mut(&tmpl)
	parent := a.cert
	// NewAuthority case
	if a.cert == nil {
		parent = &tmpl
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, parent, pub, a.privKey)
	if err != nil {
		return nil, errors.Wrap(err, "coouldn't make authority certificate")
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't re-parse created certificate")
	}
	return cert, nil
}

func (a *Authority) nextSerial() *big.Int {
	if a.lastSerial == nil {
		num := big.NewInt(1)
		a.lastSerial = num
		return num
	}
	// Make a copy
	return a.lastSerial.Add(a.lastSerial, big.NewInt(1))
}

// CACert is the root certificate of the CA.
func (a *Authority) CACert() *x509.Certificate {
	return a.cert
}

func (a *Authority) CACertPool() *x509.CertPool {
	pool := x509.NewCertPool()
	pool.AddCert(a.CACert())
	return pool
}

// ServerKey makes a client cert signed by out root CA. The returned certificate
// has a chain including the root CA cert.
func (a *Authority) ServerKey(name string) (*tls.Certificate, error) {
	privKey, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't generate private key")
	}
	cert, err := a.makeCert(privKey.Public(), func(cert *x509.Certificate) {
		cert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
		cert.DNSNames = []string{name}
	})
	if err != nil {
		return nil, errors.Wrap(err, "couldn't make server certificate")
	}
	return &tls.Certificate{
		Certificate: [][]byte{cert.Raw, a.cert.Raw},
		PrivateKey:  privKey,
		Leaf:        cert,
	}, nil
}

// MakeServerConfig makes a new server certificate, then returns a TLS config that uses it
// and will verify peer certificates
func (a *Authority) MakeServerConfig(name string) (*tls.Config, error) {
	cert, err := a.ServerKey(name)
	if err != nil {
		return nil, err
	}

	pool := x509.NewCertPool()
	pool.AddCert(a.cert)

	return &tls.Config{
		Certificates: []tls.Certificate{*cert},
		ServerName:   name,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    pool,
	}, nil
}

// ClientKey makes a client cert signed by out root CA. The returned certificate
// has a chain including the root CA
func (a *Authority) ClientKey(name string) (*tls.Certificate, error) {
	privKey, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't generate private key")
	}
	cert, err := a.makeCert(privKey.Public(), func(cert *x509.Certificate) {
		cert.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
		cert.DNSNames = []string{name}
	})
	if err != nil {
		return nil, errors.Wrap(err, "couldn't make server certificate")
	}
	return &tls.Certificate{
		Certificate: [][]byte{cert.Raw, a.cert.Raw},
		PrivateKey:  privKey,
		Leaf:        cert,
	}, nil
}
