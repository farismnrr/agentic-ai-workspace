package application

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// TicketTTL is how long a ticket works after the owner's visit.
const TicketTTL = 4 * time.Hour

const macBytes = 16

// Ticket returns a signed, expiring pass for one site address. The owner's page visit puts it in
// the address, so what the sandboxed page loads next is allowed without the person's login.
// It is bound to the token, so it dies when the site is started again, to the holder (the
// visitor's browser and, when the server can see it, network address), so a copied address does
// not work from another machine, and the key is renewed on every server start, which also
// stops every site.
func (s *Service) Ticket(token, holder string) string {
	expires := s.deps.Now().Add(TicketTTL).Unix()

	return fmt.Sprintf("%d.%s", expires, s.mac(token, holder, expires))
}

// ValidTicket checks a ticket against a site address token and the holder presenting it.
func (s *Service) ValidTicket(token, holder, ticket string) bool {
	text, mac, found := strings.Cut(ticket, ".")
	if !found {
		return false
	}

	expires, err := strconv.ParseInt(text, 10, 64)
	if err != nil || s.deps.Now().Unix() > expires {
		return false
	}

	return hmac.Equal([]byte(mac), []byte(s.mac(token, holder, expires)))
}

func (s *Service) mac(token, holder string, expires int64) string {
	h := hmac.New(sha256.New, s.deps.Secret)
	_, _ = fmt.Fprintf(h, "site-ticket|%s|%q|%d", token, holder, expires)

	return hex.EncodeToString(h.Sum(nil)[:macBytes])
}
