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
	"time"

	"github.com/i-am-david-fernandez/glog"
	resticmanager "github.com/i-am-david-fernandez/restic-manager/internal"
	"github.com/spf13/cobra"
)

// autoCmd represents the auto command
var autoCmd = &cobra.Command{
	Use:   "auto",
	Short: "Perform automatic management of backup profile.",
	Long:  `Perform automatic management of backup profile.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("auto called")

		const logNameProfile = "profile"
		const logNameSession = "session"

		glog.Infof("==== ==== ==== ====")
		tStart := time.Now()

		for _, profile := range resticmanager.AppConfig.Profiles {

			tProfile := time.Now()

			// Configure session logging (for subsequent e-mailing)
			// Note: we capture everything in the session backend and perform
			// context-specific filtering later.
			sessionBackend := glog.NewListBackend("", glog.Debug)
			glog.SetBackend(logNameSession, sessionBackend)

			// Configure profile logging (to file)
			if logFilename := profile.LogFile(); (!rootFlags.noFileLogging) && (logFilename != "") {
				glog.SetBackend(logNameProfile, glog.NewFileBackend(logFilename, profile.LogFileAppend(), "", profile.LogFileLevel(), ""))
			}

			glog.Infof("---- ---- ---- ----")
			glog.Noticef("Processing profile %v", profile.Name())
			glog.Debugf("  from file %v", profile.File())

			glog.Infof("Performing automatic management.")

			restic := resticmanager.NewRestic(resticmanager.AppConfig)

			exists, err := restic.RepoExists(profile)
			if err != nil {
				glog.Errorf("Could not determine state of repository path: %v", err)
			} else {

				if !exists {
					glog.Warningf("Repository does not exist at %v", profile.Repository())
				}

				proceed := true

				for _, operation := range profile.OperationSequence() {

					switch operation {

					case "initialise":
						// Conditionally initialise repo
						if !exists {
							glog.Infof("Repository does not exist. Initialising...")

							response, err := restic.Initialise(profile)
							if err != nil {
								glog.Errorf("%v", err)
								proceed = false
							}
							glog.Infof(response)
							exists = true
						}

					case "unlock":
						// Unlock
						response, err := restic.Unlock(profile)
						if err != nil {
							glog.Errorf("%v", err)
							proceed = false
						}
						glog.Infof(response)

					case "backup":
						// Backup
						response, err := restic.Backup(profile)
						if err != nil {
							glog.Errorf("%v", err)
							proceed = false
						}
						glog.Infof(response)

					case "check":
						// Check repo
						response, err := restic.Check(profile)
						if err != nil {
							glog.Errorf("%v", err)
							proceed = false
						}
						glog.Infof(response)

					case "apply-retention":
						// Apply retention policies
						response, err := restic.ApplyRetentionPolicy(profile)
						if err != nil {
							glog.Errorf("%v", err)
							proceed = false
						}
						glog.Infof(response)

					case "show-snapshots":
						// Show snapshots
						response, err := restic.Snapshots(profile)
						if err != nil {
							glog.Errorf("%v", err)
							proceed = false
						}
						glog.Infof(response)

					case "show-listing":
						// Show listing
						response, err := restic.Ls(profile, "latest")
						if err != nil {
							glog.Errorf("%v", err)
							proceed = false
						}
						glog.Infof(response)
					}

					if !proceed {
						glog.Errorf("Error performing operation. Cannot proceed with profile.")
						break
					}
				}
			}

			tNow := time.Now()
			elapsed := tNow.Sub(tProfile)
			glog.Infof("Profile elapsed time: %v", elapsed)

			if !resticmanager.AppConfig.DryRun {
				if mailer := resticmanager.AppConfig.NewMailer(); mailer != nil {

					glog.Infof("Mailing log.")
					context := fmt.Sprintf("Performing automatic management of profile %s", profile.Name())

					// We will sent a set of emails, each to an independent recipient list and with an independent log level filter.
					cases := []struct {
						recipients []string
						level      glog.LogLevel
						thresholds map[glog.LogLevel]int
					}{
						{
							// Messages to application-configured recipients
							resticmanager.AppConfig.EmailRecipients(),
							resticmanager.AppConfig.EmailLogLevel(),
							resticmanager.AppConfig.EmailThresholds(),
						},
						{
							// Messages to profile-configured recipients
							profile.EmailRecipients(),
							profile.EmailLogLevel(),
							profile.EmailThresholds(),
						},
					}

					data := struct {
						Preamble   string
						LogSummary []*glog.RecordSummary
						LogRecords []glog.Record
					}{
						"",
						sessionBackend.Summary(),
						nil,
					}

					for _, c := range cases {
						if c.recipients != nil {

							proceed := true

							if len(c.thresholds) > 0 {
								// We have some configured thresholds.
								// Assume we should _not_ proceed (to mail)
								// unless one or more thresholds are exceeded
								proceed = false
								for _, bin := range data.LogSummary {
									if threshold, ok := c.thresholds[bin.Level]; ok {
										if bin.Count >= threshold {
											proceed = true
										}
									}
								}
							}

							if proceed {
								message := resticmanager.NewMailMessage()
								message.Sender = resticmanager.AppConfig.EmailSender()
								message.AddRecipients(c.recipients...)
								message.SetContext(context)

								data.Preamble = fmt.Sprintf("Note: only log messages at or above level %s are displayed.", c.level)
								data.LogRecords = sessionBackend.Get(c.level)
								message.AddTemplatedContent(resticmanager.AppConfig.EmailTemplate(), data)

								if !rootFlags.noEmail {
									mailer.SendMessage(message)
								} else {
									buffer := []byte(message.Content())
									ioutil.WriteFile(fmt.Sprintf("%s.html", profile.Name()), buffer, 0600)
								}
							}
						}
					}
				}
			}

			// Clear/remove profile and session logging backends
			glog.RemoveBackend(logNameProfile)
			glog.RemoveBackend(logNameSession)
		}

		tNow := time.Now()
		elapsed := tNow.Sub(tStart)
		glog.Infof("Total elapsed time: %v", elapsed)
		glog.Infof("==== ==== ==== ====")
	},
}

func init() {
	rootCmd.AddCommand(autoCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// autoCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// autoCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
