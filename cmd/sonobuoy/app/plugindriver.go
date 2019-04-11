/*
Copyright the Sonobuoy contributors 2019

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
	"fmt"
	"strings"
)

type pluginDriver string

var driverMap = map[string]pluginDriver{
	string("job"):       pluginDriver("Job"),
	string("daemonset"): pluginDriver("DaemonSet"),
}

func (d *pluginDriver) String() string { return string(*d) }
func (d *pluginDriver) Type() string   { return "pluginDriver" }

func (d *pluginDriver) Set(str string) error {
	lcase := strings.ToLower(str)
	driver, ok := driverMap[lcase]
	if !ok {
		return fmt.Errorf("unknown plugin driver %q", str)
	}
	*d = driver
	return nil
}
