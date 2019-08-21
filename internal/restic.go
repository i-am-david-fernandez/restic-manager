package resticmanager

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/i-am-david-fernandez/glog"
)

// Restic provides an interface to the restic backup application.
type Restic struct {
	executable string
	rawLog     io.Writer
}

// NewRestic creates and returns a new Restic object.
func NewRestic(appConfig *AppConfiguration) *Restic {

	executable := appConfig.Executable()
	if executable == "" {
		glog.Criticalf("No restic executable has been configured or could be found!")
		panic(nil)
	}

	rawLog := io.Writer(nil)
	rawLogFilename := appConfig.RawLog()
	if rawLogFilename != "" {

		flags := os.O_CREATE | os.O_WRONLY | os.O_APPEND
		handle, err := os.OpenFile(rawLogFilename, flags, 0600)
		if err == nil {
			rawLog = handle
		} else {
			glog.Errorf("Could not open %s for raw logging.", rawLogFilename)
		}
	}

	return &Restic{
		executable: executable,
		rawLog:     rawLog,
	}
}

// execute runs restic with the specified command, returning the captured stdout and stderr and the run return code.
func (restic *Restic) execute(command string, arguments []string, profile *ProfileConfiguration) (string, string, error) {

	process := exec.Command(restic.executable)

	// process.Args = []string{
	// 	"--repo",
	// 	profile.Repository(),
	// 	command,
	// }

	process.Args = append(process.Args,
		"--repo",
		profile.Repository(),
		command,
	)

	if arguments != nil {
		process.Args = append(process.Args, arguments...)
	}

	// process.Env = []string{
	// 	"RESTIC_PASSWORD=" + profile.Password(),
	// }

	process.Env = append(os.Environ(),
		"TMPDIR="+AppConfig.Tempdir(),
		"RESTIC_PASSWORD="+profile.Password(),
	)

	var rawStdout, rawStderr bytes.Buffer
	process.Stdout = &rawStdout
	process.Stderr = &rawStderr

	glog.Debugf("Executing %v %v", process.Path, process.Args)

	if AppConfig.DryRun {
		glog.Infof("Dry-run; no action will be performed.")
		return "", "", nil
	}

	err := process.Run()

	stdout := rawStdout.String()
	stderr := rawStderr.String()

	if restic.rawLog != nil {
		now := time.Now()
		restic.rawLog.Write([]byte(fmt.Sprintf("\nSTDOUT %s\n", now)))
		restic.rawLog.Write(rawStdout.Bytes())
		restic.rawLog.Write([]byte(fmt.Sprintf("\nSTDERR %s\n", now)))
		restic.rawLog.Write(rawStderr.Bytes())
	}

	// Remove odd/mangled characters.
	stdout = strings.Replace(stdout, "[2K", "", -1)
	stdout = strings.Replace(stdout, "[1A", "", -1)

	if command == "backup" {

		// Replace carriage-returns and line-feeds. Do this first
		// so multiline matches work properly.
		re := regexp.MustCompile("(?m:[\r\f]+)")
		stdout = re.ReplaceAllString(stdout, "\n")

		// Trim progress, which looks like this:
		//     [0:01] 0 files 0 B, total 1 files 35 B, 0 errors
		re = regexp.MustCompile(`(?m:^.*\[.*\].* files .* total .* files .* errors.*$)`)
		stdout = re.ReplaceAllString(stdout, "")

		// Remove superfluous display of unchanged files
		re = regexp.MustCompile("(?m:^unchanged.*$)")
		stdout = re.ReplaceAllString(stdout, "")

		// Remove empty lines
		re = regexp.MustCompile("\n+")
		stdout = re.ReplaceAllString(stdout, "\n")
	}

	glog.Debugf("Return:\n%v", err)
	glog.Debugf("Stdout:\n%v\n", stdout)
	glog.Debugf("Stderr:\n%v\n", stderr)

	return stdout, stderr, err
}

// RepoExists tests for the existence of repository
func (restic *Restic) RepoExists(profile *ProfileConfiguration) (bool, error) {

	// First check directory existence
	stat, err := os.Stat(profile.Repository())
	if err != nil {
		if os.IsNotExist(err) {
			// Path does not exist.
			return false, nil
		}

		// Error performing stat
		return false, errors.New("Repository path could not be read")

	} else if !stat.IsDir() {
		// Path is not a directory
		return false, errors.New("Repository is not a directory")
	}

	// Next check for repo-ness
	if stdout, stderr, err := restic.execute("snapshots", nil, profile); err != nil {
		// Path is not a repo or we could not access it (perhaps wrong password)
		return false, errors.New(stdout + "\n" + stderr)
	}

	return true, nil
}

func (restic *Restic) simpleRepoOperation(profile *ProfileConfiguration, command string, description string) (string, error) {

	glog.Noticef("%s on repo %v", description, profile.Repository())
	stdout, stderr, err := restic.execute(command, nil, profile)

	if err != nil {
		return stdout, errors.New(stderr)
	}

	return stdout, nil
}

// Raw performs an arbitrary restic operation
func (restic *Restic) Raw(profile *ProfileConfiguration, command string, arguments []string) (string, error) {

	glog.Infof("Performing %s %v on repo %v", command, arguments, profile.Repository())
	stdout, stderr, err := restic.execute(command, arguments, profile)

	if err != nil {
		return stdout, errors.New(stderr)
	}

	return stdout, nil
}

// Initialise performs a restic init operation
func (restic *Restic) Initialise(profile *ProfileConfiguration) (string, error) {

	return restic.simpleRepoOperation(profile,
		"init",
		"Initialising repository",
	)
}

// Backup performs a restic backup operation
func (restic *Restic) Backup(profile *ProfileConfiguration) (string, error) {

	glog.Noticef("Performing backup of %v", profile.Source())

	arguments := make([]string, 0)

	arguments = append(arguments, "--verbose=8")

	// Compose a set of exclusion options
	for _, e := range profile.Exclusions() {

		// Skip commented items
		if strings.HasPrefix(e, "#") {
			continue
		}

		// Expand/substitute source directory
		e = strings.Replace(e, "<source>", profile.Source(), -1)

		arguments = append(
			arguments,
			fmt.Sprintf("--exclude=%s", e),
		)
	}

	// Add source as last argument
	arguments = append(arguments, profile.Source())

	stdout, stderr, err := restic.execute("backup", arguments, profile)

	if err != nil {
		return stdout, errors.New(stderr)
	}

	return stdout, nil
}

// Check performs a restic check operation
func (restic *Restic) Check(profile *ProfileConfiguration) (string, error) {

	return restic.simpleRepoOperation(profile,
		"check",
		"Checking repository",
	)
}

// Unlock performs a restic unlock operation
func (restic *Restic) Unlock(profile *ProfileConfiguration) (string, error) {

	return restic.simpleRepoOperation(profile,
		"unlock",
		"Unlocking repository",
	)
}

// Snapshots performs a restic snapshots operation
func (restic *Restic) Snapshots(profile *ProfileConfiguration) (string, error) {

	return restic.simpleRepoOperation(profile,
		"snapshots",
		"Listing snapshots for repository",
	)
}

// Ls performs a restic ls operation
func (restic *Restic) Ls(profile *ProfileConfiguration, snapshot string) (string, error) {

	glog.Noticef("Listing files for repository at %v", profile.Repository())

	arguments := []string{snapshot}

	stdout, stderr, err := restic.execute("ls", arguments, profile)

	if err != nil {
		glog.Criticalf("Fatal error listing files: %v\nCaptured stdout:\n%v\nCaptured stderr:\n%v", err, stdout, stderr)
		return "", errors.New("ls")
	}

	if stderr != "" {
		return stdout, errors.New(stderr)
	}

	return stdout, nil
}

// ApplyRetentionPolicy performs a restic forget operation
func (restic *Restic) ApplyRetentionPolicy(profile *ProfileConfiguration) (string, error) {

	glog.Noticef("Performing retention policy application for %v", profile.Repository())

	arguments := make([]string, 0)

	// Compose a set of keep options
	for _, p := range profile.RetentionPolicies() {
		arguments = append(
			arguments,
			fmt.Sprintf("--keep-%s", p.Period),
			fmt.Sprintf("%d", p.Value),
		)
	}

	stdout, stderr, err := restic.execute("forget", arguments, profile)

	if err != nil {
		return stdout, errors.New(stderr)
	}

	return stdout, nil
}

// Clean performs a restic prune operation
func (restic *Restic) Clean(profile *ProfileConfiguration) (string, error) {

	return restic.simpleRepoOperation(profile,
		"prune",
		"Cleaning repository",
	)
}

// RebuildIndex performs a restic rebuild-index operation
func (restic *Restic) RebuildIndex(profile *ProfileConfiguration) (string, error) {

	return restic.simpleRepoOperation(profile,
		"rebuild-index",
		"Rebuilding index for repository",
	)
}
