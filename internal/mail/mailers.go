package mail

import (
	"context"
	"crypto/tls"
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
	sb.WriteString(fmt.Sprintf("Message-ID: <%d.%s@%s>\r\n", time.Now().UnixNano(), uuid.NewString(), domain))
}

func (s *SMTPMailer) buildDate(sb *strings.Builder) {
	sb.WriteString(fmt.Sprintf("Date: %s\r\n", time.Now().Format(time.RFC1123Z)))
}

func (s *SMTPMailer) buildFrom(sb *strings.Builder, m *Message) {
	from := mail.Address{Name: m.From.Name, Address: m.From.Address}
	sb.WriteString(fmt.Sprintf("From: %s\r\n", from.String()))
}

func (s *SMTPMailer) buildTo(sb *strings.Builder, m *Message) {
	if len(m.To) > 0 {
		toList := make([]string, len(m.To))
		for i, a := range m.To {
			to := mail.Address{Name: a.Name, Address: a.Address}
			toList[i] = to.String()
		}
		sb.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(toList, ", ")))
	}
}

func (s *SMTPMailer) buildSubject(sb *strings.Builder, m *Message) {
	sb.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", m.Subject) + "\r\n")
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

	sb.WriteString(fmt.Sprintf("Content-Type: multipart/alternative; boundary=%s\r\n", boundary))
	sb.WriteString("\r\n") // end of headers

	if err := s.buildBody(&sb, m, boundary); err != nil {
		return fmt.Errorf("encode message body: %w", err)
	}

	// closing boundary
	sb.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

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
	if err := client.Quit(); err != nil {
		return fmt.Errorf("quit: %w", err)
	}
	return nil
}
