package mcp

import (
	"context"
	"errors"
	"fmt"
	"slices"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

const maxSummary = 200

// clientKey is the token field that carries the connected app's name.
const clientKey = "client"

var errNoIdentity = errors.New("sign in to use this tool")

// report is what a tool run tells the guard about the call.
type report struct {
	detail string
	// project is where an undoable change was made.
	project string
	undo    []byte
	failed  bool
}

type guard struct {
	gate        Gate
	recorder    Recorder
	metadataURL string
}

// guarded wraps a tool body with the checks every call must pass, in order:
// the person's switch for the tool, then the token's scope, then the call
// itself. Every call is recorded, whatever its outcome.
func guarded[In, Out any](
	g *guard, info domain.ToolInfo, describe func(In) string, run func(context.Context, In) (Out, report, error),
) sdk.ToolHandlerFor[In, Out] {
	return func(ctx context.Context, req *sdk.CallToolRequest, in In) (*sdk.CallToolResult, Out, error) {
		var none Out

		token := req.GetExtra().TokenInfo
		if token == nil || token.UserID == "" {
			return nil, none, errNoIdentity
		}

		client, _ := token.Extra[clientKey].(string)
		entry := Entry{UserID: token.UserID, Client: client, Tool: info.Name, Summary: clip(describe(in))}

		if refusal := g.refuse(ctx, info, token.Scopes, &entry); refusal != nil {
			g.record(ctx, entry)

			return refusal, none, nil
		}

		out, rep, err := run(context.WithValue(ctx, identityKey{}, identity{User: token.UserID, Client: client}), in)
		if err != nil {
			entry.Outcome, entry.Detail = OutcomeFailed, UserFacing(err).Error()
			g.record(ctx, entry)

			return nil, none, UserFacing(err)
		}

		entry.Outcome, entry.Detail, entry.Undo, entry.Project = OutcomeOK, rep.detail, rep.undo, rep.project
		if rep.failed {
			entry.Outcome = OutcomeFailed
		}

		g.record(ctx, entry)

		return nil, out, nil
	}
}

// refuse returns a tool error result when the call must not run, filling in the log entry.
func (g *guard) refuse(ctx context.Context, info domain.ToolInfo, scopes []string, entry *Entry) *sdk.CallToolResult {
	allowed, err := g.gate.Allowed(ctx, entry.UserID, info.Name)
	if err != nil {
		entry.Outcome, entry.Detail = OutcomeFailed, "could not check the tool switch"

		return failure("The tool switch could not be checked. Try again in a moment.", nil)
	}

	if !allowed {
		entry.Outcome, entry.Detail = OutcomeOff, "tool is turned off"

		return failure("This tool is turned off. Ask the person to turn it on in Settings, Spaces & tools.", nil)
	}

	if !slices.Contains(scopes, info.Scope) {
		entry.Outcome, entry.Detail = OutcomeForbidden, "missing permission "+info.Scope

		return failure("This tool needs the "+info.Scope+" permission. Reconnect and approve it.", g.challenge(info.Scope))
	}

	return nil
}

// challenge is the per-tool WWW-Authenticate value ChatGPT uses to start step-up sign-in.
func (g *guard) challenge(scope string) sdk.Meta {
	header := fmt.Sprintf(`Bearer error="insufficient_scope", scope=%q, resource_metadata=%q, `+
		`error_description="This tool needs the %s permission."`, scope, g.metadataURL, scope)

	return sdk.Meta{"mcp/www_authenticate": []string{header}}
}

func failure(message string, meta sdk.Meta) *sdk.CallToolResult {
	return &sdk.CallToolResult{IsError: true, Content: []sdk.Content{&sdk.TextContent{Text: message}}, Meta: meta}
}

func (g *guard) record(ctx context.Context, entry Entry) {
	// A log failure must never turn a finished call into an error.
	_ = g.recorder.Record(ctx, entry)
}

func clip(text string) string {
	runes := []rune(text)
	if len(runes) <= maxSummary {
		return text
	}

	return string(runes[:maxSummary]) + "…"
}
