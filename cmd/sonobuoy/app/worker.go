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
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/heptio/sonobuoy/pkg/errlog"
	"github.com/heptio/sonobuoy/pkg/plugin"
	"github.com/heptio/sonobuoy/pkg/worker"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewCmdWorker() *cobra.Command {

	var workerCmd = &cobra.Command{
		Use:    "worker",
		Short:  "Gather and send data to the sonobuoy master instance (for internal use)",
		Run:    runGather,
		Hidden: true,
		Args:   cobra.ExactArgs(0),
	}

	workerCmd.AddCommand(singleNodeCmd)
	workerCmd.AddCommand(globalCmd)

	return workerCmd
}



var globalCmd = &cobra.Command{
	Use:   "global",
	Short: "Submit results scoped to the whole cluster",
	Run:   runGatherGlobal,
	Args:  cobra.ExactArgs(0),
}

var singleNodeCmd = &cobra.Command{
	Use:   "single-node",
	Short: "Submit results scoped to a single node",
	Run:   runGatherSingleNode,
	Args:  cobra.ExactArgs(0),
}

func runGather(cmd *cobra.Command, args []string) {
	cmd.Help()
}

// sigHandler returns a channel that will receive a message after the timeout
// elapses after a SIGTERM is received.
func sigHandler(timeout time.Duration) <-chan struct{} {
	stop := make(chan struct{})
	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGTERM)
		sig := <-sigc
		logrus.WithField("signal", sig).Info("received a signal. Waiting then sending the real shutdown signal.")
		time.Sleep(timeout)
		stop <- struct{}{}
	}()
	return stop
}

// loadAndValidateConfig loads the config for this sonobuoy worker, validating
// that we have enough information to proceed.
func loadAndValidateConfig() (*plugin.WorkerConfig, error) {
	cfg, err := worker.LoadConfig()
	if err != nil {
		return nil, errors.Wrap(err, "error loading agent configuration")
	}

	var errlst []string
	if cfg.MasterURL == "" {
		errlst = append(errlst, "MasterURL not set")
	}
	if cfg.ResultsDir == "" {
		errlst = append(errlst, "ResultsDir not set")
	}
	if cfg.ResultType == "" {
		errlst = append(errlst, "ResultsType not set")
	}

	if len(errlst) > 0 {
		joinedErrs := strings.Join(errlst, ", ")
		return nil, errors.Errorf("invalid agent configuration: (%v)", joinedErrs)
	}

	return cfg, nil
}

func runGatherSingleNode(cmd *cobra.Command, args []string) {
	cfg, err := loadAndValidateConfig()
	if err != nil {
		errlog.LogError(err)
		os.Exit(1)
	}

	client, err := getHTTPClient(cfg)
	if err != nil {
		errlog.LogError(err)
		os.Exit(1)
	}

	// A single-node results URL looks like:
	// http://sonobuoy-master:8080/api/v1/results/by-node/node1/systemd_logs
	url := cfg.MasterURL + "/" + cfg.NodeName + "/" + cfg.ResultType

	err = worker.GatherResults(cfg.ResultsDir+"/done", url, client, sigHandler(plugin.GracefulShutdownPeriod*time.Second))
	if err != nil {
		errlog.LogError(err)
		os.Exit(1)
	}
}

func runGatherGlobal(cmd *cobra.Command, args []string) {
	cfg, err := loadAndValidateConfig()
	if err != nil {
		errlog.LogError(err)
		os.Exit(1)
	}

	client, err := getHTTPClient(cfg)
	if err != nil {
		errlog.LogError(err)
		os.Exit(1)
	}

	// A global results URL looks like:
	// http://sonobuoy-master:8080/api/v1/results/global/systemd_logs
	url := cfg.MasterURL + "/" + cfg.ResultType

	err = worker.GatherResults(cfg.ResultsDir+"/done", url, client, sigHandler(plugin.GracefulShutdownPeriod*time.Second))
	if err != nil {
		errlog.LogError(err)
		os.Exit(1)
	}
}

func getHTTPClient(cfg *plugin.WorkerConfig) (*http.Client, error) {
	caCertDER, _ := pem.Decode([]byte(cfg.CACert))
	if caCertDER == nil {
		return nil, errors.New("Couldn't parse CaCert PEM")
	}
	clientCertDER, _ := pem.Decode([]byte(cfg.ClientCert))
	if clientCertDER == nil {
		return nil, errors.New("Couldn't parse ClientCert PEM")
	}
	clientKeyDER, _ := pem.Decode([]byte(cfg.ClientKey))
	if clientKeyDER == nil {
		return nil, errors.New("Couldn't parse ClientKey PEM")
	}

	caCert, err := x509.ParseCertificate(caCertDER.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't parse CaCert")
	}
	clientCert, err := x509.ParseCertificate(clientCertDER.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't parse ClientCert")
	}
	clientKey, err := x509.ParseECPrivateKey(clientKeyDER.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't parse ClientKey")
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(caCert)

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				Certificates: []tls.Certificate{
					{
						Certificate: [][]byte{clientCertDER.Bytes},
						PrivateKey:  clientKey,
						Leaf:        clientCert,
					},
				},
				RootCAs: certPool,
			},
		},
	}, nil
}
