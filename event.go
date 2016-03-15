package main

import (
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	buildapi "github.com/openshift/origin/pkg/build/api"
	"github.com/openshift/origin/pkg/cmd/util/clientcmd"

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

func (event *BuildEvent) Logs() string {
	oclient, _, err := event.factory.Clients()
	if err != nil {
		return fmt.Sprintf("Can't get openshift client: %v", err)
	}

	logs, err := oclient.BuildLogs(event.Build.Namespace).Get(event.Build.Name, buildapi.BuildLogOptions{}).Stream()
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
