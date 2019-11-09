package resticmanager

import (
	"github.com/go-mail/mail"
	"github.com/i-am-david-fernandez/glog"
)

type _SmtpConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	TLS      bool
}

// Mailer provides an interface to sending e-mail
type Mailer struct {
	SMTP _SmtpConfig
}

// NewMailer creates a new Mailer
func NewMailer() *Mailer {

	mailer := &Mailer{}
	return mailer
}

func (mailer *Mailer) SendMail(sender string, recipients []string, subject string, content string) {

	m := mail.NewMessage()

	m.SetHeader("To", recipients...)
	m.SetHeader("From", sender)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", content)

	d := mail.NewDialer(
		mailer.SMTP.Host,
		mailer.SMTP.Port,
		mailer.SMTP.Username,
		mailer.SMTP.Password,
	)

	if mailer.SMTP.TLS {
		d.StartTLSPolicy = mail.MandatoryStartTLS
	}

	// Send the email
	if err := d.DialAndSend(m); err != nil {
		glog.Errorf("Could not send mail: %s", err)
	}
}

func (mailer *Mailer) SendMessage(message *MailMessage) {

	mailer.SendMail(
		message.Sender,
		message.Recipients,
		message.Subject,
		message.Content(),
	)
}
