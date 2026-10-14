package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/account/domain"
)

// ResetTTL is how long a password reset link works.
const (
	ResetTTL       = time.Hour
	deliverTimeout = 30 * time.Second
)

// Recovery lets people who forgot their password get back in.
type Recovery struct {
	accounts *Service
	resets   ResetStore
	mailer   Mailer
	baseURL  string
}

// NewRecovery wires the reset flow; baseURL is the public address links point to.
func NewRecovery(accounts *Service, resets ResetStore, mailer Mailer, baseURL string) *Recovery {
	return &Recovery{accounts: accounts, resets: resets, mailer: mailer, baseURL: baseURL}
}

// Request mails a reset link if the address has an account. It never says whether it does.
func (r *Recovery) Request(ctx context.Context, email string) error {
	clean, err := domain.NormalizeEmail(email)
	if err != nil {
		return nil //nolint:nilerr // an unreadable address is answered like an unknown one
	}

	user, _, err := r.accounts.users.ByEmail(ctx, clean)
	if errors.Is(err, domain.ErrUserNotFound) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("find account: %w", err)
	}

	plain, hash, err := r.accounts.tokens.New()
	if err != nil {
		return fmt.Errorf("new reset secret: %w", err)
	}

	if err := r.resets.CreateReset(ctx, hash, user.ID, r.accounts.now().Add(ResetTTL)); err != nil {
		return fmt.Errorf("store reset: %w", err)
	}

	// Sending happens off the request so the answer takes the same time for every address.
	go r.deliver(context.WithoutCancel(ctx), user, r.baseURL+"/reset?token="+plain)

	return nil
}

func (r *Recovery) deliver(ctx context.Context, user domain.User, link string) {
	ctx, cancel := context.WithTimeout(ctx, deliverTimeout)
	defer cancel()

	if err := r.mailer.SendPasswordReset(ctx, user.Email, user.Name, link); err != nil {
		slog.Error("could not send password reset link", "error", err)
	}
}

// Reset sets a new password with a link's secret and signs the person out everywhere.
func (r *Recovery) Reset(ctx context.Context, token, password string) error {
	if err := domain.ValidatePassword(password); err != nil {
		return fmt.Errorf("reset password: %w", err)
	}

	userID, err := r.resets.ConsumeReset(ctx, r.accounts.tokens.Hash(token), r.accounts.now())
	if err != nil {
		return domain.ErrInvalidResetToken
	}

	return r.accounts.replacePassword(ctx, userID, password, "")
}
