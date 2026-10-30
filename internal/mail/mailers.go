package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/mail"
	"net/smtp"
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
	sb.WriteString("Subject: " + m.Subject + "\r\n")
}

func (s *SMTPMailer) buildBody(sb *strings.Builder, m *Message, boundary string) {
	// Plain text part
	if m.Text != "" {
		sb.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		sb.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
		sb.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		sb.WriteString("Content-Disposition: inline\r\n\r\n")
		sb.WriteString(m.Text + "\r\n\r\n")
	}

	// HTML part
	if m.HTML != "" {
		sb.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		sb.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
		sb.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
		sb.WriteString("Content-Disposition: inline\r\n\r\n")
		sb.WriteString(m.HTML + "\r\n\r\n")
	}
}

func (s *SMTPMailer) Send(ctx context.Context, m *Message) error {
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

	s.buildBody(&sb, m, boundary)

	// closing boundary
	sb.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	auth := smtp.PlainAuth("", s.Username, s.Password, s.Host)
	tlsCfg := &tls.Config{
		ServerName:         s.Host,
		InsecureSkipVerify: s.SkipTLSVerify,
	}

	// honour context cancellation
	done := make(chan error, 1)

	go func() {
		var client *smtp.Client
		var err error

		if s.Port == 465 {
			conn, e := tls.Dial("tcp", addr, tlsCfg)
			if e != nil {
				done <- fmt.Errorf("dial tls: %w", e)
				return
			}

			client, err = smtp.NewClient(conn, s.Host)
			if err != nil {
				_ = conn.Close()
				done <- fmt.Errorf("smtp new client: %w", err)
				return
			}
		} else {
			client, err = smtp.Dial(addr)
			if err != nil {
				done <- fmt.Errorf("smtp dial: %w", err)
				return
			}

			if ok, _ := client.Extension("STARTTLS"); ok {
				if err = client.StartTLS(tlsCfg); err != nil {
					_ = client.Close()
					done <- fmt.Errorf("start tls: %w", err)
					return
				}
			}
		}

		// done is written exactly once below; a second send would block forever once Send has returned
		defer client.Close()

		if err = client.Auth(auth); err != nil {
			done <- fmt.Errorf("client auth: %w", err)
			return
		}
		if err = client.Mail(m.From.Address); err != nil {
			done <- fmt.Errorf("client mail: %w", err)
			return
		}
		for _, a := range m.To {
			if err = client.Rcpt(a.Address); err != nil {
				done <- fmt.Errorf("client rcpt: %s: %w", a.Address, err)
				return
			}
		}

		w, err := client.Data()
		if err != nil {
			done <- fmt.Errorf("client data: %w", err)
			return
		}

		_, err = w.Write([]byte(sb.String()))
		if err != nil {
			done <- fmt.Errorf("write: %w", err)
			return
		}

		if err = w.Close(); err != nil {
			done <- fmt.Errorf("writer close: %w", err)
			return
		}

		if err = client.Quit(); err != nil {
			done <- fmt.Errorf("quit: %w", err)
			return
		}

		done <- nil
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}
