package cmd

import (
	"github.com/i-am-david-fernandez/glog"
	"github.com/i-am-david-fernandez/restic-manager/internal"
)

func simpleCommand(resticFunction func(*resticmanager.ProfileConfiguration) (string, error), description string) {

	const logNameProfile = "profile"
	const logNameSession = "session"

	for _, profile := range resticmanager.AppConfig.Profiles {

		// Configure session logging (for subsequent e-mailing)
		sessionBackend := glog.NewListBackend("", glog.Debug)
		glog.SetBackend(logNameSession, sessionBackend)

		// Configure profile logging (to file)
		if logFilename := profile.LogFile(); (!rootFlags.noFileLogging) && (logFilename != "") {
			glog.SetBackend(logNameProfile, glog.NewFileBackend(logFilename, profile.LogFileAppend(), "", profile.LogFileLevel(), ""))
		}

		glog.Infof("Processing profile %v", profile.Name())
		glog.Debugf("  from file %v", profile.File())

		glog.Infof(description)

		restic := resticmanager.NewRestic(resticmanager.AppConfig)

		exists, err := restic.RepoExists(profile)
		if err != nil {
			glog.Errorf("Could not determine state of repository path: %v", err)
		} else if !exists {
			glog.Errorf("Repository does not exist.")
		} else {
			response, err := resticFunction(profile)
			if err != nil {
				glog.Errorf("%v", err)
			}

			glog.Infof(response)

			glog.Infof("Processing complete.")
		}

		// Clear/remove profile and session logging backends
		glog.RemoveBackend(logNameProfile)
		glog.RemoveBackend(logNameSession)
	}
}
