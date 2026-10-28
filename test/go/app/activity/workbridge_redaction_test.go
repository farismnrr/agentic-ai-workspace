package activity_test

import (
	"reflect"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/bootstrap"
)

func TestWorkBridgeReportsSecretRedactions(t *testing.T) {
	warnings := bootstrap.RedactionWarnings("token=secret-value", "Safe text")
	if !reflect.DeepEqual(warnings, []string{"redacted 1 possible secrets"}) {
		t.Fatalf("warnings = %#v", warnings)
	}
}
