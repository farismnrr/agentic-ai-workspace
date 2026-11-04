package application

import (
    "strings"
    "testing"
)

func TestRolePreflightSOPAndSitePrerequisites(t *testing.T) {
    cases := []struct {
        name, role string
        enabled []string
        required []string
    }{
        {
            name: "developer", role: "developer",
            enabled: []string{"read_file", "edit_file", "git"},
            required: []string{"Developer intentionally has NO unrestricted run_command shell", "Delegate named builds", "complete_task"},
        },
        {
            name: "devops enabled", role: "devops",
            enabled: []string{"run_command", "run_build", "start_site"},
            required: []string{"DevOps has the server run_command tool", "run_build", "start_site"},
        },
        {
            name: "devops disabled", role: "devops",
            enabled: []string{"read_file", "git"},
            required: []string{"DevOps server run_command is NOT enabled", "owner switch"},
        },
        {
            name: "qa", role: "qa",
            enabled: []string{"browser_test", "read_browser_screenshot"},
            required: []string{"before calling browser_test", "start_site", "NOT proof the browser is broken"},
        },
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            prompt := preActionToolSOP(tc.role, tc.enabled)
            for _, want := range tc.required {
                if !strings.Contains(prompt, want) {
                    t.Errorf("pre-action SOP missing %q", want)
                }
            }
        })
    }
    for _, want := range []string{"DevOps to build", "start_site", "complete_task", "Developer is intentionally denied run_command"} {
        if !strings.Contains(preActionLeadSOP, want) {
            t.Errorf("lead pre-action SOP missing %q", want)
        }
    }
}
