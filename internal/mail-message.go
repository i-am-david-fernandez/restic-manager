package resticmanager

import (
	"bytes"
	"fmt"
	"html/template"
	"os"

	"github.com/i-am-david-fernandez/glog"
)

// MailMessage encapsulates an email message.
type MailMessage struct {
	Sender     string
	Recipients []string
	Subject    string
	content    string
}

// NewMailMessage returns a new MailMessage.
func NewMailMessage() *MailMessage {

	return &MailMessage{
		Recipients: make([]string, 0),
		content:    "",
	}
}

// Content returns the message content.
func (message *MailMessage) Content() string {
	return message.content
}

// AddRecipients adds the specified set of recipients to the message 'To' header.
func (message *MailMessage) AddRecipients(recipients ...string) {

	message.Recipients = append(message.Recipients, recipients...)
}

// SetContext sets the message context, from which a message Subject will be derived.
func (message *MailMessage) SetContext(context string) {

	appname := "restic-manager"
	hostname, _ := os.Hostname()
	message.Subject = fmt.Sprintf("%s alert from %s: %s", appname, hostname, context)
}

// AddContent adds plain content to the message content.
func (message *MailMessage) AddContent(content string) {
	message.content += content
}

// AddTemplatedContent adds templated content to the message.
func (message *MailMessage) AddTemplatedContent(templateDefinition string, data interface{}) {

	templateEngine, err := template.New("message").Parse(templateDefinition)
	if err != nil {
		glog.Errorf("Could not parse message template: %v", err)
		return
	}

	var buffer bytes.Buffer

	err = templateEngine.Execute(&buffer, data)
	if err != nil {
		glog.Errorf("Could not execute message template: %v", err)
		return
	}

	message.content += buffer.String()
}
