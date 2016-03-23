package main

import (
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	buildapi "github.com/openshift/origin/pkg/build/api"
	buildutil "github.com/openshift/origin/pkg/build/util"
	"github.com/openshift/origin/pkg/cmd/util/clientcmd"

	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/watch"
)

type Event interface {
	Namespace() string
	Name() string
	ObjectType() string
	ObjectStartTime() *unversioned.Time
	ObjectEndTime() *unversioned.Time
	ObjectDuration() time.Duration
	Input() string
	Output() string
	Status() string
	IsSuccess() bool
	IsFailure() bool
	Logs() string
	Events() []string
	NodeName() string
	Url() string
}

type BuildEvent struct {
	Event   watch.Event
	Build   *buildapi.Build
	factory clientcmd.Factory
}

func NewBuildEvent(factory clientcmd.Factory, event watch.Event) *BuildEvent {
	return &BuildEvent{
		Event:   event,
		Build:   event.Object.(*buildapi.Build),
		factory: factory,
	}
}

func (event *BuildEvent) Namespace() string {
	return event.Build.Namespace
}

func (event *BuildEvent) Name() string {
	return event.Build.Name
}

func (event *BuildEvent) ObjectType() string {
	return "Build"
}

func (event *BuildEvent) ObjectStartTime() *unversioned.Time {
	return event.Build.Status.StartTimestamp
}

func (event *BuildEvent) ObjectEndTime() *unversioned.Time {
	return event.Build.Status.CompletionTimestamp
}

func (event *BuildEvent) ObjectDuration() time.Duration {
	return event.Build.Status.Duration
}

func (event *BuildEvent) Input() string {
	if event.Build.Spec.Revision != nil {
		if event.Build.Spec.Revision.Git != nil {
			if event.Build.Spec.Source.Git != nil {
				uri := event.Build.Spec.Source.Git.URI
				uri = strings.TrimSuffix(uri, ".git")
				if strings.Index(uri, "git") == 0 {
					uri = strings.Replace(uri, "git@github.com:", "https://github.com/", 1)
				}
				return fmt.Sprintf("%s/commit/%s", uri, event.Build.Spec.Revision.Git.Commit)
			}
		}
	}
	return ""
}

func (event *BuildEvent) Output() string {
	return event.Build.Status.OutputDockerImageReference
}

func (event *BuildEvent) Status() string {
	return string(event.Build.Status.Phase)
}

func (event *BuildEvent) IsSuccess() bool {
	switch event.Build.Status.Phase {
	case buildapi.BuildPhaseComplete:
		return true
	default:
		return false
	}
}

func (event *BuildEvent) IsFailure() bool {
	switch event.Build.Status.Phase {
	case buildapi.BuildPhaseCancelled, buildapi.BuildPhaseError, buildapi.BuildPhaseFailed:
		return true
	default:
		return false
	}
}

func (event *BuildEvent) NodeName() string {
	_, kclient, err := event.factory.Clients()
	if err != nil {
		return fmt.Sprintf("Can't get kube client: %v", err)
	}

	if pod, err := kclient.Pods(event.Build.Namespace).Get(buildapi.GetBuildPodName(event.Build)); err == nil {
		return pod.Spec.NodeName
	}

	return ""
}

func (event *BuildEvent) Url() string {
	config, err := event.factory.OpenShiftClientConfig.ClientConfig()
	if err != nil {
		return fmt.Sprintf("Can't get openshift config: %v", err)
	}

	return fmt.Sprintf("%s/console/project/%s/browse/builds/%s/%s?tab=logs",
		config.Host,
		event.Build.Namespace,
		buildutil.ConfigNameForBuild(event.Build),
		event.Build.Name)
}

func (event *BuildEvent) Logs() string {
	oclient, _, err := event.factory.Clients()
	if err != nil {
		return fmt.Sprintf("Can't get openshift client: %v", err)
	}

	logs, err := oclient.BuildLogs(event.Build.Namespace).Get(event.Build.Name, buildapi.BuildLogOptions{
		TailLines: func(i int64) *int64 { return &i }(30),
	}).Stream()
	if err != nil {
		return fmt.Sprintf("Can't get build logs: %v", err)
	}
	defer logs.Close()

	bytes, err := ioutil.ReadAll(logs)
	if err != nil {
		return fmt.Sprintf("Can't read build logs: %v", err)
	}

	return string(bytes)
}

func (event *BuildEvent) Events() []string {
	_, kclient, err := event.factory.Clients()
	if err != nil {
		return []string{fmt.Sprintf("Can't get kube client: %v", err)}
	}

	events, _ := kclient.Events(event.Build.Namespace).Search(event.Build)
	if events == nil {
		events = &kapi.EventList{}
	}
	// get also pod events and merge it all into one list
	if pod, err := kclient.Pods(event.Build.Namespace).Get(buildapi.GetBuildPodName(event.Build)); err == nil {
		if podEvents, _ := kclient.Events(event.Build.Namespace).Search(pod); podEvents != nil {
			events.Items = append(events.Items, podEvents.Items...)
		}
	}

	eventsAsString := []string{}
	for _, evt := range events.Items {
		optionalSourceHost := ""
		if len(evt.Source.Host) > 0 {
			optionalSourceHost = fmt.Sprintf("on %s", evt.Source.Host)
		}
		evtAsString := fmt.Sprintf("From %s %s: %s (seen %d times between %v and %v)",
			evt.Source.Component, optionalSourceHost, evt.Message, evt.Count, evt.FirstTimestamp, evt.LastTimestamp)
		eventsAsString = append(eventsAsString, evtAsString)
	}

	return eventsAsString
}
