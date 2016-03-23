package main

import (
	"fmt"

	"github.com/openshift/origin/pkg/cmd/util/clientcmd"

	"k8s.io/kubernetes/pkg/kubectl/resource"
	"k8s.io/kubernetes/pkg/watch"

	"github.com/golang/glog"
)

type Watcher interface {
	Watch(clientcmd.Factory, map[string]FlowdockNotifier) error
}

type BuildsWatcher struct {
	Name   string
	Config BuildsWatcherConfig
}

func NewBuildsWatcher(name string, config BuildsWatcherConfig) *BuildsWatcher {
	return &BuildsWatcher{
		Name:   name,
		Config: config,
	}
}

func (watcher *BuildsWatcher) Watch(factory clientcmd.Factory, notifiers map[string]FlowdockNotifier) error {
	channels := []chan Event{}
	for _, notifierName := range watcher.Config.Notifiers {
		if notifier, found := notifiers[notifierName]; found {
			channels = append(channels, notifier.Channel)
		}
	}

	if len(channels) == 0 {
		return fmt.Errorf("no notifiers for watcher %s !", watcher.Name)
	}

	callback := func(event watch.Event) {
		buildEvent := NewBuildEvent(factory, event)
		if watcher.shouldAcceptEvent(buildEvent) {
			glog.V(3).Infof("Accepting build event %+v", buildEvent)
			for _, channel := range channels {
				channel <- buildEvent
			}
		} else {
			glog.V(3).Infof("NOT accepting build event %+v", buildEvent)
		}
	}

	glog.Infof("Watching builds - and notifying %d flows", len(channels))

	return watchResource(factory, watcher.Config.Namespace, watcher.Config.AllNamespaces, "build", callback)
}

func (watcher *BuildsWatcher) shouldAcceptEvent(buildEvent *BuildEvent) bool {

	switch buildEvent.Event.Type {
	case watch.Deleted, watch.Error:
		return false
	}

	if shouldWatchForPhase, found := watcher.Config.WatchForBuildPhase[buildEvent.Build.Status.Phase]; found {
		if !shouldWatchForPhase {
			return false
		}
	}

	return true
}

func watchResource(factory clientcmd.Factory, namespace string, allNamespaces bool, resourceType string, callback func(watch.Event)) error {
	for {
		var err error
		mapper, typer := factory.Object()
		clientMapper := factory.ClientMapperForCommand()

		if len(namespace) == 0 {
			namespace, _, err = factory.OpenShiftClientConfig.Namespace()
			if err != nil {
				return err
			}
		}

		builder := resource.NewBuilder(mapper, typer, clientMapper).
			DefaultNamespace().NamespaceParam(namespace).AllNamespaces(allNamespaces).
			ResourceTypeOrNameArgs(true, resourceType).
			SingleResourceType().
			Latest()
		r := builder.Do()
		err = r.Err()
		if err != nil {
			return err
		}

		infos, err := r.Infos()
		if err != nil {
			return err
		}
		if len(infos) != 1 {
			return fmt.Errorf("watch is only supported on individual resources and resource collections - %d resources were found", len(infos))
		}
		info := infos[0]
		mapping := info.ResourceMapping()

		obj, err := r.Object()
		if err != nil {
			return err
		}
		rv, err := mapping.MetadataAccessor.ResourceVersion(obj)
		if err != nil {
			return err
		}

		w, err := r.Watch(rv)
		if err != nil {
			return err
		}

		if allNamespaces {
			glog.V(2).Infof("Starting watch loop on %s resource type for all namespaces", resourceType)
		} else {
			glog.V(2).Infof("Starting watch loop on %s resource type for namespace %s", resourceType, namespace)
		}
		for {
			event, open := <-w.ResultChan()
			if !open {
				glog.Warningf("Watch channel has been closed!")
				break
			}
			glog.V(3).Infof("Got event %v for %T", event.Type, event.Object)
			callback(event)
		}
		glog.V(2).Infof("End of watch loop on %s resource type for namespace %s", resourceType, namespace)
	}
}
