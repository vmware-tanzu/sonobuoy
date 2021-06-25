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
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const (
	bufSize = 4096

	// maxBackoffSeconds is the maximum time to backoff when waiting for pods to startup. Prevents
	// unhelpfully large noop periods.
	maxBackoffSeconds = 32
)

// Reader provides an io.Reader interface to a channel of bytes. The first error
// received on the error channel will be returned by Read after the bytestream
// is drained and on all subsequent calls to Read. It is the responsibility of
// the program writing to bytestream to write an io.EOF to the error stream when
// it is done and close all channels.
type Reader struct {
	bytestream chan []byte
	errc       chan error

	// Used for when one message is too large for the input buffer
	// TODO(chuckha) consider alternative data structures here
	overflowBuffer []byte
	err            error
	empty          bool
}

// NewReader returns a configured Reader.
func NewReader(bytestream chan []byte, errc chan error) *Reader {
	reader := &Reader{
		bytestream:     bytestream,
		errc:           errc,
		overflowBuffer: []byte{},
		err:            nil,
	}
	return reader
}

// Read tries to fill up the passed in byte slice with messages from the channel.
// Read manages the message overflow ensuring no bytes are missed.
// If an error is set on the reader it will return the error immediately.
func (r *Reader) Read(p []byte) (int, error) {
	// Send any overflow before grabbing new messages.
	if len(r.overflowBuffer) > 0 {
		// If we need to chunk it, copy as much as we can and reduce the overflow buffer.
		if len(r.overflowBuffer) > len(p) {
			copy(p, r.overflowBuffer[:len(p)])
			r.overflowBuffer = r.overflowBuffer[len(p):]
			return len(p), nil
		}
		// At this point the entire overflow will fit into the buffer.
		copy(p, r.overflowBuffer)
		n := len(r.overflowBuffer)
		r.overflowBuffer = nil
		return n, nil
	}

	// Return the error if set only if byte channel is empty.
	if r.err != nil && r.empty {
		return 0, r.err
	}

	// Be sure that we are draining any errors put on the errc and unblock senders.
	var err error
	var data []byte
	var ok bool
	select {
	case err, ok = <-r.errc:
		// Treat the errors as data as well. This ensures that the user reading the messages
		// is aware of the errors as they happen and not only after all messages are processed.
		// This is important in the case of tailing logs where one stream may never finish at all.
		if err != nil && err != io.EOF {
			data = []byte(fmt.Sprint(err))
		}

		// Save only the first error. Assume io.EOF if closed/nil.
		switch {
		case err != nil:
			r.err = err
		case err == nil || !ok:
			r.err = io.EOF
		}

		// If the channel is closed, set to nil to avoid reading this channel again.
		if !ok {
			r.errc = nil
		}
	case data, ok = <-r.bytestream:
		if !ok {
			r.empty = true
			// Ensure we don't choose this path again in the select.
			r.bytestream = nil
		}
	}

	// Avoid the case where a channel was closed and we don't have data to process. This would end up returning
	// (0, nil) which is discouraged but also ends up leaving `p` unmodified which is against the documentation
	// of io.Reader.
	if len(data) == 0 {
		return r.Read(p)
	}

	// The incoming data is bigger than size of the remaining size of the buffer. Save overflow data for next read.
	if len(data) > len(p) {
		copy(p, data[:len(p)])
		r.overflowBuffer = data[len(p):]
		return len(p), nil
	}

	// We have enough headroom in the buffer, copy all of it.
	copy(p, data)
	return len(data), nil
}

// getPodsToStreamLogs retrieves the pods to stream logs from. If a plugin name has been provided, retrieve the pods with
// only the plugin label matching that plugin name. If no pods are found, or no plugin has been specified, retrieve
// all pods within the namespace. It will immediately return an error if unabel to list pods, but will otherwise
// add pods onto the channel in a separate go routine so that this method does not block. It closes the pod channel once
// all the pods have been reported.
func getPodsToStreamLogs(client kubernetes.Interface, cfg *LogConfig, podCh chan *v1.Pod) error {
	listOptions := metav1.ListOptions{}
	if cfg.Plugin != "" {
		selector := metav1.AddLabelToSelector(&metav1.LabelSelector{}, "sonobuoy-plugin", cfg.Plugin)
		listOptions = metav1.ListOptions{LabelSelector: metav1.FormatLabelSelector(selector)}
	}

	podList, err := client.CoreV1().Pods(cfg.Namespace).List(context.TODO(), listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to list pods")
	}

	go func() {
		for _, p := range podList.Items {
			podCh <- &p
		}
		close(podCh)
	}()

	return nil
}

// watchPodsToStreamLogs creates a watch for the desired pods and, as it gets events for new pods will add them onto the pod channel.
//  If a plugin name has been provided, retrieve the pods with only the plugin label matching that plugin name. If no pods are found,
// or no plugin has been specified, retrieve all pods within the namespace. It will return an error if unable to create the watcher
// but will continue to add pods to the channel in a separate go routine.
func watchPodsToStreamLogs(client kubernetes.Interface, cfg *LogConfig, podCh chan *v1.Pod) error {
	var timeoutSeconds int64 = 5
	listOptions := metav1.ListOptions{TimeoutSeconds: &timeoutSeconds}
	if cfg.Plugin != "" {
		selector := metav1.AddLabelToSelector(&metav1.LabelSelector{}, "sonobuoy-plugin", cfg.Plugin)
		listOptions = metav1.ListOptions{LabelSelector: metav1.FormatLabelSelector(selector)}
	}

	lw := &cache.ListWatch{
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return client.CoreV1().Pods(cfg.Namespace).Watch(context.TODO(), listOptions)
		},
	}

	watcher, err := lw.Watch(listOptions)
	if err != nil {
		return errors.Wrap(err, "failed to start watching pod logs")
	}
	ch := watcher.ResultChan()

	go func() {
		for {
			v := <-ch
			if v.Type == watch.Added && v.Object != nil {
				switch t := v.Object.(type) {
				case *v1.Pod:
					podCh <- t
				default:
				}
			}
		}
	}()
	return nil
}

// LogReader configures a Reader that provides an io.Reader interface to a merged stream of logs from various containers.
func (s *SonobuoyClient) LogReader(cfg *LogConfig) (*Reader, error) {
	if cfg == nil {
		return nil, errors.New("nil LogConfig provided")
	}

	if err := cfg.Validate(); err != nil {
		return nil, errors.Wrap(err, "config validation failed")
	}

	client, err := s.Client()
	if err != nil {
		return nil, err
	}
	podCh := make(chan *v1.Pod)
	var wg sync.WaitGroup

	if cfg.Follow {
		// Extra waitGroup item ensures we just keep processing forever.
		wg.Add(1)
		if err := watchPodsToStreamLogs(client, cfg, podCh); err != nil {
			return nil, errors.Wrap(err, "failed to watch for new pods")
		}
	} else {
		if err := getPodsToStreamLogs(client, cfg, podCh); err != nil {
			return nil, errors.Wrap(err, "failed to list pods")
		}
	}

	errc := make(chan error)
	agg := make(chan *message)

	drainPodChannelAndStartStreaming := func() {
		for pod := range podCh {
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
						Follow:    cfg.Follow,
					},
					client: client,
				}

				go func(w *sync.WaitGroup, ls *logStreamer) {
					defer w.Done()
					ls.stream()
				}(&wg, ls)
			}
		}
	}

	// When following logs we will never drain the channel of pods since we are always waiting for new ones. So
	// just put that into a go routine which will start them as soon as possible. If not following pods, just
	// ensure we start streaming logs from all the existing pods before waiting for them to finish up or else you
	// introduce a race.
	if cfg.Follow {
		go drainPodChannelAndStartStreaming()
	} else {
		drainPodChannelAndStartStreaming()
	}

	// Cleanup when finished.
	go func(wg *sync.WaitGroup, agg chan *message, errc chan error) {
		wg.Wait()
		close(agg)
		close(errc)
	}(&wg, agg, errc)

	return NewReader(applyHeaders(agg), errc), nil
}

// message represents a buffer of logs from a container in a pod in a namespace.
type message struct {
	// preamble acts as the id for a particular container as well as the data to print before the actual logs.
	preamble string
	// buffer is the blob of logs that we extracted from the container.
	buffer []byte
}

func newMessage(preamble string, data []byte) *message {
	// Copy the bytes out of data so that byte slice can be reused.
	d := make([]byte, len(data))
	copy(d, data)
	return &message{
		preamble: preamble,
		buffer:   d,
	}
}

// logStreamer writes logs from a container to a fan-in channel.
type logStreamer struct {
	ns, pod, container string
	errc               chan error
	logc               chan *message
	logOpts            *v1.PodLogOptions
	client             kubernetes.Interface
}

func (l *logStreamer) podName() string {
	return fmt.Sprintf("%s/%s/%s", l.ns, l.pod, l.container)
}

func isContainerRunning(statuses *[]v1.ContainerStatus, containerName string) bool {
	for _, cs := range *statuses {
		if cs.Name == containerName && cs.State.Running != nil {
			return true
		}
	}
	return false
}

func (l *logStreamer) waitForContainerRunning() error {
	backoffSeconds := 1 * time.Second
	for {
		pod, err := l.client.CoreV1().Pods(l.ns).Get(context.TODO(), l.pod, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get pod [%s/%s]", l.ns, l.pod)
		}

		if isContainerRunning(&pod.Status.ContainerStatuses, l.container) {
			return nil
		}

		fmt.Printf("container %v, is not running, will retry streaming logs in %v seconds\n", l.podName(), backoffSeconds)
		time.Sleep(backoffSeconds)
		backoffSeconds *= 2
		if backoffSeconds > maxBackoffSeconds*time.Second {
			backoffSeconds = maxBackoffSeconds * time.Second
		}
	}
}

// stream will open a connection to the pod's logs and push messages onto a fan-in channel.
func (l *logStreamer) stream() {
	if l.logOpts.Follow {
		err := l.waitForContainerRunning()
		if err != nil {
			l.errc <- errors.Wrapf(err, "failed to wait for container [%v] in pod [%s/%s] to start running", l.container, l.ns, l.pod)
			return
		}
	}

	req := l.client.CoreV1().Pods(l.ns).GetLogs(l.pod, l.logOpts)
	readCloser, err := req.Stream(context.TODO())
	if err != nil {
		l.errc <- errors.Wrapf(err, "error streaming logs from container [%v]", l.container)
		return
	}
	defer readCloser.Close()

	// newline because logs have new lines in them
	preamble := fmt.Sprintf("namespace=%q pod=%q container=%q\n", l.ns, l.pod, l.container)

	buf := make([]byte, bufSize)
	// Loop until EOF (streaming case won't get an EOF)
	for {
		n, err := readCloser.Read(buf)
		if err != nil && err != io.EOF {
			l.errc <- errors.Wrapf(err, "error reading logs from container [%v]", l.container)
			return
		}
		if n > 0 {
			l.logc <- newMessage(preamble, buf[:n])
		}
		if err == io.EOF {
			return
		}
	}
}

// applyHeaders takes a channel of messages and transforms it into a channel of bytes.
// applyHeaders will write headers to the byte stream as appropriate.
func applyHeaders(mesc chan *message) chan []byte {
	out := make(chan []byte)
	go func() {
		header := ""
		for message := range mesc {
			// Add the header if the header is different (ie the message is coming from a different source)
			if message.preamble != header {
				out <- []byte(message.preamble)
				header = message.preamble
			}
			out <- message.buffer
		}
		close(out)
	}()
	return out
}
