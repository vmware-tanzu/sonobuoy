package results

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func checkItem(expected Item, actual Item) error {
	if expected.Name != actual.Name {
		return fmt.Errorf("expected item name to be %q, got %q", expected.Name, actual.Name)
	}

	if expected.Status != actual.Status {
		return fmt.Errorf("expected item status to be %q, got %q", expected.Status, actual.Status)
	}

	for k, v := range expected.Metadata {
		if actual.Metadata[k] != v {
			return fmt.Errorf("expected item metadata %q to be %q, got %q", k, v, actual.Metadata[k])
		}
	}

	// Check that the unmarshalled Details object matches. This check will fail if the types don't match even if the values are the same
	// e.g. (string vs interface{})
	if !reflect.DeepEqual(expected.Details, actual.Details) {
		return fmt.Errorf("expected item details to be %q (%v), got %q (%v)", expected.Details, reflect.TypeOf(expected.Details), actual.Details, reflect.TypeOf(actual.Details))
	}

	if len(expected.Items) != len(actual.Items) {
		return fmt.Errorf("unexpected number of items, expected %v, got %v", len(expected.Items), len(actual.Items))
	}

	for i, item := range expected.Items {
		err := checkItem(item, actual.Items[i])
		if err != nil {
			return err
		}
	}

	return nil
}

func TestManualProcessFile(t *testing.T) {
	testCases := []struct {
		name                string
		pluginDir           string
		currentFile         string
		expectedError       string
		expectMetadataError bool
		expectedItem        Item
	}{
		{
			name:                "missing file includes error in metadata and unknown status",
			pluginDir:           "./testdata/mockResults/manualProcessing",
			currentFile:         "./testdata/mockResults/manualProcessing/missingFile.yaml",
			expectedError:       "opening file ./testdata/mockResults/manualProcessing/missingFile.yaml",
			expectMetadataError: true,
			expectedItem: Item{
				Name:   "missingFile.yaml",
				Status: StatusUnknown,
				Metadata: map[string]string{
					metadataFileKey: "missingFile.yaml",
					metadataTypeKey: metadataTypeFile,
				},
			},
		},
		{
			name:          "invalid yaml file includes error in metadata and unknown status",
			pluginDir:     "./testdata/mockResults/manualProcessing",
			currentFile:   "./testdata/mockResults/manualProcessing/invalid-results.yaml",
			expectedError: "error processing manual results",
			expectedItem: Item{
				Name:   "invalid-results.yaml",
				Status: StatusUnknown,
				Metadata: map[string]string{
					metadataFileKey: "invalid-results.yaml",
					metadataTypeKey: metadataTypeFile,
				},
			},
		},
		{
			name:        "status is taken from the manual results",
			pluginDir:   "./testdata/mockResults/manualProcessing",
			currentFile: "./testdata/mockResults/manualProcessing/manual-results.yaml",
			expectedItem: Item{
				Name:   "manual-results.yaml",
				Status: "status-from-manual-results",
				Metadata: map[string]string{
					metadataFileKey: "manual-results.yaml",
					metadataTypeKey: metadataTypeFile,
				},
				Items: []Item{
					{
						Name:   "a test file",
						Status: "status 1",
					},
					{
						Name:   "another test file",
						Status: "status 2",
					},
				},
			},
		},
		{
			name:        "result item with arbitrary details is valid",
			pluginDir:   "./testdata/mockResults/manualProcessing",
			currentFile: "./testdata/mockResults/manualProcessing/manual-results-arbitrary-details.yaml",
			expectedItem: Item{
				Name:   "manual-results-arbitrary-details.yaml",
				Status: "status-from-manual-results",
				Metadata: map[string]string{
					metadataFileKey: "manual-results-arbitrary-details.yaml",
					metadataTypeKey: metadataTypeFile,
				},
				Details: map[string]interface{}{
					"arbitrary-data": map[interface{}]interface{}{
						"key": "value",
						"array-of-integers": []interface{}{
							1,
							2,
							3,
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			item, err := manualProcessFile(tc.pluginDir, tc.currentFile)
			if tc.expectedError != "" && err == nil {
				t.Fatalf("expected error %q but err was nil", tc.expectedError)
			}
			if err != nil {
				if tc.expectedError != "" {
					if _, ok := item.Metadata["error"]; !ok && tc.expectMetadataError {
						t.Errorf("expected metadata error field to be set")
					}

					if !strings.Contains(err.Error(), tc.expectedError) {
						t.Errorf("expected error %q to contain %q", err.Error(), tc.expectedError)
					}
				} else {
					t.Errorf("unexpected error %q", err)
				}
			}

			checkErr := checkItem(tc.expectedItem, item)
			if checkErr != nil {
				t.Error(checkErr)
			}
		})
	}
}
