package mail

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net"
	"strings"
	"testing"
	"time"
)

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
	listener, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	if err := listener.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	mailer := &SMTPMailer{Host: "127.0.0.1", Port: listener.Addr().(*net.TCPAddr).Port}
	done := make(chan error, 1)
	go func() {
		done <- mailer.Send(ctx, &Message{From: Address{Address: "sender@example.com"}, Text: "hello"})
	}()

	conn, err := listener.AcceptTCP()
	if err != nil {
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
