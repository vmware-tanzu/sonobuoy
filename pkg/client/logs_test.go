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
	"io"
	"testing"

	"github.com/pkg/errors"
)

func TestLateErrors(t *testing.T) {
	quotes := []string{
		"WHAT ONE DOES WHEN FACED WITH THE TRUTH IS MORE DIFFICULT THAN YOU’D THINK.",
		"YOU ARE STRONGER THAN YOU BELIEVE. YOU HAVE GREATER POWERS THAN YOU KNOW.",
		"YOU LET THIS LITTLE THING TELL YOU WHAT TO DO?",
		"I’M WILLING TO FIGHT FOR THOSE WHO CANNOT FIGHT FOR THEMSELVES.",
	}
	bytestream := make(chan []byte)
	go func() {
		for _, quote := range quotes {
			bytestream <- []byte(quote)
		}
		close(bytestream)
	}()
	errc := make(chan error)

	reader := NewReader(bytestream, errc)

	// read the entire first message.
	mybuf := make([]byte, len(quotes[0]))

	n, err := reader.Read(mybuf)
	if err != nil {
		t.Fatalf("expected nil but got: %v", err)
	}
	if n != len(mybuf) {
		t.Fatalf("unexpected number of bytes read: %v", n)
	}

	go func() { errc <- errors.New("introduce an error") }()

	// We are guaranteed to eventually get the error because we never close bytestream.
	errcount := 0
	for i := 0; i <= 3; i++ {
		_, err := reader.Read(mybuf)
		if err != nil && err != io.EOF {
			errcount++
		}
	}
	if errcount == 0 {
		t.Fatalf("Never saw an expected error.")
	}
}

func TestLogEarlyErrors(t *testing.T) {
	input := "sonobuoy will help you on your way to greatness"
	bytestream := make(chan []byte)
	go func() {
		defer close(bytestream)
		bytestream <- []byte(input)
	}()
	errc := make(chan error)
	go func() { errc <- errors.New("A seriously bad error") }()

	reader := NewReader(bytestream, errc)

	mybuf := make([]byte, 1024)
	errcount := 0
	for i := 0; i <= 5; i++ {
		_, err := reader.Read(mybuf)
		if err != nil && err != io.EOF {
			errcount++
		}
	}
	if errcount == 0 {
		t.Fatal("did not receive any errors but there should be one.")
	}
}

func TestLogReaderNoError(t *testing.T) {
	testcases := []struct {
		name          string
		input         []string
		bufsize       int
		expectedReads []string
	}{
		{
			name:          "tiny buffer, simple input",
			input:         []string{"Hello world 0"},
			bufsize:       1,
			expectedReads: []string{"H", "e", "l", "l", "o", " ", "w", "o", "r", "l", "d", " ", "0", ""},
		},
		{
			name:          "small buffer, simple input",
			input:         []string{"Hello world 0"},
			bufsize:       2,
			expectedReads: []string{"He", "ll", "o ", "wo", "rl", "d ", "0"},
		},
		{
			name:          "big buffer, simple input",
			input:         []string{"Hello world 0"},
			bufsize:       1000,
			expectedReads: []string{"Hello world 0"},
		},
		{
			name:          "exact buffer, simple input",
			input:         []string{"Hello world 0"},
			bufsize:       len("Hello world 0"),
			expectedReads: []string{"Hello world 0"},
		},
		{
			name: "big buffer, small messages",
			input: []string{
				"Once you start down the dark path, forever will it dominate your destiny.",
				"Luminous beings are we, not this crude matter.",
				"Fear is the path to the dark side. Fear leads to anger. Anger leads to hate. Hate leads to suffering.",
			},
			bufsize: 1024,
			expectedReads: []string{
				"Once you start down the dark path, forever will it dominate your destiny.",
				"Luminous beings are we, not this crude matter.",
				"Fear is the path to the dark side. Fear leads to anger. Anger leads to hate. Hate leads to suffering.",
			},
		},
		{
			name: "small buffer, big input",
			input: []string{
				"this is some log line",
				"this is another log line",
				"this is a third log line!!",
			},
			bufsize: 10,
			expectedReads: []string{
				"this is so",
				"me log lin",
				"e",
				"this is an",
				"other log ",
				"line",
				"this is a ",
				"third log ",
				"line!!",
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			bytestream := make(chan []byte)
			errc := make(chan error)

			go func(data chan []byte, e chan error, inputs []string) {
				for _, input := range inputs {
					data <- []byte(input)
				}
				close(data)
				errc <- io.EOF
			}(bytestream, errc, tc.input)
			reader := NewReader(bytestream, errc)
			mybuf := make([]byte, tc.bufsize)
			i := 0
			for ; ; i++ {
				n, err := reader.Read(mybuf)
				if err != nil && err != io.EOF {
					t.Fatalf("Expected no errors got %v", err)
				}
				if err == io.EOF {
					break
				}
				if n > len(mybuf) {
					t.Fatalf("n is too big: %v mybuf is only %v", n, len(mybuf))
				}
				if i >= len(tc.expectedReads) {
					t.Fatalf("Too many actual reads, not enough expected reads. BUF: %v", mybuf[:n])
				}
				if len(mybuf[:n]) != len(tc.expectedReads[i]) {
					t.Errorf("Expected to read %v bytes, got %v buf: '%v' expected: '%v'", len(tc.expectedReads[i]), n, string(mybuf[:n]), tc.expectedReads[i])
				}
				if string(mybuf[:n]) != tc.expectedReads[i] {
					t.Errorf("Expected '%v' got '%v'", tc.expectedReads[i], string(mybuf[:n]))
				}
			}
			i++ // add one to i for the final read.
			if i < len(tc.expectedReads) {
				t.Fatalf("Expected Read to be called %v times but was only called %v times", len(tc.expectedReads), i)
			}
		})
	}
}
