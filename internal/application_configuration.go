package resticmanager

import (
	"os"

	"github.com/i-am-david-fernandez/glog"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	yaml "gopkg.in/yaml.v2"
)

// AppConfiguration encapsulates the global application configuration.
type AppConfiguration struct {
	viper    *viper.Viper
	Profiles []*ProfileConfiguration
	DryRun   bool
}

// AppConfig is the global, singleton application configuration object.
var AppConfig *AppConfiguration

//var appConfigInstance *AppConfiguration

// func AppConfig() *AppConfiguration {
// 	return appConfigInstance
// }

// NewAppConfiguration creates and returns a new, empty AppConfiguration.
func NewAppConfiguration() *AppConfiguration {

	v := viper.New()

	// Set defaults
	v.SetDefault("executable", "restic")

	return &AppConfiguration{
		viper:    v,
		Profiles: make([]*ProfileConfiguration, 0),
	}
}

// LoadAppConfiguration creates and returns a new AppConfiguration populated from a file.
func LoadAppConfiguration(filename string) *AppConfiguration {

	config := NewAppConfiguration()

	config.Load(filename)

	return config
}

// Load populates an existing AppConfig from a file.
func (appConfig *AppConfiguration) Load(filename string) {

	if filename != "" {
		// Use config file from the flag.
		appConfig.viper.SetConfigFile(filename)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			glog.Errorf("Error determining home directory: %v", err)
			os.Exit(1)
		}

		// Search config in home directory with name ".restic-manager" (without extension).
		appConfig.viper.AddConfigPath(home)
		appConfig.viper.SetConfigName(".restic-manager")
	}

	appConfig.viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := appConfig.viper.ReadInConfig(); err != nil {
		glog.Errorf("Could not load application configuration from %v: %v", filename, err)
	} else {
		glog.Debugf("Using config file: %s", appConfig.viper.ConfigFileUsed())
	}
}

// String returns a string representation of the AppConfiguration
func (appConfig *AppConfiguration) String() string {

	c := appConfig.viper.AllSettings()

	content, err := yaml.Marshal(c)
	if err != nil {
		glog.Errorf("unable to marshal config to YAML: %v", err)
		return ""
	}
	return string(content)
}

// GetProfileDefaults returns a map with default profile values
func (appConfig *AppConfiguration) GetProfileDefaults() map[string]interface{} {

	key := "profile-defaults"

	if appConfig.viper.IsSet(key) {
		return appConfig.viper.Sub(key).AllSettings()
	}

	return nil
}

// Executable returns the restic executable filename
func (appConfig *AppConfiguration) Executable() string {

	key := "executable"

	if appConfig.viper.IsSet(key) {
		return appConfig.viper.GetString(key)
	}

	return ""
}

// Tempdir returns the restic temporary directory
func (appConfig *AppConfiguration) Tempdir() string {

	key := "tempdir"

	if appConfig.viper.IsSet(key) {
		return appConfig.viper.GetString(key)
	}

	return os.TempDir()
}

type _LoggingConfig struct {
	Filename string `mapstructure:"file"`
	Level    glog.LogLevel
	Append   bool
}

// An intermediate structure, primarily to ease conversion of a string Level
// to a glog.LogLevel Level.
type _RawLoggingConfig struct {
	Filename string `mapstructure:"file"`
	Level    string
	Append   bool
}

// LoggingConfig returns the application logging configuration.
func (appConfig *AppConfiguration) LoggingConfig() *_LoggingConfig {

	key := "logging"

	if appConfig.viper.IsSet(key) {
		var r _RawLoggingConfig
		if err := appConfig.viper.UnmarshalKey(key, &r); err != nil {
			glog.Errorf("Could not retrieve configuration key %s: %v", key, err)
			return nil
		}

		// 'Level' requires some work
		level, _ := glog.NewLogLevel(appConfig.viper.GetString(key + ".level"))

		c := _LoggingConfig{
			Filename: r.Filename,
			Level:    level,
			Append:   r.Append,
		}

		return &c
	}

	return nil
}

// RawLog returns the optional raw log filename
func (appConfig *AppConfiguration) RawLog() string {

	key := "logging.raw"

	if appConfig.viper.IsSet(key) {
		return appConfig.viper.GetString(key)
	}

	return ""
}

// type _EmailConfig struct {
// 	Sender     string
// 	Recipients []string
// 	Level      glog.LogLevel
// }

// // EmailConfig retuns the application email configuration.
// func (appConfig *AppConfiguration) EmailConfig() *_EmailConfig {

// 	key := "email"

// 	if appConfig.viper.IsSet(key) {
// 		var c _EmailConfig
// 		appConfig.viper.UnmarshalKey(key, &c)

// 		// 'Level' requires some work
// 		level, _ := glog.NewLogLevel(appConfig.viper.GetString(key + ".level"))
// 		c.Level = level

// 		return &c
// 	}

// 	return nil
// }

// EmailSender returns the email sender address.
func (appConfig *AppConfiguration) EmailSender() string {

	key := "email.sender"

	if appConfig.viper.IsSet(key) {
		return appConfig.viper.GetString(key)
	}

	return ""
}

// EmailRecipients returns the email recipient addresses.
func (appConfig *AppConfiguration) EmailRecipients() []string {

	key := "email.recipients"

	if appConfig.viper.IsSet(key) {
		return appConfig.viper.GetStringSlice(key)
	}

	return nil
}

// EmailLogLevel returns the email log level
func (appConfig *AppConfiguration) EmailLogLevel() glog.LogLevel {

	key := "email.level"

	levelString := ""

	if appConfig.viper.IsSet(key) {
		levelString = appConfig.viper.GetString(key)
	}

	level, _ := glog.NewLogLevel(levelString)

	return level
}

// EmailThresholds returns the email log thresholds
func (appConfig *AppConfiguration) EmailThresholds() map[glog.LogLevel]int {

	key := "email.thresholds"

	thresholds := make(map[glog.LogLevel]int)
	rawThresholds := make(map[string]int)

	if appConfig.viper.IsSet(key) {
		if err := appConfig.viper.UnmarshalKey(key, &rawThresholds); err != nil {
			glog.Errorf("Could not retrieve configuration key %s: %v", key, err)
			return thresholds
		}
	}

	for k, v := range rawThresholds {
		level, _ := glog.NewLogLevel(k)
		thresholds[level] = v
	}

	return thresholds
}

// EmailTemplate returns the email template.
func (appConfig *AppConfiguration) EmailTemplate() string {

	key := "email.template"

	if appConfig.viper.IsSet(key) {
		return appConfig.viper.GetString(key)
	}

	return `
	<html>

	<head>
		<style>
			.code {
				font-family: monospace;
				white-space: pre;
				vertical-align: baseline;
				text-align: left;
			}
	
			.debug {
				color: darkgray;
				display: table-row;
			}
			
			.info {
				color: steelblue;
				display: table-row;
			}
			
			.notice {
				color: seagreen;
				display: table-row;
			}
			
			.warning {
				color: orange;
				display: table-row;
			}
			
			.error {
				color: darkred;
				display: table-row;
			}
			
			.critical {
				color: darkorchid;
				display: table-row;
			}

		</style>

	</head>
	
	<body>

	<div>{{.Preamble}}</div>

	<h2>Log Summary</h2>
	<table>
        {{range .LogSummary}}
		<tr class="code {{.Level}}">
        	<th>Messages at level {{.Level}}</th>
        	<td>{{.Count}}</td>
		</tr>
        {{end}}
	</table>

	<h2>Log Records</h2>
	<table>
	<tr>
		<th>Time</th>
		<th>Level</th>
		<th>Message</th>
	</tr>
	{{range .LogRecords}}
	<tr class="code {{.Level}}">
		<td>{{.Time.Format "2006-01-02 15:04:05.000"}}</td>
		<td>{{.Level}}</td>
		<td>{{.Message}}</td>
	</tr>
	{{end}}
	</table>

	</body>

	</html>
	`
}

// NewMailer returns a new Mailer
func (appConfig *AppConfiguration) NewMailer() *Mailer {

	key := "email"

	if appConfig.viper.IsSet(key) {
		mailer := NewMailer()
		if err := appConfig.viper.UnmarshalKey(key, mailer); err != nil {
			glog.Errorf("Could not retrieve configuration key %s: %v", key, err)
			return nil
		}

		return mailer
	}

	return nil
}
