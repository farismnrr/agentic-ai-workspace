package agentsession

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestScopeIsolationAndCapacity(t *testing.T) {
	registry := testRegistry(t, 2, time.Minute, time.Hour, time.Second, time.Now)
	url := sessionURL(t)
	first, second := sessionScope("u1", "t1"), sessionScope("u2", "t1")
	setCookie(t, registry, first, url, "one")
	setCookie(t, registry, second, url, "two")
	assertCookie(t, registry, first, url, "one")
	assertCookie(t, registry, second, url, "two")
	if err := registry.Do(t.Context(), sessionScope("u3", "t1"), func(context.Context, http.CookieJar) error {
		return nil
	}); !errors.Is(err, ErrFull) {
		t.Fatalf("capacity error = %v", err)
	}
}

func TestIdleAndAbsoluteExpiry(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	registry := testRegistry(t, 2, 30*time.Minute, 4*time.Hour, time.Second, clock)
	scope, target := sessionScope("u", "t"), sessionURL(t)
	setCookie(t, registry, scope, target, "active")
	now = now.Add(31 * time.Minute)
	assertNoCookie(t, registry, scope, target)
	setCookie(t, registry, scope, target, "new")
	now = now.Add(4 * time.Hour)
	assertNoCookie(t, registry, scope, target)
}

func TestCookieJarHasPerSessionLimits(t *testing.T) {
	registry := testRegistry(t, 1, time.Hour, 2*time.Hour, time.Second, time.Now)
	scope, target := sessionScope("u", "t"), sessionURL(t)
	err := registry.Do(t.Context(), scope, func(_ context.Context, jar http.CookieJar) error {
		cookies := make([]*http.Cookie, 0, maxCookiesPerSession+1)
		for i := range maxCookiesPerSession + 1 {
			cookies = append(cookies, &http.Cookie{
				Name: fmt.Sprintf("c%d", i), Value: "v", Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode,
			})
		}
		cookies = append(cookies, &http.Cookie{
			Name: "large", Value: strings.Repeat("x", maxCookieBytes+1),
			Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode,
		})
		jar.SetCookies(target, cookies)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	assertCookieCount(t, registry, scope, target, maxCookiesPerSession)
}

func TestPerUserCapacityDoesNotBlockOtherUsers(t *testing.T) {
	registry := testRegistryWithUserLimit(t, 3, 2)
	for _, task := range []string{"t1", "t2"} {
		if err := registry.Do(t.Context(), sessionScope("u1", task), func(context.Context, http.CookieJar) error {
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := registry.Do(t.Context(), sessionScope("u1", "t3"), func(context.Context, http.CookieJar) error {
		return nil
	}); !errors.Is(err, ErrFull) {
		t.Fatalf("third session for one user error = %v", err)
	}
	if err := registry.Do(t.Context(), sessionScope("u2", "t1"), func(context.Context, http.CookieJar) error {
		return nil
	}); err != nil {
		t.Fatalf("another user was blocked: %v", err)
	}
}

func TestOperationsSerializeAndRevocationCancels(t *testing.T) {
	registry := testRegistry(t, 2, time.Hour, 2*time.Hour, time.Second, time.Now)
	scope := sessionScope("u", "t")
	started, release := make(chan struct{}), make(chan struct{})
	var active, maximum atomic.Int32
	first := make(chan error, 1)
	go func() {
		first <- registry.Do(context.Background(), scope, func(ctx context.Context, _ http.CookieJar) error {
			close(started)
			active.Add(1)
			defer active.Add(-1)
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}()
	<-started
	second := make(chan error, 1)
	go func() {
		second <- registry.Do(context.Background(), scope, func(context.Context, http.CookieJar) error {
			current := active.Add(1)
			defer active.Add(-1)
			maximum.Store(max(maximum.Load(), current))
			return nil
		})
	}()
	time.Sleep(10 * time.Millisecond)
	close(release)
	if err := <-first; err != nil {
		t.Fatalf("first operation error = %v", err)
	}
	if err := <-second; err != nil {
		t.Fatalf("second operation error = %v", err)
	}
	if maximum.Load() != 1 {
		t.Fatalf("maximum concurrent operations = %d, want 1", maximum.Load())
	}

	started = make(chan struct{})
	first = make(chan error, 1)
	go func() {
		first <- registry.Do(context.Background(), scope, func(ctx context.Context, _ http.CookieJar) error {
			close(started)
			<-ctx.Done()
			return ctx.Err()
		})
	}()
	<-started
	registry.RevokeTask(scope.UserID, scope.TaskID)
	if err := <-first; !errors.Is(err, context.Canceled) {
		t.Fatalf("revoked operation error = %v", err)
	}
}

func TestResetDropsOnlyTheExactSessionScope(t *testing.T) {
	registry := testRegistry(t, 2, time.Hour, 2*time.Hour, time.Second, time.Now)
	target := sessionURL(t)
	first, second := sessionScope("u1", "t1"), sessionScope("u1", "t2")
	setCookie(t, registry, first, target, "one")
	setCookie(t, registry, second, target, "two")
	registry.Reset(first)
	assertNoCookie(t, registry, first, target)
	assertCookie(t, registry, second, target, "two")
}

func TestOperationTimeoutAndClose(t *testing.T) {
	registry := testRegistry(t, 1, time.Hour, 2*time.Hour, 20*time.Millisecond, time.Now)
	scope := sessionScope("u", "t")
	err := registry.Do(context.Background(), scope, func(ctx context.Context, _ http.CookieJar) error {
		<-ctx.Done()
		return ctx.Err()
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timeout error = %v", err)
	}
	registry.Close()
	err = registry.Do(context.Background(), scope, func(context.Context, http.CookieJar) error { return nil })
	if !errors.Is(err, ErrClosed) {
		t.Fatalf("closed registry error = %v", err)
	}
}
