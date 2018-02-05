package image

import (
	"testing"
)

const (
	badImageName  = "gcr.IO/HEPTIOIMAGES/SONOBUOY:master"
	goodImageName = "gcr.io/heptio-images/sonobuoy:master"
)

func TestSet(t *testing.T) {
	var id ID
	if err := id.Set(badImageName); err == nil {
		t.Errorf(
			"expected %v to fail with uppercase letters but got %v",
			badImageName,
			err,
		)
	}
	if err := id.Set(goodImageName); err != nil {
		t.Errorf(
			"expected %v to be succeed, but failed with %v",
			goodImageName,
			err,
		)
	}

	if str := id.String(); str != goodImageName {
		t.Errorf("expected id to be %v, got %v", goodImageName, str)
	}
}
