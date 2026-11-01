package application

import (
	"encoding/json"
	"strings"
)

func writeTargets(call Call) ([]string, bool) {
	switch call.Name {
	case "write_file", "edit_file", "delete_file":
		var args struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(call.Input, &args); err != nil {
			return nil, false
		}
		return []string{args.Path}, true
	case "download_asset":
		var args struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(call.Input, &args); err != nil || strings.TrimSpace(args.Path) == "" {
			// A default/generated asset path is unknown until download completes.
			// Scoped parallel writers must supply a concrete target.
			return []string{""}, true
		}
		return []string{args.Path}, true
	case "move_file":
		var args struct {
			From string `json:"from"`
			To   string `json:"to"`
		}
		if err := json.Unmarshal(call.Input, &args); err != nil {
			return nil, false
		}
		return []string{args.From, args.To}, true
	default:
		return nil, false
	}
}
