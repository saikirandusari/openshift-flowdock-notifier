package main

import (
	"bytes"
	"fmt"
	"os"
	"strconv"

	buildapi "github.com/openshift/origin/pkg/build/api"

	"github.com/golang/glog"
	"github.com/spf13/viper"
)

type AppConfig struct {
	BuildsWatchers map[string]*BuildsWatcherConfig
	Notifiers      map[string]*FlowdockNotifierConfig
}

type BuildsWatcherConfig struct {
	Namespace          string
	AllNamespaces      bool
	Notifiers          []string
	WatchForBuildPhase map[buildapi.BuildPhase]bool
}

type FlowdockNotifierConfig struct {
	Token           string
	SubjectTemplate string
	ContentTemplate string
	FromAddress     string
	FromName        string
	Source          string
	Tags            []string
}

func LoadAppConfig() (*AppConfig, error) {
	if path := os.Getenv("CONFIG_PATH"); len(path) > 0 {
		glog.Infof("Loading configuration from path %s", path)
		viper.AddConfigPath(path)
	}
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		glog.Warningf("Failed to read config file (%s), falling back to environment variable defined configuration...", err)
	}

	appConfig := &AppConfig{}
	if err := viper.Unmarshal(appConfig); err != nil {
		return nil, err
	}

	glog.V(1).Infof("Loaded configuration is %s", appConfig.String())

	if err := appConfig.SetFromEnvVar(); err != nil {
		return nil, err
	}

	appConfig.SetDefaults()

	glog.V(1).Infof("Full configuration (post set-from-env-var / set-defaults) is %s", appConfig.String())

	return appConfig, nil
}

func (appConfig *AppConfig) HasWatchers() bool {
	if len(appConfig.BuildsWatchers) > 0 {
		return true
	}
	return false
}

func (appConfig *AppConfig) HasNotifiers() bool {
	if len(appConfig.Notifiers) > 0 {
		return true
	}
	return false
}

func (appConfig *AppConfig) SetFromEnvVar() error {
	if appConfig.Notifiers == nil {
		appConfig.Notifiers = make(map[string]*FlowdockNotifierConfig)
	}
	if _, found := appConfig.Notifiers[DefaultNotifierName]; !found {
		appConfig.Notifiers[DefaultNotifierName] = &FlowdockNotifierConfig{}
	}
	if defaultToken := os.Getenv("NOTIFIERS_DEFAULT_TOKEN"); len(defaultToken) > 0 {
		appConfig.Notifiers[DefaultNotifierName].Token = defaultToken
	}
	if defaultSource := os.Getenv("NOTIFIERS_DEFAULT_SOURCE"); len(defaultSource) > 0 {
		appConfig.Notifiers[DefaultNotifierName].Source = defaultSource
	}
	if defaultFromName := os.Getenv("NOTIFIERS_DEFAULT_FROM_NAME"); len(defaultFromName) > 0 {
		appConfig.Notifiers[DefaultNotifierName].FromName = defaultFromName
	}
	if defaultFromAddress := os.Getenv("NOTIFIERS_DEFAULT_FROM_ADDRESS"); len(defaultFromAddress) > 0 {
		appConfig.Notifiers[DefaultNotifierName].FromAddress = defaultFromAddress
	}

	if appConfig.BuildsWatchers == nil {
		appConfig.BuildsWatchers = make(map[string]*BuildsWatcherConfig)
	}
	if len(os.Getenv("ENABLE_DEFAULT_BUILDS_WATCHER")) > 0 {
		enableDefaultBuildsWatcher, err := strconv.ParseBool(os.Getenv("ENABLE_DEFAULT_BUILDS_WATCHER"))
		if err != nil {
			return err
		}
		if enableDefaultBuildsWatcher {
			if _, found := appConfig.BuildsWatchers["default"]; !found {
				appConfig.BuildsWatchers["default"] = &BuildsWatcherConfig{
					Namespace: os.Getenv("DEFAULT_BUILDS_WATCHER_NAMESPACE"),
				}
			}
		}
	}
	if len(os.Getenv("ENABLE_ALL_BUILDS_WATCHER")) > 0 {
		enableAllBuildsWatcher, err := strconv.ParseBool(os.Getenv("ENABLE_ALL_BUILDS_WATCHER"))
		if err != nil {
			return err
		}
		if enableAllBuildsWatcher {
			if _, found := appConfig.BuildsWatchers["all"]; !found {
				appConfig.BuildsWatchers["all"] = &BuildsWatcherConfig{
					AllNamespaces: true,
				}
			}
		}
	}

	return nil
}

func (appConfig *AppConfig) SetDefaults() {
	for _, notifierConfig := range appConfig.Notifiers {
		notifierConfig.SetDefaults()
	}
	for _, watcherConfig := range appConfig.BuildsWatchers {
		watcherConfig.SetDefaults()
	}
}

func (appConfig *AppConfig) String() string {
	buffer := &bytes.Buffer{}
	fmt.Fprintf(buffer, "AppConfig with %d Builds Watchers and %d Notifiers", len(appConfig.BuildsWatchers), len(appConfig.Notifiers))
	for watcherName, watcherConfig := range appConfig.BuildsWatchers {
		fmt.Fprintf(buffer, "\n  - Build Watcher %s: %s", watcherName, watcherConfig.String())
	}
	for notifierName, notifierConfig := range appConfig.Notifiers {
		fmt.Fprintf(buffer, "\n  - Notifier %s: %s", notifierName, notifierConfig.String())
	}
	return buffer.String()
}

func (watcherConfig *BuildsWatcherConfig) SetDefaults() {
	if len(watcherConfig.Notifiers) == 0 {
		watcherConfig.Notifiers = []string{DefaultNotifierName}
	}

	if watcherConfig.WatchForBuildPhase == nil {
		watcherConfig.WatchForBuildPhase = make(map[buildapi.BuildPhase]bool)
	}
	if _, found := watcherConfig.WatchForBuildPhase[buildapi.BuildPhaseCancelled]; !found {
		watcherConfig.WatchForBuildPhase[buildapi.BuildPhaseCancelled] = false
	}
	if _, found := watcherConfig.WatchForBuildPhase[buildapi.BuildPhaseComplete]; !found {
		watcherConfig.WatchForBuildPhase[buildapi.BuildPhaseComplete] = true
	}
	if _, found := watcherConfig.WatchForBuildPhase[buildapi.BuildPhaseError]; !found {
		watcherConfig.WatchForBuildPhase[buildapi.BuildPhaseError] = true
	}
	if _, found := watcherConfig.WatchForBuildPhase[buildapi.BuildPhaseFailed]; !found {
		watcherConfig.WatchForBuildPhase[buildapi.BuildPhaseFailed] = true
	}
	if _, found := watcherConfig.WatchForBuildPhase[buildapi.BuildPhaseNew]; !found {
		watcherConfig.WatchForBuildPhase[buildapi.BuildPhaseNew] = false
	}
	if _, found := watcherConfig.WatchForBuildPhase[buildapi.BuildPhasePending]; !found {
		watcherConfig.WatchForBuildPhase[buildapi.BuildPhasePending] = false
	}
	if _, found := watcherConfig.WatchForBuildPhase[buildapi.BuildPhaseRunning]; !found {
		watcherConfig.WatchForBuildPhase[buildapi.BuildPhaseRunning] = false
	}
}

func (watcherConfig *BuildsWatcherConfig) String() string {
	return fmt.Sprintf("%+v", *watcherConfig)
}

func (notifierConfig *FlowdockNotifierConfig) SetDefaults() {
	if len(notifierConfig.SubjectTemplate) == 0 {
		notifierConfig.SubjectTemplate = DefaultSubjectTemplate
	}
	if len(notifierConfig.ContentTemplate) == 0 {
		notifierConfig.ContentTemplate = DefaultContentTemplate
	}
	if len(notifierConfig.FromAddress) == 0 {
		notifierConfig.FromAddress = DefaultFromAddress
	}
	if len(notifierConfig.FromName) == 0 {
		notifierConfig.FromName = DefaultFromName
	}
	if len(notifierConfig.Source) == 0 {
		notifierConfig.Source = DefaultSource
	}
}

func (notifierConfig *FlowdockNotifierConfig) String() string {
	return fmt.Sprintf("%+v", *notifierConfig)
}
