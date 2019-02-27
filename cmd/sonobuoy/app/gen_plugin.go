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

package app

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"math/big"
	"os"

	"github.com/heptio/sonobuoy/pkg/config"
	"github.com/heptio/sonobuoy/pkg/errlog"
	"github.com/heptio/sonobuoy/pkg/plugin"
	"github.com/heptio/sonobuoy/pkg/plugin/loader"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	placeholderHostname = "<hostname>"
)

// GenPluginConfig are the input options for running
type GenPluginConfig struct {
	Paths      []string
	PluginName string
}

var genPluginOpts GenPluginConfig

func NewCmdGenPlugin() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "plugin",
		Short:  "Generates the manifest Sonobuoy uses to run a worker for the given plugin",
		Run:    genPluginManifest,
		Hidden: true,
		Args:   cobra.ExactArgs(1),
	}

	cmd.PersistentFlags().StringArrayVarP(
		&genPluginOpts.Paths, "paths", "p", []string{".", "./examples/plugins.d/"},
		"the paths to search for the plugins in. Defaults to . and ./plugins.d/",
	)
	return cmd
}

func genPluginManifest(cmd *cobra.Command, args []string) {
	genPluginOpts.PluginName = args[0]
	code := 0
	manifest, err := generatePluginManifest(genPluginOpts)
	if err == nil {
		fmt.Printf("%s\n", manifest)
	} else {
		errlog.LogError(errors.Wrap(err, "error attempting to generate sonobuoy manifest"))
		code = 1
	}
	os.Exit(code)
}

func generatePluginManifest(cfg GenPluginConfig) ([]byte, error) {
	plugins, err := loader.LoadAllPlugins(
		config.DefaultNamespace,
		config.DefaultImage,
		"Always",
		cfg.Paths,
		[]plugin.Selection{{Name: cfg.PluginName}},
	)
	if err != nil {
		return nil, err
	}

	if len(plugins) != 1 {
		return nil, fmt.Errorf("expected 1 plugin, got %v", len(plugins))
	}

	cert, err := genCert()
	if err != nil {
		return nil, err
	}

	return plugins[0].FillTemplate(placeholderHostname, cert)
}

func genCert() (*tls.Certificate, error) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't generate private key")
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(0),
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &privKey.PublicKey, privKey)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't create certificate")
	}

	return &tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  privKey,
	}, nil
}
