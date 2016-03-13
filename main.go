package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang/glog"
	"github.com/openshift/origin/pkg/cmd/util/clientcmd"
	"github.com/spf13/pflag"
)

func main() {
	flags := pflag.NewFlagSet("openshift-flowdock-notifier", pflag.ExitOnError)
	flags.AddGoFlagSet(flag.CommandLine)
	flags.Parse(os.Args[1:])

	factory := getFactory(flags)

	appConfig, err := LoadAppConfig()
	if err != nil {
		glog.Fatalf("Failed to load configuration: %v", err)
	}

	if !appConfig.HasWatchers() {
		glog.Fatalf("No watchers have been defined in the configuration. Closing application!")
	}
	if !appConfig.HasNotifiers() {
		glog.Fatalf("No notifiers have been defined in the configuration. Closing application!")
	}

	notifiers := make(map[string]FlowdockNotifier)
	for notifierName, notifierConfig := range appConfig.Notifiers {
		notifier, err := NewFlowdockNotifier(*notifierConfig)
		if err != nil {
			glog.Fatalf("Failed to create Flowdock Notifier %s: %v", notifierName, err)
		}
		notifiers[notifierName] = *notifier
		go notifier.Run()
	}

	errors := make(chan error)
	for watcherName, watcherConfig := range appConfig.BuildsWatchers {
		watcher := NewBuildsWatcher(watcherName, *watcherConfig)
		go func(factory *clientcmd.Factory, notifiers *map[string]FlowdockNotifier, errors chan<- error) {
			if err := watcher.Watch(*factory, *notifiers); err != nil {
				errors <- err
			}
		}(factory, &notifiers, errors)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGTERM)

	select {
	case <-c:
		glog.Infof("Interrupted by user (or killed) !")
	case err := <-errors:
		glog.Fatalf("Error caught while watching: %v", err)
	}
}
