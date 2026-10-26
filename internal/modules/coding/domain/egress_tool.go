package domain

// ToolRequestEgress lets an agent ask for one more HTTPS host for its own task. It is not in Tools(): the coding
// space that outside AI apps use never offers it, and it is on only inside the app's own task loop when the server
// has egress grants enabled. It follows run_command (see SwitchOf), because a host is only needed to run something.
// It can only ask. A person decides, in the app.
var ToolRequestEgress = ToolInfo{
	Name: "request_egress", Title: "Ask for a website", Scope: ScopeExec, Group: GroupExec,
	Description: "Ask the person to let this task reach one more website from the sandbox, when a command fails " +
		"because the site is blocked. Give only the host name (for example example.com) and say why it is needed. " +
		"It stays blocked until the person approves; you cannot approve it, and asking again for the same site " +
		"just returns the same answer. Allowed sites last about a day and only for this task.",
}
