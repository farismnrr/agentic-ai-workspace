package molab_test

import (
	"strings"
	"testing"

	molab "github.com/farismnrr/agentic-ai-workspace/internal/modules/molab/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func request(text string) taskdomain.Event {
	return taskdomain.Event{Kind: taskdomain.EventRequest, Text: text}
}

// Pure chat interface: no special task creation, pairing API or token store.
func TestMoLabReadsLatestUserURLAndTokenFromSameChatMessage(t *testing.T) {
	old := request("https://sb-old.sb.molab.run/ --token old-synthetic-token-123")
	newRequest := request("Connect to https://sb-new.sb.molab.run/ and run code with --token new-synthetic-token-456")
	got, err := molab.FromChat([]taskdomain.Event{old, {Kind: taskdomain.EventMessage, Text: "assistant text"}, newRequest})
	if err != nil {
		t.Fatal(err)
	}
	if got.Host != "sb-new.sb.molab.run" || got.URL != "https://sb-new.sb.molab.run/" || got.Token != "new-synthetic-token-456" {
		t.Fatalf("wrong credentials selected: %q %q", got.Host, got.URL)
	}
}

func TestMoLabNeverCombinesDifferentUserMessages(t *testing.T) {
	events := []taskdomain.Event{request("MoLab URL https://sb-new.sb.molab.run/"), request("new token: new-synthetic-token-456")}
	if _, err := molab.FromChat(events); err == nil {
		t.Fatal("combined credentials from separate messages")
	}
	if _, err := molab.FromChat([]taskdomain.Event{request("Only URL https://sb-a.sb.molab.run/")}); err == nil {
		t.Fatal("URL without token accepted")
	}
}

func TestMoLabRejectsUnsafeOrMalformedURLs(t *testing.T) {
	inputs := []string{
		"http://sb-a.sb.molab.run/",
		"https://sb-a.sb.molab.run.evil.com/",
		"https://sb-a.sb.molab.run/other",
		"https://evil.test@sb-a.sb.molab.run/",
		"https://sb-a.sb.molab.run?redirect=evil.test",
		"https://localhost/",
		"https://sb-a.sb.molab.run:444/",
	}
	for _, url := range inputs {
		_, err := molab.FromChat([]taskdomain.Event{request("URL " + url + " --token synthetic-token-123")})
		if err == nil {
			t.Errorf("unsafe URL accepted: %s", url)
		}
	}
	if _, err := molab.FromChat([]taskdomain.Event{request("URL https://sb-safe.sb.molab.run/ --token synthetic-token-123")}); err != nil {
		t.Fatal(err)
	}
	if _, err := molab.FromChat([]taskdomain.Event{request("URL https://sb-safe.sb.molab.run/ --token short")}); err == nil {
		t.Fatal("too short token accepted")
	}
	if strings.Contains(molab.ErrMissingChatCredentials.Error(), "synthetic-token-123") {
		t.Fatal("error includes token")
	}
}
