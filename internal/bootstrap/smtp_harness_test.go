package bootstrap

import (
	"bufio"
	"net"
	"strings"
	"testing"
	"time"
)

// fakeSMTP accepts mail on a local port and hands each message body to messages.
func fakeSMTP(t *testing.T) (addr string, messages <-chan string) {
	t.Helper()

	lc := net.ListenConfig{}

	ln, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = ln.Close() })

	out := make(chan string, 8)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}

			go serveSMTP(conn, out)
		}
	}()

	return ln.Addr().String(), out
}

func serveSMTP(conn net.Conn, out chan<- string) {
	defer func() { _ = conn.Close() }()

	reader := bufio.NewReader(conn)
	say := func(line string) { _, _ = conn.Write([]byte(line + "\r\n")) }

	say("220 fake ready")

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}

		switch cmd := strings.ToUpper(strings.TrimSpace(line)); {
		case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
			say("250 fake")
		case cmd == "DATA":
			say("354 go ahead")
			out <- readData(reader)
			say("250 queued")
		case cmd == "QUIT":
			say("221 bye")

			return
		default:
			say("250 ok")
		}
	}
}

func readData(reader *bufio.Reader) string {
	var body strings.Builder

	for {
		line, err := reader.ReadString('\n')
		if err != nil || line == ".\r\n" {
			return body.String()
		}

		body.WriteString(line)
	}
}

func waitMail(t *testing.T, messages <-chan string) string {
	t.Helper()

	select {
	case msg := <-messages:
		return msg
	case <-time.After(5 * time.Second):
		t.Fatal("no email arrived")

		return ""
	}
}
