package notification

import (
	"fmt"
	"net/smtp"
)

type SMTPNotificationProvider struct {
	host     string
	port     int
	user     string
	password string
	from     string
}

func NewSMTPNotificationProvider(host string, port int, user string, password string, from string) *SMTPNotificationProvider {
	return &SMTPNotificationProvider{
		host:     host,
		port:     port,
		user:     user,
		password: password,
		from:     from,
	}
}

func (sp *SMTPNotificationProvider) SendNotification(recipientEmail string, subject string, body string) error {

	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", sp.from, recipientEmail, subject, body)

	auth := smtp.PlainAuth("", sp.user, sp.password, sp.host)
	addr := fmt.Sprintf("%s:%d", sp.host, sp.port)

	if err := smtp.SendMail(addr, auth, sp.from, []string{recipientEmail}, []byte(message)); err != nil {
		return fmt.Errorf("failed to send email to %s: %w", recipientEmail, err)
	}

	return nil
}
