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
	"net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/vmware-tanzu/sonobuoy/pkg/plugin"
	"github.com/vmware-tanzu/sonobuoy/pkg/plugin/aggregation"
	"github.com/vmware-tanzu/sonobuoy/pkg/worker"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// NewCmdWorker is the cobra command that acts as the entrypoint for Sonobuoy when running
// as a sidecar with a plugin. It will wait for a 'done' file then transmit the results to the
// aggregator pod.
func NewCmdWorker() *cobra.Command {
	var workerCmd = &cobra.Command{
		Use:    "worker",
		Short:  "Gather and send data to the sonobuoy aggregator instance (for internal use)",
		Hidden: true,
		Args:   cobra.ExactArgs(0),
	}

	workerCmd.AddCommand(newSingleNodeCmd())
	workerCmd.AddCommand(newGlobalCmd())

	return workerCmd
}

func newGlobalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "global",
		Short: "Submit results scoped to the whole cluster",
		RunE:  runGatherGlobal,
		Args:  cobra.ExactArgs(0),
	}

	return cmd
}

func newSingleNodeCmd() *cobra.Command {
	var sleep int64
	cmd := &cobra.Command{
		Use:   "single-node",
		Short: "Submit results scoped to a single node",
		RunE:  runGatherSingleNode(&sleep),
		Args:  cobra.ExactArgs(0),
	}

	cmd.Flags().Int64Var(&sleep, "sleep", 0, "After sending results, keeps the process alive for N seconds to avoid restarting the container. If N<0, Sonobuoy sleeps forever.")
	return cmd
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
	if cfg.AggregatorURL == "" {
		errlst = append(errlst, "AggregatorURL not set")
	}
	if cfg.ResultsDir == "" {
		errlst = append(errlst, "ResultsDir not set")
	}
	if cfg.ResultType == "" {
		errlst = append(errlst, "ResultsType not set")
	}
	if cfg.ProgressUpdatesPort == "" {
		errlst = append(errlst, "ProgressUpdatesPort not set")
	}

	if len(errlst) > 0 {
		joinedErrs := strings.Join(errlst, ", ")
		return nil, errors.Errorf("invalid agent configuration: (%v)", joinedErrs)
	}

	return cfg, nil
}

func runGatherGlobal(cmd *cobra.Command, args []string) error {
	return runGather(true)
}

// runGatherSingleNode returns a closure which will run the data gathering and then sleep
// for the specified amount of seconds.
func runGatherSingleNode(sleep *int64) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		err := runGather(false)

		switch {
		case sleep == nil || *sleep == 0:
			// No sleep.
		case *sleep < 0:
			// Sleep forever.
			logrus.Infof("Results transmitted to aggregator.  Sleeping forever.")
			for {
				time.Sleep(60 * time.Minute)
			}
		case *sleep > 0:
			logrus.Infof("Results transmitted to aggregator. Sleeping for %v seconds", *sleep)
			time.Sleep(time.Duration(*sleep) * time.Second)
		}
		return err
	}
}

func runGather(global bool) error {
	cfg, err := loadAndValidateConfig()
	if err != nil {
		return errors.Wrap(err, "loading config")
	}

	client, err := getHTTPClient(cfg)
	if err != nil {
		return errors.Wrap(err, "getting HTTP client")
	}

	resultURL, err := url.Parse(cfg.AggregatorURL)
	if err != nil {
		return errors.Wrap(err, "parsing AggregatorURL")
	}
	progressURL, err := url.Parse(cfg.AggregatorURL)
	if err != nil {
		return errors.Wrap(err, "parsing AggregatorURL")
	}

	if global {
		// A global results URL looks like:
		// http://sonobuoy-aggregator:8080/api/v1/results/global/systemd_logs
		resultURL.Path = path.Join(aggregation.PathResultsGlobal, cfg.ResultType)
		progressURL.Path = path.Join(aggregation.PathProgressGlobal, cfg.ResultType)
	} else {
		// A single-node results URL looks like:
		// http://sonobuoy-aggregator:8080/api/v1/results/by-node/node1/systemd_logs
		resultURL.Path = path.Join(aggregation.PathResultsByNode, cfg.NodeName, cfg.ResultType)
		progressURL.Path = path.Join(aggregation.PathProgressByNode, cfg.NodeName, cfg.ResultType)
	}

	go worker.RelayProgressUpdates(cfg.ProgressUpdatesPort, progressURL.String(), client)
	err = worker.GatherResults(filepath.Join(cfg.ResultsDir, "done"), resultURL.String(), client, sigHandler(plugin.GracefulShutdownPeriod*time.Second))

	return errors.Wrap(err, "gathering results")
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
