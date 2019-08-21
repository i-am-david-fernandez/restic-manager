/*
Copyright © 2019 David Fernandez <i.am.david.fernandez@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"os"

	"github.com/i-am-david-fernandez/glog"
	resticmanager "github.com/i-am-david-fernandez/restic-manager/internal"
	"github.com/spf13/cobra"
)

var rootFlags struct {
	dryrun        bool
	appConfigFile string
	logLevel      string
	profileDir    string
	profileFiles  []string
	profileFilter resticmanager.ProfileFilter
	noFileLogging bool
	noEmail       bool
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "restic-manager",
	Short: "A tool to provide profile-based management of restic backups.",
	Long: `restic-manager is a tool to provide management of restic backups via configured profiles.

	Among other things, a profile specifies a source and destination as well as an exclusion list
	and a retention policy.`,

	PersistentPreRun: func(cmd *cobra.Command, args []string) {

		glog.Noticef("Application Version: [%s]", resticmanager.VersionGitCommit)

		//glog.Debugf("AppConfig:\n%v\n", resticmanager.AppConfig)

		if rootFlags.profileDir != "" {
			// Find and add profile files from the specified directory
			glog.Infof("Searching for profiles in %v", rootFlags.profileDir)

			rootFlags.profileFiles = append(
				rootFlags.profileFiles,
				resticmanager.FindProfiles(rootFlags.profileDir)...,
			)
		}

		glog.Debugf("Specified and discovered profile files:\n%v\n", rootFlags.profileFiles)

		// Load all required profiles, subject to filter criteria
		resticmanager.AppConfig.Profiles = resticmanager.LoadProfiles(
			rootFlags.profileFiles,
			rootFlags.profileFilter,
			resticmanager.AppConfig.GetProfileDefaults(),
		)

		if len(resticmanager.AppConfig.Profiles) == 0 {
			glog.Warningf("No profiles loaded!")
		}

		//glog.Debugf("Active profiles:\n", resticmanager.AppConfig.Profiles)
	},
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {

	if err := rootCmd.Execute(); err != nil {
		glog.Criticalf("Error executing application: %s", err)
		os.Exit(1)
	}
}

func init() {

	cobra.OnInitialize(initConfig)

	resticmanager.AppConfig = resticmanager.NewAppConfiguration()

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&rootFlags.appConfigFile, "config", "", "config file (default is $HOME/.restic-manager.yaml)")
	rootCmd.PersistentFlags().StringVar(&rootFlags.profileDir, "profile-dir", "", "profile directory")
	rootCmd.PersistentFlags().StringArrayVar(&rootFlags.profileFiles, "profile", make([]string, 0), "profile file")

	// Profile selection filter flags
	rootCmd.PersistentFlags().BoolVar(&rootFlags.profileFilter.OnlyActive, "filter-active", true, "select only active profiles")
	rootCmd.PersistentFlags().StringSliceVar(&rootFlags.profileFilter.Names, "filter-names", make([]string, 0), "select only profiles with one of the specified names")
	rootCmd.PersistentFlags().StringSliceVar(&rootFlags.profileFilter.Tags, "filter-tags", make([]string, 0), "select only profiles with all specified tags")

	// Console logging level
	rootCmd.PersistentFlags().StringVar(&rootFlags.logLevel, "log-level", "info", "console logging level")

	rootCmd.PersistentFlags().BoolVar(&rootFlags.noEmail, "no-email", false, "disable sending of emails")
	rootCmd.PersistentFlags().BoolVar(&rootFlags.noFileLogging, "no-logfiles", false, "disable logging to file")

	// Dry-run
	rootCmd.PersistentFlags().BoolVar(&rootFlags.dryrun, "dry-run", false, "dry-run (restic will not be executed)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {

	// Configure logging
	const logNameConsole = "console"
	level, _ := glog.NewLogLevel(rootFlags.logLevel)
	glog.ClearBackends()
	glog.SetBackend(logNameConsole, glog.NewWriterBackend(os.Stderr, "", level, ""))

	// Load config from file
	resticmanager.AppConfig.Load(rootFlags.appConfigFile)
	resticmanager.AppConfig.DryRun = rootFlags.dryrun

	// Add file logging if required
	if logConfig := resticmanager.AppConfig.LoggingConfig(); (!rootFlags.noFileLogging) && (logConfig != nil) {
		const logNameFile = "appfile"

		glog.SetBackend(logNameFile,
			glog.NewFileBackend(
				logConfig.Filename,
				logConfig.Append,
				"",
				logConfig.Level,
				"", // Use default logging format
			))

	}

}
