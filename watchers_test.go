package main

import (
	"testing"

	buildapi "github.com/openshift/origin/pkg/build/api"

	"k8s.io/kubernetes/pkg/watch"
)

func TestBuildsWatcherShouldAcceptEvent(t *testing.T) {
	tests := []struct {
		buildsWatcher  *BuildsWatcher
		buildEvent     *BuildEvent
		expectedResult bool
	}{
		// should not accept "deleted" events
		{
			buildsWatcher: NewBuildsWatcher("test", BuildsWatcherConfig{}),
			buildEvent: &BuildEvent{
				Event: watch.Event{
					Type: watch.Deleted,
				},
			},
			expectedResult: false,
		},
		// should not accept "error" events
		{
			buildsWatcher: NewBuildsWatcher("test", BuildsWatcherConfig{}),
			buildEvent: &BuildEvent{
				Event: watch.Event{
					Type: watch.Error,
				},
			},
			expectedResult: false,
		},
		// should not accept an event if we don't want to watch for its phase
		{
			buildsWatcher: NewBuildsWatcher("test", BuildsWatcherConfig{
				WatchForBuildPhase: map[buildapi.BuildPhase]bool{
					buildapi.BuildPhaseNew: false,
				},
			}),
			buildEvent: &BuildEvent{
				Event: watch.Event{
					Type: watch.Added,
				},
				Build: &buildapi.Build{
					Status: buildapi.BuildStatus{
						Phase: buildapi.BuildPhaseNew,
					},
				},
			},
			expectedResult: false,
		},
		// should accept an event if we want to watch for its phase
		{
			buildsWatcher: NewBuildsWatcher("test", BuildsWatcherConfig{
				WatchForBuildPhase: map[buildapi.BuildPhase]bool{
					buildapi.BuildPhaseNew: true,
				},
			}),
			buildEvent: &BuildEvent{
				Event: watch.Event{
					Type: watch.Added,
				},
				Build: &buildapi.Build{
					Status: buildapi.BuildStatus{
						Phase: buildapi.BuildPhaseNew,
					},
				},
			},
			expectedResult: true,
		},
		// should accept an event if we don't explicitly want to watch for its phase
		{
			buildsWatcher: NewBuildsWatcher("test", BuildsWatcherConfig{
				WatchForBuildPhase: map[buildapi.BuildPhase]bool{},
			}),
			buildEvent: &BuildEvent{
				Event: watch.Event{
					Type: watch.Added,
				},
				Build: &buildapi.Build{
					Status: buildapi.BuildStatus{
						Phase: buildapi.BuildPhaseNew,
					},
				},
			},
			expectedResult: true,
		},
	}

	for count, test := range tests {
		result := test.buildsWatcher.shouldAcceptEvent(test.buildEvent)
		if result != test.expectedResult {
			t.Errorf("Test[%d] Failed: Expected '%v' but got '%v'", count, test.expectedResult, result)
		}
	}
}
