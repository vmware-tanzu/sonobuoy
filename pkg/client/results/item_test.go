/*
Copyright the Sonobuoy contributors 2022

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
	"strings"
	"testing"

	"github.com/pkg/errors"
)

func TestItem_Walk(t *testing.T) {
	getItem := func() Item {
		return Item{
			Name:   "1",
			Status: StatusFailed,
			Items: []Item{
				{
					Name:   "2",
					Status: StatusPassed,
				},
				{
					Name:   "3",
					Status: StatusFailed,
				},
				{
					Name:   "4",
					Status: StatusPassed,
					Items: []Item{
						{
							Name:   "5",
							Status: StatusFailed,
							Items:  []Item{},
						},
					},
				},
			},
		}
	}

	sb := strings.Builder{}
	testCases := []struct {
		desc         string
		input        Item
		fn           func(*Item) error
		expectErr    error
		expectString string
	}{
		{
			desc:  "Each item is visited in the proper order",
			input: getItem(),
			fn: func(i *Item) error {
				sb.WriteString(i.Name)
				return nil
			},
			expectString: "23541",
		}, {
			desc:  "Error stops traversal",
			input: getItem(),
			fn: func(i *Item) error {
				sb.WriteString(i.Name)
				if i.Name == "3" {
					return errors.New("stop at 3")
				}
				return nil
			},
			expectErr:    errors.New("stop at 3"),
			expectString: "23",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			sb.Reset()
			err := tc.input.Walk(tc.fn)
			switch {
			case tc.expectErr == nil && err == nil:
			case tc.expectErr == nil && err != nil:
				t.Errorf("Expected nil error but got %v", err)
			case tc.expectErr != nil && err == nil:
				t.Errorf("Expected error %v but got nil", err)
			case tc.expectErr.Error() != err.Error():
				t.Errorf("Expected error %v but got %v", tc.expectErr, err)
			}

			if tc.expectString != sb.String() {
				t.Errorf("Expected %q but got %q", tc.expectString, sb.String())
			}
		})
	}
}
