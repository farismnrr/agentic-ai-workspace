package application_test

import (
	"strings"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/application"
)

func ticketService(now *time.Time, secret string) *application.Service {
	return application.NewService(application.Deps{
		Now: func() time.Time { return *now }, Secret: []byte(secret),
	})
}

func TestTicketsAreBoundToTheSiteAndExpire(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	svc := ticketService(&now, "secret-a")
	ticket := svc.Ticket("tok-1")

	if !svc.ValidTicket("tok-1", ticket) {
		t.Fatal("a fresh ticket must work")
	}

	if svc.ValidTicket("tok-2", ticket) {
		t.Error("a ticket must not work for another address")
	}

	if ticketService(&now, "secret-b").ValidTicket("tok-1", ticket) {
		t.Error("a ticket signed with another key must not work (a restart renews the key)")
	}

	now = now.Add(application.TicketTTL + time.Second)

	if svc.ValidTicket("tok-1", ticket) {
		t.Error("an expired ticket must not work")
	}
}

func TestMalformedAndTamperedTicketsAreRefused(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	svc := ticketService(&now, "secret-a")
	good := svc.Ticket("tok")
	expires, mac, _ := strings.Cut(good, ".")

	for name, bad := range map[string]string{
		"empty":         "",
		"no dot":        "123",
		"not a number":  "x." + mac,
		"empty mac":     expires + ".",
		"tampered mac":  expires + "." + strings.Repeat("0", len(mac)),
		"later expiry":  "9999999999." + mac,
		"extra segment": good + ".extra",
	} {
		if svc.ValidTicket("tok", bad) {
			t.Errorf("%s must be refused", name)
		}
	}
}
