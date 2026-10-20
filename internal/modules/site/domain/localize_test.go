package domain_test

import (
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

const (
	base   = "/run/shop"
	ticket = "/run/shop/~t1"
)

func localize(kind, text string) string {
	return string(domain.Localize(kind, "", base, ticket, []byte(text)))
}

func TestLocalizeMovesRootLinksOfAPageUnderTheAddress(t *testing.T) {
	got := localize("text/html; charset=utf-8", `<html><head><link rel="stylesheet" href="/_next/a.css"></head><body>`+
		`<a href='/login'>x</a><img src="/assets/l.webp" srcset="/a.png 1x, /b.png 2x"><form action="/api/x"></form>`+
		`<div style="background:url(/bg.png)"></div></body></html>`)

	for _, want := range []string{
		`href="/run/shop/~t1/_next/a.css"`, `href='/run/shop/~t1/login'`, `src="/run/shop/~t1/assets/l.webp"`,
		`srcset="/run/shop/~t1/a.png 1x, /run/shop/~t1/b.png 2x"`, `action="/run/shop/~t1/api/x"`,
		`url(/run/shop/~t1/bg.png)`, `<head><script data-site-shim>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in\n%s", want, got)
		}
	}
}

func TestLocalizeLeavesWhatIsAlreadyRightAlone(t *testing.T) {
	in := `<head></head><a href="./x">1</a><a href="../y">2</a><a href="https://a.test/z">3</a>` +
		`<a href="//cdn.test/z">4</a><a href="#top">5</a><a href="/run/shop/~t1/ok">6</a><a href="/run/shop/~t1">7</a>`
	got := localize("text/html", in)

	if strings.Contains(strings.Replace(got, `<script data-site-shim>`, "", 1), "~t1/~t1") ||
		!strings.Contains(got, `href="/run/shop/~t1/ok"`) || !strings.Contains(got, `href="//cdn.test/z"`) ||
		!strings.Contains(got, `href="./x"`) || !strings.Contains(got, `href="/run/shop/~t1"`) {
		t.Fatalf("unexpected rewrite:\n%s", got)
	}
}

func TestLocalizeMovesALinkWithoutTheTicketToTheOpenedAddress(t *testing.T) {
	got := localize("text/html", `<head></head><link href="/run/shop/_nuxt/a.css"><a href="/run/shopping">x</a>`)

	if !strings.Contains(got, `href="/run/shop/~t1/_nuxt/a.css"`) ||
		!strings.Contains(got, `href="/run/shop/~t1/run/shopping"`) {
		t.Fatalf("got %s", got)
	}
}

func TestLocalizeStylesheetUrlsAndImports(t *testing.T) {
	rest := ` b{background:url(../j.png)} c{background:url(//x.test/k.png)}`
	got := localize("text/css", `@import "/base.css"; a{background:url('/i.png')}`+rest)
	want := `@import "/run/shop/~t1/base.css"; a{background:url('/run/shop/~t1/i.png')}` + rest

	if got != want {
		t.Fatalf("got %s", got)
	}
}

func TestLocalizeKeepsOtherAnswersAndCompressedBodies(t *testing.T) {
	text := `href="/x" url(/y)`
	if got := localize("application/json", text); got != text {
		t.Fatalf("json changed: %s", got)
	}

	if got := domain.Localize("text/html", "gzip", base, ticket, []byte(text)); string(got) != text {
		t.Fatalf("compressed body changed: %s", got)
	}
}

func TestLocalizeAddsTheShimOnceWithoutAHeadTag(t *testing.T) {
	once := localize("text/html", `<p>hi</p>`)
	if strings.Count(once, "data-site-shim") != 1 || !strings.Contains(once, `var P="/run/shop/~t1"`) {
		t.Fatalf("got %s", once)
	}

	if twice := localize("text/html", once); strings.Count(twice, "data-site-shim") != 1 {
		t.Fatal("the shim must not be added twice")
	}
}

func TestLinkKeepsTheAddressAnAppAlreadyPut(t *testing.T) {
	for in, want := range map[string]string{
		"/login": "/run/shop/~t1/login", "/run/shop/~t1/login": "/run/shop/~t1/login", "/run/shop": "/run/shop/~t1",
		"/run/shop/a?b=1": "/run/shop/~t1/a?b=1", "//x.test/y": "//x.test/y", "next": "next",
	} {
		if got := domain.Link(in, base, ticket); got != want {
			t.Errorf("Link(%q) = %q, want %q", in, got, want)
		}
	}
}
