// Copyright © 2018 David Fernandez <i.am.david.fernandez@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"fmt"
	"io/ioutil"

	"github.com/i-am-david-fernandez/glog"
	resticmanager "github.com/i-am-david-fernandez/restic-manager/internal"
	"github.com/spf13/cobra"
)

// emailCmd represents the email command
var emailCmd = &cobra.Command{
	Use:   "email",
	Short: "Perform email commands.",
	Long:  `Perform email commands.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("email called")

		const logNameSession = "session"
		sessionBackend := glog.NewListBackend("", glog.Debug)
		glog.SetBackend(logNameSession, sessionBackend)

		glog.Debugf("Debug message")
		glog.Infof("Info message")
		glog.Noticef("Notice message")
		glog.Warningf("Warning message")
		glog.Errorf("Error message")
		glog.Criticalf("Critical message")

		level := resticmanager.AppConfig.EmailLogLevel()

		data := struct {
			Preamble   string
			LogSummary []*glog.RecordSummary
			LogRecords []glog.Record
		}{
			fmt.Sprintf("Note: only log messages at or above level %s are displayed.", level),
			sessionBackend.Summary(),
			sessionBackend.Get(level),
		}

		context := "Test message from restic-manager."

		message := resticmanager.NewMailMessage()
		message.Sender = resticmanager.AppConfig.EmailSender()
		message.AddRecipients(resticmanager.AppConfig.EmailRecipients()...)
		message.SetContext(context)
		message.AddTemplatedContent(resticmanager.AppConfig.EmailTemplate(), data)

		if mailer := resticmanager.AppConfig.NewMailer(); (!rootFlags.noEmail) && (mailer != nil) {

			mailer.SendMessage(message)
		} else {
			buffer := []byte(message.Content())
			ioutil.WriteFile("restic-manager.email.html", buffer, 0600)
		}
	},
}

func init() {
	rootCmd.AddCommand(emailCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// emailCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// emailCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
