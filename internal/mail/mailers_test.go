package mail

import (
	"io"
	"mime/multipart"
	"strings"
	"testing"
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
