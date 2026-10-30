package mail

import (
	"context"
	"errors"

	"github.com/gelozr/go-dash/internal/config"
)

type Driver string

const (
	SMTP   = Driver("smtp")
	Custom = Driver("custom")
)

type Mailer interface {
	Send(context.Context, *Message) error
}

type Manager interface {
	Mailer
	RegisterDriver(Driver, Mailer) error
	Mailer(Driver) (Mailer, error)
}

type Address struct {
	Name    string
	Address string
}

type Message struct {
	From    Address
	To      []Address
	Subject string
	HTML    string
	Text    string
	Headers map[string]string
}

type manager struct {
	mailers       map[Driver]Mailer
	defaultDriver Driver
	defaultFrom   Address
}

func NewManager(cfg *config.Config) Manager {
	mailers := make(map[Driver]Mailer)
	mailers[SMTP] = NewSMTPMailer(cfg)

	return &manager{
		mailers:       mailers,
		defaultDriver: getDefaultDriver(cfg),
		defaultFrom:   Address{Name: cfg.MailFromName, Address: cfg.MailFromAddress},
	}
}

func (m *manager) Mailer(driver Driver) (Mailer, error) {
	if ml, ok := m.mailers[driver]; ok {
		return ml, nil
	}
	return nil, errors.New("mailer not found")
}

func (m *manager) Send(ctx context.Context, msg *Message) error {
	mailer, err := m.Mailer(m.defaultDriver)
	if err != nil {
		return err
	}

	if msg.From.Address == "" {
		msg.From = m.defaultFrom
	}
	if msg.From.Address == "" {
		return errors.New("mail: no sender address, set MAIL_FROM_ADDRESS")
	}

	return mailer.Send(ctx, msg)
}

func (m *manager) RegisterDriver(driver Driver, mailer Mailer) error {
	_, ok := m.mailers[driver]
	if ok {
		return errors.New("driver already exists")
	}

	m.mailers[driver] = mailer
	return nil
}

func (m *manager) SetDefaultDriver(driver Driver) error {
	if _, ok := m.mailers[driver]; !ok {
		return errors.New("driver not found")
	}

	m.defaultDriver = driver
	return nil
}

func getDefaultDriver(cfg *config.Config) Driver {
	defaultDriver := SMTP
	if cfg.MailDriver != "" {
		defaultDriver = Driver(cfg.MailDriver)
	}
	return defaultDriver
}
