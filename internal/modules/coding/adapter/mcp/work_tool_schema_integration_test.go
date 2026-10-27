package mcp

import (
	"encoding/json"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func assertBeginWorkRoleSchema(t *testing.T, session *sdk.ClientSession) {
	t.Helper()
	listed, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range listed.Tools {
		if tool.Name != "begin_work" {
			continue
		}
		assertRoleRequired(t, tool.InputSchema)
		return
	}
	t.Fatal("begin_work was absent from MCP tools/list")
}

func assertRoleRequired(t *testing.T, inputSchema any) {
	t.Helper()
	encoded, err := json.Marshal(inputSchema)
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(encoded, &schema); err != nil {
		t.Fatal(err)
	}
	for _, field := range schema.Required {
		if field == "role" {
			return
		}
	}
	t.Fatalf("begin_work role is not required in MCP schema: %s", encoded)
}
