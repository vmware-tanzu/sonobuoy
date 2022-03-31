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
	"context"
	"fmt"

	"github.com/pkg/errors"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// RBACMode determines whether to enable or disable RBAC for a Sonobuoy run
type RBACMode string

var (
	//ErrRBACNoClient is the error returned when we need a client but didn't get on
	ErrRBACNoClient = errors.New(`can't use nil client with "detect" RBAC mode`)
)

const (
	// DisableRBACMode means rbac is always disable
	DisableRBACMode RBACMode = "Disable"
	// EnabledRBACMode means rbac is always enabled
	EnabledRBACMode RBACMode = "Enable"
	// DetectRBACMode means "query the server to see if RBAC is enabled"
	DetectRBACMode RBACMode = "Detect"
)

var rbacModeMap = map[string]RBACMode{
	string(DisableRBACMode): DisableRBACMode,
	string(EnabledRBACMode): EnabledRBACMode,
	string(DetectRBACMode):  DetectRBACMode,
}

// String needed for pflag.Value.
func (r *RBACMode) String() string { return string(*r) }

// Type needed for pflag.Value.
func (r *RBACMode) Type() string { return "RBACMode" }

// Set the RBACMode to the given string, or error if it's not a known RBAC mode.
func (r *RBACMode) Set(str string) error {
	// Allow lowercase on the command line
	upcase := cases.Title(language.AmericanEnglish).String(str)
	mode, ok := rbacModeMap[upcase]
	if !ok {
		return fmt.Errorf("unknown RBAC mode %s", str)
	}
	*r = mode
	return nil
}

const apiRootPath = "/apis"

var supportedRBACGroupVersion = rbacv1.SchemeGroupVersion.String()

// Enabled retrieves whether to enable or disable rbac. If the mode is disable or
// enabled, the client is unused. If the mode is "detect", the client will be
// used to query the server's API groups and detect whether an RBAC api group exists.
func (r *RBACMode) Enabled(client kubernetes.Interface) (bool, error) {
	if r == nil {
		return false, fmt.Errorf("RBACMode is nil")
	}
	switch *r {
	case DisableRBACMode:
		return false, nil
	case EnabledRBACMode:
		return true, nil
	case DetectRBACMode:
		if client == nil {
			return false, ErrRBACNoClient
		}
		return checkRBACEnabled(client.Discovery().RESTClient())
	default:
		return false, fmt.Errorf("Unknown RBAC mode %v", *r)
	}
}

func checkRBACEnabled(client rest.Interface) (bool, error) {
	// This checks that the rbac API group is enabled. Note that this does NOT
	// mean that RBAC is enabled. It simply means the API is present, and
	// therefore will not error if we send RBAC objects.
	result, err := client.Get().AbsPath(apiRootPath).Do(context.TODO()).Get()
	if err != nil {
		return false, errors.Wrap(err, "couldn't retrieve API groups")
	}

	groups, ok := result.(*metav1.APIGroupList)
	if !ok {
		return false, fmt.Errorf("Unknown type for API group %T", groups)
	}

	for _, group := range groups.Groups {
		if group.PreferredVersion.GroupVersion == supportedRBACGroupVersion {
			return true, nil
		}
	}
	return false, nil
}
