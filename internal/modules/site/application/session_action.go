package application

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

var errBadSessionAction = errors.New("unsupported session action")

const maxSessionSelector = 256

func validateSessionAction(action *SessionAction) error {
	if action.Action == "reset" {
		return nil
	}
	if action.Action != "login" && action.Action != "logout" {
		return errBadSessionAction
	}
	if err := validateActionPath(action); err != nil {
		return err
	}
	if !validSelector(action.SubmitSelector) || !validSelector(action.ReadySelector) {
		return errBadSessionAction
	}
	if action.ErrorSelector != "" && !validSelector(action.ErrorSelector) {
		return errBadSessionAction
	}
	if action.Action == "login" {
		return validateLoginAction(*action)
	}
	return nil
}

func validateActionPath(action *SessionAction) error {
	if strings.ContainsAny(action.Path, "?#") {
		return domain.ErrBadPath
	}
	spec, err := (domain.CaptureSpec{Path: action.Path}).Normalize()
	if err != nil {
		return fmt.Errorf("validate session page path: %w", err)
	}
	action.Path = spec.Path
	return nil
}

func validateLoginAction(action SessionAction) error {
	if !validSelector(action.EmailSelector) || !validSelector(action.PasswordSelector) ||
		action.Email == "" || action.Password == "" {
		return errBadSessionAction
	}
	return nil
}

func validSelector(selector string) bool {
	return strings.TrimSpace(selector) != "" && len(selector) <= maxSessionSelector
}

func parseSessionStatus(output string) string {
	for _, line := range strings.Split(output, "\n") {
		payload, found := strings.CutPrefix(strings.TrimSpace(line), "SESSION_JSON:")
		if !found {
			continue
		}
		var result struct {
			Status string `json:"status"`
		}
		if json.Unmarshal([]byte(payload), &result) != nil {
			return "operation_result_unreadable"
		}
		switch result.Status {
		case "submitted", "indicator_found", "indicator_not_found", "login_failed", "not_found",
			"unsupported", "external_redirect_blocked", "operation_failed":
			return result.Status
		default:
			return "operation_result_unreadable"
		}
	}
	return "operation_result_unreadable"
}
