/*
Copyright 2017 Heptio Inc.

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

package aggregation

import (
	"strconv"
	"time"

	"github.com/heptio/sonobuoy/pkg/plugin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Run runs an aggregation server and gathers results, in accordance with the
// given sonobuoy configuration.
//
// Basic workflow:
//
// 1. Create the aggregator object (`aggr`) to keep track of results
// 2. Launch the HTTP server with the aggr's HandleHTTPResult function as the
//    callback
// 3. Run all the aggregation plugins, monitoring each one in a goroutine,
//    configuring them to send failure results through a shared channel
// 4. Hook the shared monitoring channel up to aggr's IngestResults() function
// 5. Block until aggr shows all results accounted for (results come in through
//    the HTTP callback), stopping the HTTP server on completion
func Run(client kubernetes.Interface, plugins []plugin.Interface, cfg plugin.AggregationConfig, outdir string) error {
	// Construct a list of things we'll need to dispatch
	if len(plugins) == 0 {
		logrus.Info("Skipping host data gathering: no plugins defined")
		return nil
	}

	// Get a list of nodes so the plugins can properly estimate what
	// results they'll give.
	// TODO: there are other places that iterate through the CoreV1.Nodes API
	// call, we should only do this in one place and cache it.
	nodes, err := client.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return errors.WithStack(err)
	}

	// Find out what results we should expect for each of the plugins
	var expectedResults []plugin.ExpectedResult
	for _, p := range plugins {
		expectedResults = append(expectedResults, p.ExpectedResults(nodes.Items)...)
	}

	logrus.Infof("Starting server Expected Results: %v", expectedResults)

	// 1. Await results from each plugin
	aggr := NewAggregator(outdir+"/plugins", expectedResults)
	doneAggr := make(chan bool, 1)
	monitorCh := make(chan *plugin.Result, len(expectedResults))
	stopWaitCh := make(chan bool, 1)

	go func() {
		aggr.Wait(stopWaitCh)
		doneAggr <- true
	}()

	// 2. Launch the aggregation server
	srv := NewServer(cfg.BindAddress+":"+strconv.Itoa(cfg.BindPort), aggr.HandleHTTPResult)
	doneServ := make(chan error)
	go func() {
		doneServ <- srv.Start()
	}()

	// 3. Launch each plugin, to dispatch workers which submit the results back
	for _, p := range plugins {
		logrus.Infof("Running (%v) plugin", p.GetName())
		err := p.Run(client)
		// Have the plugin monitor for errors
		go p.Monitor(client, nodes.Items, monitorCh)
		if err != nil {
			return errors.Wrapf(err, "error running plugin %v", p.GetName())
		}
	}
	// 4. Have the aggregator plumb results from each plugins' monitor function
	go aggr.IngestResults(monitorCh)

	// Ensure we only wait for results for a certain time
	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(time.Duration(cfg.TimeoutSeconds) * time.Second)
		timeout <- true
	}()

	// 5. Wait for aggr to show that all results are accounted for
	select {
	case <-timeout:
		srv.Stop()
		stopWaitCh <- true
		return errors.Errorf("timed out waiting for plugins, shutting down HTTP server")
	case err := <-doneServ:
		stopWaitCh <- true
		if err != nil {
			return err
		}
	case <-doneAggr:
		break
	}

	return nil
}

func Cleanup(client kubernetes.Interface, plugins []plugin.Interface) {
	// Cleanup after each plugin
	for _, p := range plugins {
		p.Cleanup(client)
	}
}
