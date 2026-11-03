package mcp

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image/png"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const maxBrowserScreenshot = 5 << 20

type screenshotRef struct {
	Ref    string `json:"screenshot_ref"`
	MIME   string `json:"mime_type"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Bytes  int    `json:"bytes"`
}

type screenshotInput struct {
	WorkTarget
	Ref string `json:"screenshot_ref" jsonschema:"reference returned by browser_test, at most 60 characters"`
}

type screenshotOutput struct {
	screenshotRef
	PNGBase64 string `json:"png_base64"`
}

func (t *tools) readBrowserScreenshotWithImage(
	ctx context.Context, request *sdk.CallToolRequest, in screenshotInput,
) (*sdk.CallToolResult, screenshotOutput, error) {
	handler := guarded(t.guard, domain.ToolReadBrowserScreenshot, func(input screenshotInput) string {
		return "read browser screenshot " + input.Ref
	}, t.readBrowserScreenshot)
	result, output, err := handler(ctx, request, in)
	if err != nil || result != nil || output.PNGBase64 == "" {
		return result, output, err
	}
	pngBytes, err := base64.StdEncoding.DecodeString(output.PNGBase64)
	if err != nil {
		return failure("The saved screenshot could not be decoded.", nil), screenshotOutput{}, nil
	}
	message := fmt.Sprintf("Retrieved screenshot_ref %s (%dx%d, %d bytes).", output.Ref, output.Width, output.Height, output.Bytes)
	return &sdk.CallToolResult{Content: []sdk.Content{
		&sdk.TextContent{Text: message},
		&sdk.ImageContent{Data: pngBytes, MIMEType: "image/png"},
	}}, output, nil
}

func (t *tools) readBrowserScreenshot(
	ctx context.Context, in screenshotInput,
) (screenshotOutput, report, error) {
	if t.shots == nil || t.works == nil {
		return screenshotOutput{}, report{}, domain.ErrBrowserUnavailable
	}
	svc, err := t.services(ctx)
	if err != nil {
		return screenshotOutput{}, report{}, err
	}
	who, _ := ctx.Value(identityKey{}).(identity)
	if err = t.works.RequireRole(ctx, who.User, who.Client, svc.ProjectID, in.WorkID, "qa"); err != nil {
		return screenshotOutput{}, report{}, workToolError("require QA role", err)
	}
	state, err := t.works.State(ctx, who.User, who.Client, svc.ProjectID, in.WorkID)
	if err != nil || state.TaskID == "" {
		return screenshotOutput{}, report{}, workToolError("resolve active work task", taskdomain.ErrUnknownWork)
	}
	storedName, version, valid := taskdomain.ParseScreenshotReference(in.Ref, in.WorkID)
	if !valid {
		return screenshotOutput{}, report{}, domain.ErrBrowserShotUnavailable
	}
	runner, ok := svc.Sites.(BrowserRunner)
	if !ok {
		return screenshotOutput{}, report{}, domain.ErrBrowserUnavailable
	}
	currentVersion, err := runner.SiteVersion(ctx, who.User)
	if err != nil || currentVersion != version {
		return screenshotOutput{}, report{}, domain.ErrBrowserShotUnavailable
	}
	pngBytes, err := t.shots.Load(ctx, state.TaskID, storedName)
	if err != nil || len(pngBytes) == 0 || len(pngBytes) > maxBrowserScreenshot {
		return screenshotOutput{}, report{}, domain.ErrBrowserShotUnavailable
	}
	config, err := png.DecodeConfig(bytes.NewReader(pngBytes))
	if err != nil || config.Width <= 0 || config.Height <= 0 {
		return screenshotOutput{}, report{}, domain.ErrBrowserShotUnavailable
	}
	return screenshotOutput{
		screenshotRef: screenshotRef{
			Ref: in.Ref, MIME: "image/png", Width: config.Width, Height: config.Height, Bytes: len(pngBytes),
		},
		PNGBase64: base64.StdEncoding.EncodeToString(pngBytes),
	}, report{detail: "browser screenshot retrieved"}, nil
}
