package bootstrap

import (
	"reflect"
	"testing"
)

func TestWorkBridgeReportsSecretRedactions(t *testing.T) {
	warnings := redactionWarnings("token=secret-value", "Safe text")
	if !reflect.DeepEqual(warnings, []string{"redacted 1 possible secrets"}) {
		t.Fatalf("warnings = %#v", warnings)
	}
}
