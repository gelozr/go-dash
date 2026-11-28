package mail

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/gelozr/go-dash/internal/config"
)

const maxHeaderLineLength = 998

type SMTPMailer struct {
	Host          string
	Port          int
	Username      string
	Password      string
	SkipTLSVerify bool
}

func NewSMTPMailer(cfg *config.Config) *SMTPMailer {
	return &SMTPMailer{
		Host:          cfg.MailHost,
		Port:          cfg.MailPort,
		Username:      cfg.MailUser,
		Password:      cfg.MailPass,
		SkipTLSVerify: cfg.MailSkipTLSVerify,
	}
}

func (s *SMTPMailer) buildMessageID(sb *strings.Builder, m *Message) {
	domain := m.From.Address[strings.LastIndex(m.From.Address, "@")+1:]
	fmt.Fprintf(sb, "Message-ID: <%d.%s@%s>\r\n", time.Now().UnixNano(), uuid.NewString(), domain)
}

func (s *SMTPMailer) buildDate(sb *strings.Builder) {
	fmt.Fprintf(sb, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
}

func (s *SMTPMailer) buildFrom(sb *strings.Builder, m *Message) {
	from := mail.Address{Name: m.From.Name, Address: m.From.Address}
	fmt.Fprintf(sb, "From: %s\r\n", from.String())
}

func (s *SMTPMailer) buildTo(sb *strings.Builder, m *Message) {
	if len(m.To) > 0 {
		toList := make([]string, len(m.To))
		for i, a := range m.To {
			to := mail.Address{Name: a.Name, Address: a.Address}
			toList[i] = to.String()
		}
		fmt.Fprintf(sb, "To: %s\r\n", strings.Join(toList, ", "))
	}
}

func (s *SMTPMailer) buildSubject(sb *strings.Builder, m *Message) {
	subject := mime.QEncoding.Encode("utf-8", m.Subject)
	if subject == m.Subject && len("Subject: ")+len(subject) > maxHeaderLineLength {
		// The encoded-word syntax leaves 60 base64 characters under the RFC 2047 limit.
		const maxWordBytes = 45
		sb.WriteString("Subject:")
		for len(subject) > 0 {
			size := min(len(subject), maxWordBytes)
			word := base64.StdEncoding.EncodeToString([]byte(subject[:size]))
			fmt.Fprintf(sb, "\r\n =?us-ascii?b?%s?=", word)
			subject = subject[size:]
		}
		sb.WriteString("\r\n")
		return
	}
	// RFC 2047 limits lines containing encoded words to 76 characters.
	if subject != m.Subject && len("Subject: ")+len(subject) > 76 {
		sb.WriteString("Subject:\r\n " + strings.ReplaceAll(subject, " ", "\r\n ") + "\r\n")
		return
	}
	sb.WriteString("Subject: " + subject + "\r\n")
}

func writeCustomHeaders(sb *strings.Builder, headers map[string]string) error {
	seen := make(map[string]struct{}, len(headers))
	for name, value := range headers {
		if name == "" || strings.IndexFunc(name, func(c rune) bool {
			return c < '!' || c > '~' || c == ':'
		}) >= 0 {
			return fmt.Errorf("invalid custom header name %q", name)
		}
		key := strings.ToLower(name)
		switch key {
		case "from", "sender", "to", "cc", "bcc", "date", "message-id", "subject", "received", "return-path", "mime-version":
			return fmt.Errorf("reserved custom header %q", name)
		}
		if strings.HasPrefix(key, "content-") || strings.HasPrefix(key, "resent-") {
			return fmt.Errorf("reserved custom header %q", name)
		}
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate custom header %q", name)
		}
		if len(name)+len(": ")+len(value) > maxHeaderLineLength || strings.IndexFunc(value, func(c rune) bool {
			return c != '\t' && (c < ' ' || c > '~')
		}) >= 0 {
			return fmt.Errorf("invalid value for custom header %q", name)
		}
		seen[key] = struct{}{}
		fmt.Fprintf(sb, "%s: %s\r\n", name, value)
	}
	return nil
}

func (s *SMTPMailer) buildBody(sb *strings.Builder, m *Message, boundary string) error {
	for _, part := range []struct {
		contentType string
		body        string
	}{
		{"text/plain", m.Text},
		{"text/html", m.HTML},
	} {
		if part.body == "" {
			continue
		}

		fmt.Fprintf(sb, "--%s\r\nContent-Type: %s; charset=\"UTF-8\"\r\n", boundary, part.contentType)
		sb.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		sb.WriteString("Content-Disposition: inline\r\n\r\n")
		w := quotedprintable.NewWriter(sb)
		if _, err := w.Write([]byte(part.body)); err != nil {
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}
		sb.WriteString("\r\n")
	}
	return nil
}

func (s *SMTPMailer) Send(ctx context.Context, m *Message) (err error) {
	boundary := uuid.New().String()

	var sb strings.Builder
	sb.WriteString("MIME-Version: 1.0\r\n")

	s.buildDate(&sb)
	s.buildMessageID(&sb, m)
	s.buildSubject(&sb, m)
	s.buildFrom(&sb, m)
	s.buildTo(&sb, m)
	if err := writeCustomHeaders(&sb, m.Headers); err != nil {
		return err
	}

	fmt.Fprintf(&sb, "Content-Type: multipart/alternative; boundary=%s\r\n", boundary)
	sb.WriteString("\r\n") // end of headers

	if err := s.buildBody(&sb, m, boundary); err != nil {
		return fmt.Errorf("encode message body: %w", err)
	}

	// closing boundary
	fmt.Fprintf(&sb, "--%s--\r\n", boundary)

	host := s.Host
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		host = host[1 : len(host)-1]
	}
	addr := net.JoinHostPort(host, strconv.Itoa(s.Port))
	tlsCfg := &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: s.SkipTLSVerify,
	}

	defer func() {
		if ctxErr := ctx.Err(); err != nil && ctxErr != nil {
			err = ctxErr
		}
	}()

	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()

	smtpConn := conn
	if s.Port == 465 {
		tlsConn := tls.Client(conn, tlsCfg)
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			return fmt.Errorf("dial tls: %w", err)
		}
		smtpConn = tlsConn
	}
	client, err := smtp.NewClient(smtpConn, host)
	if err != nil {
		return fmt.Errorf("smtp new client: %w", err)
	}
	defer client.Close()

	if s.Port != 465 {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(tlsCfg); err != nil {
				return fmt.Errorf("start tls: %w", err)
			}
		}
	}
	if s.Username != "" || s.Password != "" {
		auth := smtp.PlainAuth("", s.Username, s.Password, host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("client auth: %w", err)
		}
	}
	if err := client.Mail(m.From.Address); err != nil {
		return fmt.Errorf("client mail: %w", err)
	}
	for _, a := range m.To {
		if err := client.Rcpt(a.Address); err != nil {
			return fmt.Errorf("client rcpt: %s: %w", a.Address, err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("client data: %w", err)
	}
	if _, err := w.Write([]byte(sb.String())); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("writer close: %w", err)
	}
	// The relay has accepted the message, so a QUIT failure must not prompt a retry.
	_ = client.Quit()
	return nil
}
