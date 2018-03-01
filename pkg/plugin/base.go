package plugin

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/heptio/sonobuoy/pkg/plugin/manifest"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
)

type Base struct {
	Definition    Definition
	SessionID     string
	Namespace     string
	SonobuoyImage string
	CleanedUp     bool
}

type TemplateData struct {
	PluginName        string
	ResultType        string
	SessionID         string
	Namespace         string
	SonobuoyImage     string
	ProducerContainer string
	MasterAddress     string
	CACert            string
	SecretName        string
}

// GetSessionID returns the session id associated with the plugin
func (b *Base) GetSessionID() string {
	return b.SessionID
}

// GetName returns the name of this Job plugin
func (b *Base) GetName() string {
	return b.Definition.Name
}

func (b *Base) GetSecretName() string {
	return fmt.Sprintf("job-%s-%s", b.GetName(), b.GetSessionID())
}

// GetResultType returns the ResultType for this plugin (to adhere to plugin.Interface)
func (b *Base) GetResultType() string {
	return b.Definition.ResultType
}

//FillTemplate populates the internal Job YAML template with the values for this particular job.
func (b *Base) GetTemplateData(hostname string, cert *tls.Certificate) (*TemplateData, error) {

	container, err := kuberuntime.Encode(manifest.Encoder, &b.Definition.Spec)
	if err != nil {
		return nil, errors.Wrapf(err, "couldn't reserialize container for job %q", b.Definition.Name)
	}

	cacert := getCACertPEM(cert)

	return &TemplateData{
		PluginName:        b.Definition.Name,
		ResultType:        b.Definition.ResultType,
		SessionID:         b.SessionID,
		Namespace:         b.Namespace,
		SonobuoyImage:     b.SonobuoyImage,
		ProducerContainer: string(container),
		MasterAddress:     b.GetMasterAddress(hostname),
		CACert:            cacert,
		SecretName:        b.GetSecretName(),
	}, nil
}

func (b *Base) GetMasterAddress(hostname string) string {
	panic("base GetMasterAddress called")
}

// MakeTLSSecret makes a Kubernetes secret object for the given TLS certificate.
func (b *Base) MakeTLSSecret(cert *tls.Certificate) (*v1.Secret, error) {
	rsaKey, ok := cert.PrivateKey.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key not ECDSA")
	}

	if len(cert.Certificate) <= 0 {
		return nil, errors.New("no certs in tls.certificate")
	}

	certDER := cert.Certificate[0]
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	keyPEM, err := getKeyPEM(rsaKey)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't PEM encode TLS key")
	}

	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.GetSecretName(),
			Namespace: b.Namespace,
		},
		Data: map[string][]byte{
			v1.TLSPrivateKeyKey: keyPEM,
			v1.TLSCertKey:       certPEM,
		},
		Type: v1.SecretTypeTLS,
	}, nil

}

// GetCACertPEM extracts the CA cert from a tls.Certificate.
// If the provided Certificate has only one certificate in the chain, the CA
// will be the leaf cert.
func getCACertPEM(cert *tls.Certificate) string {
	cacert := ""
	if len(cert.Certificate) > 0 {
		caCertDER := cert.Certificate[len(cert.Certificate)-1]
		cacert = string(pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: caCertDER,
		}))
	}
	return cacert
}

// GetKeyPEM turns an RSA Private Key into a PEM-encoded string
func getKeyPEM(key *ecdsa.PrivateKey) ([]byte, error) {
	derKEY, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: derKEY,
	}), nil
}
