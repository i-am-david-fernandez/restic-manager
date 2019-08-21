package resticmanager

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/i-am-david-fernandez/glog"

	"github.com/spf13/afero"
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v2"
)

// ProfileFilter encapsulates profile selection criteria.
type ProfileFilter struct {
	OnlyActive bool
	Names      []string
	Tags       []string
}

// NewProfileFilter creates and returns a new, empty ProfileFilter.
func NewProfileFilter() *ProfileFilter {

	return &ProfileFilter{
		OnlyActive: true,
	}
}

// ProfileConfiguration encapsulates the configuration of a restic backup profile.
type ProfileConfiguration struct {
	viper *viper.Viper
}

// IsActive returns the profile active state.
func (profile *ProfileConfiguration) IsActive() bool {

	key := "active"

	if profile.viper.IsSet(key) {
		return profile.viper.GetBool(key)
	}

	return false
}

// Name returns the profile name.
func (profile *ProfileConfiguration) Name() string {

	key := "name"

	if profile.viper.IsSet(key) {
		return profile.viper.GetString(key)
	}

	return ""
}

// Tags returns the profile tags.
func (profile *ProfileConfiguration) Tags() []string {

	key := "tags"

	if profile.viper.IsSet(key) {
		return profile.viper.GetStringSlice(key)
	}

	return nil
}

// File returns the profile file.
func (profile *ProfileConfiguration) File() string {

	return profile.viper.ConfigFileUsed()
}

// Source returns the profile source directory.
func (profile *ProfileConfiguration) Source() string {

	key := "source"

	if profile.viper.IsSet(key) {
		return profile.viper.GetString(key)
	}

	return ""
}

// Password returns the profile password.
func (profile *ProfileConfiguration) Password() string {

	key := "password"

	if profile.viper.IsSet(key) {
		return profile.viper.GetString(key)
	}

	return ""
}

// Repository returns the profile repository.
func (profile *ProfileConfiguration) Repository() string {

	key := "repo"

	if profile.viper.IsSet(key) {
		return profile.viper.GetString(key)
	}

	return ""
}

// Exclusions returns the profile exclusions.
func (profile *ProfileConfiguration) Exclusions() []string {

	key := "exclusions"

	if profile.viper.IsSet(key) {
		return profile.viper.GetStringSlice(key)
	}

	return nil
}

// LogFile returns the profile logfile name.
func (profile *ProfileConfiguration) LogFile() string {

	key := "logging.file"

	if profile.viper.IsSet(key) {
		filename := profile.viper.GetString(key)

		// Expand/substitute source or repo directory if required
		filename = strings.Replace(filename, "<source>", profile.Source(), -1)
		filename = strings.Replace(filename, "<repo>", profile.Repository(), -1)

		return filename
	}

	return ""
}

// LogFileLevel returns the profile file log level.
func (profile *ProfileConfiguration) LogFileLevel() glog.LogLevel {

	key := "logging.level"

	levelString := ""

	if profile.viper.IsSet(key) {
		levelString = profile.viper.GetString(key)
	}

	level, _ := glog.NewLogLevel(levelString)

	return level
}

// LogFileAppend returns the profile log file append mode
func (profile *ProfileConfiguration) LogFileAppend() bool {

	key := "logging.append"

	if profile.viper.IsSet(key) {
		return profile.viper.GetBool(key)
	}

	return false
}

// EmailRecipients returns the profile email recipients.
func (profile *ProfileConfiguration) EmailRecipients() []string {

	key := "email.recipients"

	if profile.viper.IsSet(key) {
		return profile.viper.GetStringSlice(key)
	}

	return nil
}

// EmailLogLevel returns the profile email log level.
func (profile *ProfileConfiguration) EmailLogLevel() glog.LogLevel {

	key := "email.level"

	levelString := ""

	if profile.viper.IsSet(key) {
		levelString = profile.viper.GetString(key)
	}

	level, _ := glog.NewLogLevel(levelString)

	return level
}

// EmailThresholds returns the email log thresholds
func (profile *ProfileConfiguration) EmailThresholds() map[glog.LogLevel]int {

	key := "email.thresholds"

	thresholds := make(map[glog.LogLevel]int)
	rawThresholds := make(map[string]int)

	if profile.viper.IsSet(key) {
		profile.viper.UnmarshalKey(key, &rawThresholds)
	}

	for k, v := range rawThresholds {
		level, _ := glog.NewLogLevel(k)
		thresholds[level] = v
	}

	return thresholds
}

// RetentionPolicy encapsulates a repository retention policy
type RetentionPolicy struct {
	Period string
	Value  int
}

// RetentionPolicies returns the profile retention policies.
func (profile *ProfileConfiguration) RetentionPolicies() []RetentionPolicy {

	key := "keep-policy"

	if profile.viper.IsSet(key) {

		var policies []RetentionPolicy

		profile.viper.UnmarshalKey(key, &policies)

		return policies
	}

	return nil
}

// OperationSequence returns the profile operation sequence.
func (profile *ProfileConfiguration) OperationSequence() []string {

	key := "operation-sequence"

	if profile.viper.IsSet(key) {
		return profile.viper.GetStringSlice(key)
	}

	return nil
}

// NewProfileConfiguration creates and returns a new, empty ProfileConfiguration.
func NewProfileConfiguration() *ProfileConfiguration {

	return &ProfileConfiguration{
		viper: viper.New(),
	}
}

// LoadProfileConfiguration creates and returns a new ProfileConfiguration populated from a file.
func LoadProfileConfiguration(filename string) *ProfileConfiguration {

	profile := NewProfileConfiguration()

	profile.Load(filename)

	return profile
}

// SetDefaults populates an existing ProfileConfiguration with defaults from a given map.
func (profile *ProfileConfiguration) SetDefaults(defaults map[string]interface{}) {

	for k, v := range defaults {
		profile.viper.SetDefault(k, v)
	}
}

// Load populates an existing ProfileConfiguration from a file.
func (profile *ProfileConfiguration) Load(filename string) {

	profile.viper.SetConfigFile(filename)
	if err := profile.viper.ReadInConfig(); err != nil {
		glog.Errorf("Could not read profile from %v: %v", profile.viper.ConfigFileUsed(), err)
	} else {
		glog.Debugf("Read profile from %v", profile.viper.ConfigFileUsed())
	}
}

// MatchesFilter returns true if the ProfileConfiguration matches the specified filter criteria.
func (profile *ProfileConfiguration) MatchesFilter(filter ProfileFilter) (bool, string) {

	if filter.OnlyActive && (!profile.IsActive()) {
		return false, "not active"
	}

	if len(filter.Names) > 0 {
		// The filter has some names. Verify that we match one of them
		profileName := profile.Name()
		matched := false
		for _, name := range filter.Names {
			if name == profileName {
				matched = true
				break
			}
		}

		if !matched {
			return false, "name not matched"
		}
	}

	if len(filter.Tags) > 0 {

		// The filter has some tags. Verify that all are present in the profile
		profileTags := profile.Tags()

		if profileTags != nil {

			// Create a set(-like) object from our slice of tags to aid intersection check
			profileTagSet := make(map[string]struct{})
			for _, tag := range profileTags {
				profileTagSet[tag] = struct{}{}
			}

			for _, tag := range filter.Tags {
				if _, exists := profileTagSet[tag]; !exists {
					// A required (filter) tag is not present in the profile.
					return false, "tags not matched"
				}
			}

		} else {
			// Profile has no tags. Match fails
			return false, "no tags to match"
		}
	}

	return true, ""
}

// SourceIsPresent returns true if the Profile source directory exists.
func (profile *ProfileConfiguration) SourceIsPresent() bool {

	source := profile.Source()

	if source != "" {

		if stat, err := os.Stat(source); err == nil && stat.IsDir() {
			// path is a directory
			return true
		}
	}

	return false
}

// String returns a string representation of the ProfileConfiguration
func (profile *ProfileConfiguration) String() string {

	c := profile.viper.AllSettings()

	content, err := yaml.Marshal(c)
	if err != nil {
		glog.Errorf("Unable to marshal config to YAML: %v", err)
		return ""
	}
	return string(content)
}

func recursiveFindProfiles(profiles []string, directory string) []string {

	fsRoot := afero.NewReadOnlyFs(afero.NewBasePathFs(afero.NewOsFs(), directory))

	items, _ := afero.ReadDir(fsRoot, "/")
	for _, item := range items {

		fullItem := path.Join(directory, item.Name())

		if item.IsDir() {
			profiles = recursiveFindProfiles(profiles, fullItem)
		} else {
			switch filepath.Ext(item.Name()) {
			case ".yaml", ".json":
				profiles = append(profiles, fullItem)
			}
		}
	}

	return profiles
}

// FindProfiles searches for profile configuration files in a specified location and returns a list of found files.
func FindProfiles(directory string) []string {

	profiles := make([]string, 0)

	directory, _ = filepath.Abs(directory)
	profiles = recursiveFindProfiles(profiles, directory)

	return profiles
}

// LoadProfiles loads the subset of the specified set of profile files that match the specified filter criteria.
func LoadProfiles(files []string, filter ProfileFilter, defaults map[string]interface{}) []*ProfileConfiguration {

	profiles := make([]*ProfileConfiguration, 0)

	for _, file := range files {

		profile := NewProfileConfiguration()
		profile.SetDefaults(defaults)
		profile.Load(file)

		if match, reason := profile.MatchesFilter(filter); !match {
			glog.Debugf("Skipping profile (filter criteria not matched: %s).", reason)
			continue
		}

		profiles = append(profiles, profile)
	}

	return profiles
}
