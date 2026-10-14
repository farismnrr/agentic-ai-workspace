package bootstrap

import (
	"database/sql"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/account/adapter/argon2"
	accounthttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/account/adapter/http"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/account/adapter/mail"
	accountsqlite "github.com/farismnrr/agentic-ai-workspace/internal/modules/account/adapter/sqlite"
	accountapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/account/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/config"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/opaque"
)

func newAccount(
	db *sql.DB, cfg config.Config, sameOrigin fiber.Handler, limits accounthttp.Limits,
) (*accountapp.Service, httpserver.Mount) {
	store := accountsqlite.NewStore(db)
	service := accountapp.NewService(store, store, argon2.Hasher{}, opaque.Generator{}, time.Now)

	recovery := accountapp.NewRecovery(service, store, newMailer(cfg), cfg.PublicURL)

	return service, accounthttp.NewHandler(service, recovery, cfg.CookieSecure).Mount(sameOrigin, limits)
}

// newMailer sends through SMTP when SMTP_ADDR is set, otherwise into the log.
func newMailer(cfg config.Config) accountapp.Mailer {
	if cfg.SMTPAddr == "" {
		return mail.Log{}
	}

	return mail.SMTP{Addr: cfg.SMTPAddr, User: cfg.SMTPUser, Password: cfg.SMTPPassword, From: cfg.SMTPFrom}
}
