package e2e_test

import (
	"testing"

	"github.com/tellor-io/layer/e2e"
)

func TestLearn(t *testing.T) {
	e2e.LayerSpinup(t)
}
