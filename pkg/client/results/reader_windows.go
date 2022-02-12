//go:build windows
// +build windows

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

package results

import (
	"fmt"
	"io"
	"os"
)

// fileInfoToReader takes the given FileInfo object and tries to return a reader
// for the data. In the case of normal FileInfo objects (e.g. from os.Stat())
// you need to provide the full path to the file so it can be opened since the
// FileInfo object only contains the name but not the directory.
func fileInfoToReader(info os.FileInfo, path string) (io.Reader, error) {
	switch v := info.Sys().(type) {
	case io.Reader:
		return info.Sys().(io.Reader), nil
	default:
		return nil, fmt.Errorf("info.Sys() (name=%v) is type %v and unable to be used as an io.Reader", info.Name(), v)
	}
}
