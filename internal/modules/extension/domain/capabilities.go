package domain

import "strings"

// Risk says how much trust an ability asks for.
type Risk string

// The levels of risk shown beside an ability.
const (
	RiskLow  Risk = "low"
	RiskHigh Risk = "high"
)

// Capability is one thing the extension may do, in words for a person who is not a programmer.
type Capability struct {
	Text string `json:"text"`
	Risk Risk   `json:"risk"`
}

var permissionWords = map[string]Capability{
	"storage":               {"Remember its settings in your browser", RiskLow},
	"alarms":                {"Run things on a timer", RiskLow},
	"notifications":         {"Show notifications", RiskLow},
	"contextMenus":          {"Add items to the right-click menu", RiskLow},
	"activeTab":             {"Work on the page you are looking at, only when you click it", RiskLow},
	"sidePanel":             {"Show a side panel", RiskLow},
	"clipboardWrite":        {"Put text on your clipboard", RiskLow},
	"scripting":             {"Run its own code on pages it is allowed to open", RiskLow},
	"tabs":                  {"See the address and title of your open tabs", RiskHigh},
	"cookies":               {"Read and change your cookies, which can include logins", RiskHigh},
	"history":               {"Read your browsing history", RiskHigh},
	"bookmarks":             {"Read and change your bookmarks", RiskHigh},
	"downloads":             {"Start and manage downloads", RiskHigh},
	"webRequest":            {"See the requests your browser makes", RiskHigh},
	"clipboardRead":         {"Read what is on your clipboard", RiskHigh},
	"management":            {"See and control your other extensions", RiskHigh},
	"debugger":              {"Control other pages like a developer tool", RiskHigh},
	"nativeMessaging":       {"Talk to programs on your computer", RiskHigh},
	"declarativeNetRequest": {"Block or change requests your browser makes", RiskHigh},
}

// Capabilities describes what the manifest lets the extension do, the riskiest first.
func (m Manifest) Capabilities() []Capability {
	var high, low []Capability

	add := func(c Capability) {
		if c.Risk == RiskHigh {
			high = append(high, c)
		} else {
			low = append(low, c)
		}
	}

	for _, permission := range m.Permissions {
		if words, ok := permissionWords[permission]; ok {
			add(words)
		}
	}

	for _, host := range m.HostPermissions {
		add(hostWords(host))
	}

	for _, script := range m.ContentScripts {
		for _, match := range script.Matches {
			add(hostWords(match))
		}
	}

	return unique(append(high, low...))
}

func unique(list []Capability) []Capability {
	seen := map[string]bool{}
	out := make([]Capability, 0, len(list))

	for _, item := range list {
		if !seen[item.Text] {
			seen[item.Text] = true
			out = append(out, item)
		}
	}

	return out
}

func hostWords(pattern string) Capability {
	switch pattern {
	case "<all_urls>", "*://*/*", "http://*/*", "https://*/*":
		return Capability{"Read and change everything on every website you visit", RiskHigh}
	default:
		host := strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(pattern, "*://"), "https://"), "http://")
		host, _, _ = strings.Cut(host, "/")

		return Capability{"Read and change what you see on " + strings.TrimPrefix(host, "*."), RiskHigh}
	}
}
