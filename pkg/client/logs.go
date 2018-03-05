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

package client

import (
	"fmt"
	"io"
	"sync"

	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// message represents a buffer of logs from a container in a pod in a namespace.
type message struct {
	// header acts as the header message for the log line and an id to know where the message came from.
	header string
	// buffer is the blob that we extracted from the contianer.
	buffer []byte
}

// logStreamer has all the pieces necessary for streaming logs from a container.
type logStreamer struct {
	ns, pod, container string
	errc               chan error
	logc               chan *message
	logOpts            *v1.PodLogOptions
	client             kubernetes.Interface
}

// stream will open a connection to the pod's logs and push messages onto a channel.
func (l *logStreamer) stream() {
	req := l.client.CoreV1().Pods(l.ns).GetLogs(l.pod, l.logOpts)
	readCloser, err := req.Stream()
	if err != nil {
		l.errc <- errors.Wrapf(err, "error streaming logs from container [%v]", l.container)
		return
	}
	defer readCloser.Close()

	header := fmt.Sprintf("## Namespace: %v ## Pod: %v ## Container: %v", l.ns, l.pod, l.container)

	buf := make([]byte, 1024)
	// Loop until EOF (streaming case won't get an EOF)
	for {
		n, err := readCloser.Read(buf)
		if err != nil && err != io.EOF {
			l.errc <- errors.Wrapf(err, "error reading logs from container [%v]", l.container)
			return
		}
		l.logc <- &message{
			header: header,
			buffer: buf[:n],
		}
		if err == io.EOF {
			return
		}
	}
}

// StreamLogs finds all pods in a given namespace and writes them to the configured io.Writer.
func (c *SonobuoyClient) StreamLogs(cfg *LogConfig, client kubernetes.Interface) chan error {
	errc := make(chan error)
	agg := make(chan *message)
	// waitGroup to make sure we don't exit before we finish reading all of the logs.
	var wg sync.WaitGroup

	pods, err := client.CoreV1().Pods(cfg.Namespace).List(metav1.ListOptions{})
	if err != nil {
		errc <- err
		close(errc)
		close(agg)
		return errc
	}

	// TODO(chuckha) if we get an error back that the container is still creating maybe we could retry?
	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			wg.Add(1)
			ls := &logStreamer{
				ns:        pod.Namespace,
				pod:       pod.Name,
				container: container.Name,
				errc:      errc,
				logc:      agg,
				logOpts: &v1.PodLogOptions{
					Container: container.Name,
					Follow:    *cfg.Follow,
				},
				client: client,
			}

			go func(w *sync.WaitGroup, ls *logStreamer) {
				defer w.Done()
				ls.stream()
			}(&wg, ls)
		}
	}

	// Cleanup when finished.
	go func(wg *sync.WaitGroup, agg chan *message, errc chan error) {
		wg.Wait()
		close(agg)
		close(errc)
	}(&wg, agg, errc)

	// Do something with the messages.
	go func(cfg *LogConfig, agg chan *message) {
		// Print the header line again whenever we change which container we read from
		header := ""
		for message := range agg {
			if message.header != header {
				fmt.Fprintln(cfg.Out, message.header)
				header = message.header
			}
			fmt.Fprintf(cfg.Out, string(message.buffer))
		}
	}(cfg, agg)

	return errc
}
