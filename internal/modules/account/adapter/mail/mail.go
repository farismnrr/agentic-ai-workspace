// Package mail delivers account emails over SMTP, or into the log when no server is set up.
package mail

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"strings"
)

// Log writes the message to the server log. It is the fallback for installs
// without an SMTP server, so the owner can still hand out reset links.
type Log struct{}

// SendPasswordReset logs the reset link.
func (Log) SendPasswordReset(_ context.Context, to, _, link string) error {
	slog.Info("password reset link (no SMTP server configured)", "to", to, "link", link)

	return nil
}

// SMTP sends mail through an SMTP server, using STARTTLS when the server offers it.
type SMTP struct {
	Addr     string
	User     string
	Password string
	From     string
}

// SendPasswordReset mails the reset link.
func (s SMTP) SendPasswordReset(ctx context.Context, to, name, link string) error {
	body := fmt.Sprintf("Hi %s,\r\n\r\nUse this link to choose a new password. It works for one hour.\r\n\r\n%s\r\n\r\n"+
		"If you did not ask for this, you can ignore this email.\r\n", clean(name), link)
	msg := "From: " + s.From + "\r\nTo: " + clean(to) + "\r\nSubject: Reset your password\r\n" +
		"MIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n" + body

	host, _, err := net.SplitHostPort(s.Addr)
	if err != nil {
		return fmt.Errorf("smtp address: %w", err)
	}

	var auth smtp.Auth
	if s.User != "" {
		auth = smtp.PlainAuth("", s.User, s.Password, host)
	}

	done := make(chan error, 1)

	go func() { done <- smtp.SendMail(s.Addr, auth, s.From, []string{clean(to)}, []byte(msg)) }()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("send mail: %w", err)
		}

		return nil
	case <-ctx.Done():
		return fmt.Errorf("send mail: %w", ctx.Err())
	}
}

// clean strips line breaks so a name or address can never inject mail headers.
func clean(value string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(value)
}
