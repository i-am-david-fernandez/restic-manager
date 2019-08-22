package resticmanager

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
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

// SnapshotIDFromIndex retrieves a snapshot ID from an index, where 0 is the most-recent
func (restic *Restic) SnapshotIDFromIndex(profile *ProfileConfiguration, index int) (string, error) {

	glog.Noticef("Retrieving ID of snapshot %d for repository at %v", index, profile.Repository())

	arguments := []string{"--json"}

	stdout, stderr, err := restic.execute("snapshots", arguments, profile)

	if err != nil {
		glog.Criticalf("Fatal error listing files: %v\nCaptured stdout:\n%v\nCaptured stderr:\n%v", err, stdout, stderr)
		return "", errors.New("snapshots")
	}

	if stderr != "" {
		return "", errors.New(stderr)
	}

	var bytes = []byte(stdout)
	var data []map[string]interface{}

	err = json.Unmarshal(bytes, &data)
	if err != nil {
		glog.Criticalf("Fatal error listing snapshots: %v\nCaptured stdout:\n%v\nCaptured stderr:\n%v", err, stdout, stderr)
		return "", errors.New("snapshots")
	}

	var Nsnapshots = len(data)

	if Nsnapshots <= index {
		return "", errors.New("snapshots")
	}

	if id, ok := data[Nsnapshots-index-1]["id"].(string); ok {
		return id, nil
	}

	return "", errors.New("snapshots")
}

// Diff retrieves a difference summary between two specified snapshot IDs
func (restic *Restic) Diff(profile *ProfileConfiguration, beforeID string, afterID string) (*SnapshotDiff, error) {
	glog.Noticef("Diffing snapshot %s -> %s for repository at %v", beforeID, afterID, profile.Repository())

	arguments := []string{
		beforeID,
		afterID,
	}

	stdout, stderr, err := restic.execute("diff", arguments, profile)

	if err != nil {
		glog.Criticalf("Fatal error performing diff: %v\nCaptured stdout:\n%v\nCaptured stderr:\n%v", err, stdout, stderr)
		return nil, errors.New("diff")
	}

	diff := NewSnapshotDiff(stdout)

	if thresholds := profile.ChangeThresholds(); thresholds != nil {

		if thresholds.TotalFiles >= 0 {
			totalFiles := diff.FilesNew + diff.FilesRemoved + diff.FilesChanged
			if totalFiles > thresholds.TotalFiles {
				glog.Warningf("Total file change threshold exceeded (%v > %v).", totalFiles, thresholds.TotalFiles)
			}
		}

		if thresholds.TotalBytes >= 0 {
			totalBytes := diff.BytesAdded + diff.BytesRemoved
			if totalBytes > thresholds.TotalBytes {
				glog.Warningf("Total size change threshold exceeded (%v > %v).", totalBytes, thresholds.TotalBytes)
			}
		}
	}

	if stderr != "" {
		return diff, errors.New(stderr)
	}

	return diff, nil
}

// DiffFromIndices retrieves a difference summary between two specified snapshot indices (0 being most recent, 1 being second-most-recent, etc.)
func (restic *Restic) DiffFromIndices(profile *ProfileConfiguration, beforeIndex int, afterIndex int) (*SnapshotDiff, error) {

	snapshotBefore, err := restic.SnapshotIDFromIndex(profile, beforeIndex)
	if err != nil {
		glog.Errorf("Could not determine snapshot (before) at index %d: %v", beforeIndex, err)
		return nil, errors.New("diff")
	} else if snapshotBefore == "" {
		glog.Errorf("Snapshot (before) at index %d does not exist.", beforeIndex)
		return nil, errors.New("diff")
	}
	glog.Debugf("Snapshot (before) ID: %s", snapshotBefore)

	snapshotAfter, err := restic.SnapshotIDFromIndex(profile, afterIndex)
	if err != nil {
		glog.Errorf("Could not determine snapshot (after) at index %d: %v", afterIndex, err)
		return nil, errors.New("diff")
	} else if snapshotAfter == "" {
		glog.Errorf("Snapshot (after) at index %d does not exist.", afterIndex)
		return nil, errors.New("diff")
	}
	glog.Debugf("Snapshot (after) ID: %s", snapshotAfter)

	diff, err := restic.Diff(profile, snapshotBefore, snapshotAfter)
	if err != nil {
		glog.Errorf("%v", err)
		return diff, errors.New("diff")
	}

	// Perform change-threshold checks if required

	return diff, nil
}

// SnapshotDiff encapsulates the differences between two snapshots
type SnapshotDiff struct {
	Report       string
	FilesNew     int
	FilesRemoved int
	FilesChanged int
	DirsNew      int
	DirsRemoved  int
	BytesAdded   float64
	BytesRemoved float64
}

// NewSnapshotDiff creates and returns a new SnapshotDiff object.
func NewSnapshotDiff(diffText string) *SnapshotDiff {

	snapshotDiff := SnapshotDiff{
		Report:       diffText,
		FilesNew:     0,
		FilesRemoved: 0,
		FilesChanged: 0,
		DirsNew:      0,
		DirsRemoved:  0,
		BytesAdded:   0,
		BytesRemoved: 0,
	}

	snapshotDiff.parse((diffText))

	return &snapshotDiff
}

func (diff *SnapshotDiff) parse(diffText string) {

	/*
	 We are looking for a section as follows:

	   Files:          80 new,     0 removed,     0 changed
	   Dirs:           57 new,     0 removed
	   Others:          0 new,     0 removed
	   Data Blobs:     90 new,     0 removed
	   Tree Blobs:     60 new,     3 removed
	     Added:   27.734 MiB
	     Removed: 941 B
	*/

	reFiles := regexp.MustCompile(`Files:\s*(\d+)\s+new,\s*(\d+)\s+removed,\s*(\d+)\s+changed`)
	reDirs := regexp.MustCompile(`Dirs:\s*(\d+)\s+new,\s*(\d+)\s+removed`)
	reBytesAdded := regexp.MustCompile(`Added:\s*(\d+\.?\d*)\s+(.*)`)
	reBytesRemoved := regexp.MustCompile(`Removed:\s*(\d+\.?\d*)\s+(.*)`)

	scanner := bufio.NewScanner(strings.NewReader(diffText))
	for scanner.Scan() {
		line := scanner.Text()

		// Check for "Files" section
		if match := reFiles.FindStringSubmatch(line); match != nil {

			// Try to extract "new"
			index := 1
			if len(match) <= index {
				glog.Errorf("Error extracting new-file count at index %d from %v", index, match)
			} else {
				if count, err := strconv.Atoi(match[index]); err != nil {
					glog.Errorf("Error converting new-file count (%s): %v", match[index], err)
				} else {
					diff.FilesNew = count
				}
			}

			// Try to extract "removed"
			index = 2
			if len(match) <= index {
				glog.Errorf("Error extracting removed-file count at index %d from %v", index, match)
			} else {
				if count, err := strconv.Atoi(match[index]); err != nil {
					glog.Errorf("Error converting removed-file count (%s): %v", match[index], err)
				} else {
					diff.FilesRemoved = count
				}
			}

			// Try to extract "changed"
			index = 3
			if len(match) <= index {
				glog.Errorf("Error extracting changed-file count at index %d from %v", index, match)
			} else {
				if count, err := strconv.Atoi(match[index]); err != nil {
					glog.Errorf("Error converting changed-file count (%s): %v", match[index], err)
				} else {
					diff.FilesChanged = count
				}
			}
		}

		// Check for "Dirs" section
		if match := reDirs.FindStringSubmatch(line); match != nil {

			// Try to extract "new"
			index := 1
			if len(match) <= index {
				glog.Errorf("Error extracting new-dir count at index %d from %v", index, match)
			} else {
				if count, err := strconv.Atoi(match[index]); err != nil {
					glog.Errorf("Error converting new-dir count (%s): %v", match[index], err)
				} else {
					diff.DirsNew = count
				}
			}

			// Try to extract "removed"
			index = 2
			if len(match) <= index {
				glog.Errorf("Error extracting removed-dir count at index %d from %v", index, match)
			} else {
				if count, err := strconv.Atoi(match[index]); err != nil {
					glog.Errorf("Error converting removed-dir count (%s): %v", match[index], err)
				} else {
					diff.DirsRemoved = count
				}
			}
		}

		// Check for (bytes) "Added" section
		if match := reBytesAdded.FindStringSubmatch(line); match != nil {

			if len(match) < 3 {
				glog.Errorf("Error extracting added bytes from %v", match)
			} else {
				// Extract number
				index := 1
				if value, err := strconv.ParseFloat(match[index], 64); err != nil {
					glog.Errorf("Error converting added bytes count (%s): %v", match[index], err)
				} else {
					// Extract units
					index = 2
					switch unit := match[index]; unit {
					case "B":
						value *= 1
					case "KiB":
						value *= 1024
					case "MiB":
						value *= 1024 * 1024
					case "GiB":
						value *= 1024 * 1024 * 1024
					case "TiB":
						value *= 1024 * 1024 * 1024 * 1024
					}

					diff.BytesAdded = value
				}
			}
		}

		// Check for (bytes) "Removed" section
		if match := reBytesRemoved.FindStringSubmatch(line); match != nil {

			if len(match) < 3 {
				glog.Errorf("Error extracting removed bytes from %v", match)
			} else {
				// Extract number
				index := 1
				if value, err := strconv.ParseFloat(match[index], 64); err != nil {
					glog.Errorf("Error converting removed bytes count (%s): %v", match[index], err)
				} else {

					// Extract units
					index = 2
					switch unit := match[index]; unit {
					case "B":
						value *= 1
					case "KiB":
						value *= 1024
					case "MiB":
						value *= 1024 * 1024
					case "GiB":
						value *= 1024 * 1024 * 1024
					case "TiB":
						value *= 1024 * 1024 * 1024 * 1024
					}

					diff.BytesRemoved = value
				}
			}
		}
	}
}
