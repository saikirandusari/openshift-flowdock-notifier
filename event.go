package main

import (
	"fmt"
	"strings"
	"time"

	buildapi "github.com/openshift/origin/pkg/build/api"

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
}

type BuildEvent struct {
	Event watch.Event
	Build *buildapi.Build
}

func NewBuildEvent(event watch.Event) *BuildEvent {
	return &BuildEvent{
		Event: event,
		Build: event.Object.(*buildapi.Build),
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
