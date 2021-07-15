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

package tarball

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

const stoppingByTheWoods = `Whose woods these are I think I know.
His house is in the village though;
He will not see me stopping here
To watch his woods fill up with snow.

My little horse must think it queer
To stop without a farmhouse near
Between the woods and frozen lake
The darkest evening of the year.

He gives his harness bells a shake
To ask if there is some mistake.
The only other sound’s the sweep
Of easy wind and downy flake.

The woods are lovely, dark and deep,
But I have promises to keep,
And miles to go before I sleep,
And miles to go before I sleep.
`

func TestDecodeTarball(t *testing.T) {
	buffer := &bytes.Buffer{}
	gz := gzip.NewWriter(buffer)

	dirName := "testdirname"
	testData := []byte(stoppingByTheWoods)
	fileName := "stoppingByTheWoods"
	symLinkName := "poem"

	w := tar.NewWriter(gz)
	err := w.WriteHeader(&tar.Header{
		Name:     dirName,
		Mode:     0755,
		Typeflag: tar.TypeDir,
		ModTime:  time.Now(),
	})
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}
	err = w.WriteHeader(&tar.Header{
		Name:     path.Join(dirName, fileName),
		Mode:     0755,
		Typeflag: tar.TypeReg,
		Size:     int64(len(testData)),
		ModTime:  time.Now(),
	})
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	_, err = w.Write(testData)
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	err = w.WriteHeader(&tar.Header{
		Name:     path.Join(dirName, symLinkName),
		Linkname: path.Join(dirName, fileName),
		Mode:     0755,
		Typeflag: tar.TypeSymlink,
		ModTime:  time.Now(),
	})
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	err = w.Close()
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}
	err = gz.Close()
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	rBuffer := bytes.NewBuffer(buffer.Bytes())

	file, err := os.Open(path.Join("testdata/archive.tar.gz"))
	if err != nil {
		t.Fatalf("couldn't extract archive: %v", err)
	}
	defer file.Close()

	archives := []struct {
		reader io.Reader
		name   string
	}{
		{rBuffer, "go-created archive"},
		{file, "bsdtar-created archive"},
	}

	for _, archive := range archives {
		t.Run(archive.name, func(t *testing.T) {
			dir, err := ioutil.TempDir("", "tarball-test")
			if err != nil {
				t.Fatalf("Unexpected error %v", err)
			}
			defer os.RemoveAll(dir)

			err = DecodeTarball(archive.reader, dir)
			if err != nil {
				t.Fatalf("Unexpected error %v", err)
			}

			contents, err := ioutil.ReadFile(filepath.Join(dir, dirName, fileName))
			if err != nil {
				t.Fatalf("Unexpected error %v", err)
			}

			if !reflect.DeepEqual(contents, testData) {
				t.Errorf("Expected %s, got %s", testData, contents)
			}
			contents, err = ioutil.ReadFile(filepath.Join(dir, dirName, symLinkName))
			if err != nil {
				t.Fatalf("Unexpected error %v", err)
			}
			if !reflect.DeepEqual(contents, testData) {
				t.Errorf("Expected %s, got %s", testData, contents)
			}
		})
	}
}
