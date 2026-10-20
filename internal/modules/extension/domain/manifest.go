package domain

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// Limits the browsers' stores set, kept here so an extension that would be refused is fixed before it is packaged.
const (
	maxName        = 45
	maxDescription = 132
	// supportedManifest is the only manifest version Chrome and Edge still accept.
	supportedManifest = 3
)

var versionPattern = regexp.MustCompile(`^\d{1,5}(\.\d{1,5}){0,3}$`)

// Manifest is the part of manifest.json that the checks and the install page need.
type Manifest struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Description     string            `json:"description"`
	Icons           map[string]string `json:"icons"`
	OptionsPage     string            `json:"options_page"`
	CSP             json.RawMessage   `json:"content_security_policy"`
	Action          Action            `json:"action"`
	OptionsUI       OptionsUI         `json:"options_ui"`
	Background      Background        `json:"background"`
	Permissions     []string          `json:"permissions"`
	HostPermissions []string          `json:"host_permissions"`
	ContentScripts  []ContentScript   `json:"content_scripts"`
	ManifestVersion int               `json:"manifest_version"`
}

// Action is the toolbar button.
type Action struct {
	DefaultPopup string          `json:"default_popup"`
	DefaultIcon  json.RawMessage `json:"default_icon"`
}

// OptionsUI is the options page opened by the browser.
type OptionsUI struct {
	Page string `json:"page"`
}

// Background is the service worker.
type Background struct {
	ServiceWorker string `json:"service_worker"`
}

// ContentScript runs inside the pages it matches.
type ContentScript struct {
	Matches []string `json:"matches"`
	JS      []string `json:"js"`
	CSS     []string `json:"css"`
}

// ParseManifest reads manifest.json.
func ParseManifest(data []byte) (Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("manifest.json is not valid JSON: %w", err)
	}

	return m, nil
}

// Referenced lists the files the manifest points at, so a missing one is found before the person installs.
func (m Manifest) Referenced() []string {
	files := []string{m.Background.ServiceWorker, m.Action.DefaultPopup, m.OptionsPage, m.OptionsUI.Page}

	for _, icon := range m.Icons {
		files = append(files, icon)
	}

	for _, script := range m.ContentScripts {
		files = append(files, script.JS...)
		files = append(files, script.CSS...)
	}

	out := make([]string, 0, len(files))

	for _, file := range files {
		if file = strings.TrimPrefix(strings.TrimSpace(file), "/"); file != "" {
			out = append(out, file)
		}
	}

	return out
}

// Problems lists what must be fixed before the extension can be installed, each as an instruction to the builder.
func (m Manifest) Problems() []string {
	var found []string

	if m.ManifestVersion != supportedManifest {
		found = append(found, "manifest_version must be 3 (Chrome and Edge no longer take version 2)")
	}

	if strings.TrimSpace(m.Name) == "" || utf8.RuneCountInString(m.Name) > maxName {
		found = append(found, fmt.Sprintf("name is required and at most %d characters", maxName))
	}

	if !versionPattern.MatchString(m.Version) {
		found = append(found, `version must be numbers separated by dots, such as "1.0.0"`)
	}

	if utf8.RuneCountInString(m.Description) > maxDescription {
		found = append(found, fmt.Sprintf("description is at most %d characters", maxDescription))
	}

	if csp := string(m.CSP); strings.Contains(csp, "unsafe-eval") || strings.Contains(csp, "http:") ||
		regexp.MustCompile(`script-src[^;"]*https?://`).MatchString(csp) {
		found = append(found, "content_security_policy may not allow eval or scripts loaded from the web: "+
			"the code must be in the package")
	}

	return found
}
