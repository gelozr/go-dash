package mail

import (
	"context"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/mail"
	"net/textproto"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestSMTPSendsWithoutCredentials(t *testing.T) {
	t.Run("successful QUIT", func(t *testing.T) {
		testSMTPSendsWithoutCredentials(t, "221 goodbye")
	})
	t.Run("failed QUIT after acceptance", func(t *testing.T) {
		testSMTPSendsWithoutCredentials(t, "421 closing connection")
	})
}

func testSMTPSendsWithoutCredentials(t *testing.T, quitReply string) {
	t.Helper()
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1")})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	deadline := time.Now().Add(5 * time.Second)
	if err := listener.SetDeadline(deadline); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithDeadline(t.Context(), deadline)
	defer cancel()
	mailer := &SMTPMailer{Host: "127.0.0.1", Port: listener.Addr().(*net.TCPAddr).Port}
	headers := map[string]string{
		"x-invoice-id": "123",
		"Reply-To":     "billing@example.com",
	}
	done := make(chan error, 1)
	go func() {
		done <- mailer.Send(ctx, &Message{
			From:    Address{Address: "sender@example.com"},
			To:      []Address{{Address: "recipient@example.com"}},
			Text:    "hello",
			Headers: headers,
		})
	}()

	conn, err := listener.AcceptTCP()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.SetDeadline(deadline); err != nil {
		t.Fatal(err)
	}
	relay := textproto.NewConn(conn)
	for _, step := range []struct {
		reply   string
		command string
	}{
		{"220 localhost", "EHLO localhost"},
		{"250 localhost", "MAIL FROM:<sender@example.com>"},
		{"250 sender accepted", "RCPT TO:<recipient@example.com>"},
		{"250 recipient accepted", "DATA"},
		{"354 send message", ""},
		{"250 message accepted", "QUIT"},
		{quitReply, ""},
	} {
		if err := relay.PrintfLine("%s", step.reply); err != nil {
			t.Fatal(err)
		}
		if step.command != "" {
			if got, err := relay.ReadLine(); err != nil || got != step.command {
				t.Fatalf("SMTP command = %q, want %q (read error %v)", got, step.command, err)
			}
		} else if strings.HasPrefix(step.reply, "354") {
			data, err := relay.ReadDotBytes()
			if err != nil {
				t.Fatal(err)
			}
			msg, err := mail.ReadMessage(strings.NewReader(string(data)))
			if err != nil {
				t.Fatal(err)
			}
			for name, want := range headers {
				if got := msg.Header.Get(name); got != want {
					t.Errorf("%s header = %q, want %q", name, got, want)
				}
			}
		}
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestSMTPRejectsInvalidCustomHeadersBeforeDial(t *testing.T) {
	for _, tc := range []struct {
		name    string
		headers map[string]string
	}{
		{"invalid name", map[string]string{"X Empty": "value"}},
		{"invalid value", map[string]string{"X-Id": "multiple\nlines"}},
		{"reserved name", map[string]string{"Content-Transfer-Encoding": "base64"}},
		{"duplicate name", map[string]string{"X-Id": "one", "x-id": "two"}},
		{"long line", map[string]string{"X-Id": strings.Repeat("x", 993)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			mailer := &SMTPMailer{}
			err := mailer.Send(ctx, &Message{Headers: tc.headers})
			if err == nil || !strings.Contains(err.Error(), "custom header") {
				t.Fatalf("send error = %v, want a custom header error before dialing", err)
			}
		})
	}
}

func TestSMTPSubjectEncoding(t *testing.T) {
	for _, tc := range []struct {
		subject string
		want    string
	}{
		{"Your invoice", "Your invoice"},
		{"Café invoice", "=?utf-8?q?Caf=C3=A9_invoice?="},
	} {
		var header strings.Builder
		(&SMTPMailer{}).buildSubject(&header, &Message{Subject: tc.subject})
		if got, want := header.String(), "Subject: "+tc.want+"\r\n"; got != want {
			t.Errorf("subject header = %q, want %q", got, want)
		}
	}
}

func TestSMTPLongSubjectFolding(t *testing.T) {
	subject := strings.Repeat("Café  🧪 ", 150) + "invoice"
	var header strings.Builder
	(&SMTPMailer{}).buildSubject(&header, &Message{Subject: subject})
	for _, line := range strings.Split(header.String(), "\r\n") {
		if len(line) > 76 {
			t.Fatalf("subject header line has %d characters, want at most 76", len(line))
		}
	}
	msg, err := mail.ReadMessage(strings.NewReader(header.String() + "\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := (&mime.WordDecoder{}).DecodeHeader(msg.Header.Get("Subject"))
	if err != nil {
		t.Fatal(err)
	}
	if got != subject {
		t.Errorf("decoded subject = %q, want %q", got, subject)
	}
}

func TestSMTPBodyRoundTrips(t *testing.T) {
	msg := &Message{
		Text: "Use =20 literally.\r\n" + strings.Repeat("café ", 30),
		HTML: "<p>Use =20 literally.</p>",
	}
	const boundary = "test-boundary"
	var body strings.Builder
	if err := (&SMTPMailer{}).buildBody(&body, msg, boundary); err != nil {
		t.Fatal(err)
	}
	body.WriteString("--" + boundary + "--\r\n")

	reader := multipart.NewReader(strings.NewReader(body.String()), boundary)
	for _, want := range []struct {
		contentType string
		body        string
	}{
		{`text/plain; charset="UTF-8"`, msg.Text},
		{`text/html; charset="UTF-8"`, msg.HTML},
	} {
		part, err := reader.NextPart()
		if err != nil {
			t.Fatal(err)
		}
		if got := part.Header.Get("Content-Type"); got != want.contentType {
			t.Fatalf("content type = %q, want %q", got, want.contentType)
		}
		got, err := io.ReadAll(part)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != want.body {
			t.Errorf("%s body = %q, want %q", want.contentType, got, want.body)
		}
	}
	if _, err := reader.NextPart(); err != io.EOF {
		t.Fatalf("expected end of multipart body, got %v", err)
	}
}

func TestSMTPCancelClosesConnection(t *testing.T) {
	t.Run("IPv4", func(t *testing.T) {
		testSMTPCancelClosesConnection(t, "127.0.0.1", "127.0.0.1")
	})
	t.Run("IPv6", func(t *testing.T) {
		testSMTPCancelClosesConnection(t, "::1", "::1")
	})
	t.Run("bracketed IPv6", func(t *testing.T) {
		testSMTPCancelClosesConnection(t, "::1", "[::1]")
	})
}

func testSMTPCancelClosesConnection(t *testing.T, listenIP, mailHost string) {
	t.Helper()
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP(listenIP)})
	if err != nil {
		if listenIP == "::1" && (errors.Is(err, syscall.EAFNOSUPPORT) || errors.Is(err, syscall.EADDRNOTAVAIL)) {
			t.Skipf("IPv6 loopback unavailable: %v", err)
		}
		t.Fatal(err)
	}
	defer listener.Close()
	if err := listener.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	mailer := &SMTPMailer{Host: mailHost, Port: listener.Addr().(*net.TCPAddr).Port}
	done := make(chan error, 1)
	go func() {
		done <- mailer.Send(ctx, &Message{From: Address{Address: "sender@example.com"}, Text: "hello"})
	}()

	conn, err := listener.AcceptTCP()
	if err != nil {
		select {
		case sendErr := <-done:
			t.Fatalf("SMTP connection was not accepted: %v; Send returned %v", err, sendErr)
		default:
		}
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	cancel()
	if _, err := conn.Read(make([]byte, 1)); err != io.EOF {
		t.Fatalf("expected SMTP connection to close after cancellation, got %v", err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Send returned %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Send did not return after cancellation")
	}
}
