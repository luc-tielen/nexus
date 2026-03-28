package nexus_test

import (
	"testing"

	"github.com/luc/nexus/internal/nexus"
)

func TestNew(t *testing.T) {
	n := nexus.New()
	if n == nil {
		t.Fatal("expected non-nil Nexus")
	}
}
