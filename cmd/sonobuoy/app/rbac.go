package app

import (
	"fmt"

	"github.com/pkg/errors"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// RBACMode determines whether to enable or disable RBAC for a Sonobuoy run
type RBACMode string

var (
	RBACErrorNoClient = errors.New("can't use nil client with \"detect\" RBAC mode")
)

const (
	// DisableRBACMode means rbac is always disable
	DisableRBACMode RBACMode = "disable"
	// EnabledRBACMode means rbac is always enabled
	EnabledRBACMode RBACMode = "enabled"
	// DetectRBACMode means "query the server to see if RBAC is enabled"
	DetectRBACMode RBACMode = "detect"
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
	mode, ok := rbacModeMap[str]
	if !ok {
		return fmt.Errorf("unknown RBAC mode %s", str)
	}
	*r = mode
	return nil
}

const apiRootPath = "/apis"

var supportedRBACGroupVersion = rbacv1.SchemeGroupVersion.String()

// Get retrieves whether to enable or disable rbac. If the mode is disable or
// enabled, the client is unused. If the mode is "detect", the client will be
// used to query the server's API groups and detect whether an RBAC api group exists.
func (r *RBACMode) Get(client *kubernetes.Clientset) (bool, error) {
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
			return false, RBACErrorNoClient
		}
		return checkRBACEnabled(client.Discovery().RESTClient())
	default:
		return false, fmt.Errorf("Unknown RBAC mode %v", *r)
	}
}

func checkRBACEnabled(client rest.Interface) (bool, error) {
	result, err := client.Get().AbsPath(apiRootPath).Do().Get()
	if err != nil {
		return false, errors.Wrap(err, "couldn't retrieve API groups")
	}

	groups, ok := result.(*metav1.APIGroupList)
	if !ok {
		return false, fmt.Errorf("Unknown type for API group %t", groups)
	}

	for _, group := range groups.Groups {
		if group.PreferredVersion.GroupVersion == supportedRBACGroupVersion {
			return true, nil
		}
	}
	return false, nil
}
