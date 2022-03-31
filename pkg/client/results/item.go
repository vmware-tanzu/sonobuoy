/*
Copyright the Sonobuoy contributors 2020

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
	"strings"
)

const (
	// StatusFailed is the key we base junit pass/failure off of and save into
	// our canonical results format.
	StatusFailed = "failed"

	// StatusPassed is the key we base junit pass/failure off of and save into
	// our canonical results format.
	StatusPassed = "passed"

	// StatusSkipped is the key we base junit pass/failure off of and save into
	// our canonical results format.
	StatusSkipped = "skipped"

	// StatusUnknown is the key we fallback to in our canonical results format
	// if another can not be determined.
	StatusUnknown = "unknown"

	// StatusTimeout is the key used when the plugin does not report results within the
	// timeout period. It will be treated as a failure (e.g. its parent will be marked
	// as a failure).
	StatusTimeout = "timeout"

	// PostProcessedResultsFile is the name of the file we create when doing
	// postprocessing on the plugin results.
	PostProcessedResultsFile = "sonobuoy_results.yaml"

	// MetadataFileKey is the key used in an Item's metadata field when the Item is
	// representing the a file summary (and its leaf nodes are individual tests or Suites).
	MetadataFileKey = "file"

	// MetadataTypeKey is the key used in an Item's metadata field when describing what type
	// of entry in the tree it is. Currently we just tag summaries, files, and nodes.
	MetadataTypeKey = "type"

	MetadataTypeNode    = "node"
	MetadataTypeFile    = "file"
	MetadataTypeSummary = "summary"

	MetadataDetailsFailure = "failure"
	MetadataDetailsOutput  = "output"
)

// Item is the central format for plugin results. Various plugin
// types can be transformed into this simple format and set at a standard
// location in our results tarball for simplified processing by any consumer.
type Item struct {
	Name     string                 `json:"name" yaml:"name"`
	Status   string                 `json:"status" yaml:"status"`
	Metadata map[string]string      `json:"meta,omitempty" yaml:"meta,omitempty"`
	Details  map[string]interface{} `json:"details,omitempty" yaml:"details,omitempty"`
	Items    []Item                 `json:"items,omitempty" yaml:"items,omitempty"`
}

// Empty returns true if the Item is empty.
func (i Item) Empty() bool {
	if i.Name == "" && i.Status == "" && len(i.Items) == 0 && len(i.Metadata) == 0 {
		return true
	}
	return false
}

// GetSubTreeByName traverses the tree and returns a reference to the
// subtree whose root has the given name.
func (i *Item) GetSubTreeByName(root string) *Item {
	if i == nil {
		return nil
	}

	if root == "" || i.Name == root {
		return i
	}

	if len(i.Items) > 0 {
		for _, v := range i.Items {
			subItem := (&v).GetSubTreeByName(root)
			if subItem != nil {
				return subItem
			}
		}
	}

	return nil
}

// IsTimeoutErr is the snippet of logic that determines whether or not a given Item represents
// a timeout error (i.e. Sonobuoy timed out waiting for results).
func IsTimeoutErr(i Item) bool {
	return strings.Contains(fmt.Sprint(i.Details["error"]), "timeout")
}

// IsLeaf returns true if the item has no sub-items (children). Typically
// refers to individual tests whereas non-leaf nodes more commonly refer to
// suites, nodes, or files rolled up from the individual tests.
func (i *Item) IsLeaf() bool {
	return len(i.Items) == 0
}

// Walk will do a depth-first traversal of the Item tree calling fn on each
// item. If an error is returned, traversal will stop.
func (i *Item) Walk(fn func(*Item) error) error {
	for subIndex := range i.Items {
		if err := i.Items[subIndex].Walk(fn); err != nil {
			return err
		}
	}
	return fn(i)
}
