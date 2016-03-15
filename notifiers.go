package main

import (
	"bytes"
	"text/template"

	"github.com/golang/glog"
	"github.com/wm/go-flowdock/flowdock"
)

const (
	DefaultNotifierName       = "default"
	DefaultSuccessFromAddress = "build+ok@flowdock.com"
	DefaultFailureFromAddress = "build+fail@flowdock.com"
	DefaultFromAddress        = "openshift@example.org"
	DefaultFromName           = "OpenShift"
	DefaultSource             = "OpenShift"
	DefaultSubjectTemplate    = "{{.ObjectType}} {{.Namespace}}/{{.Name}} {{.Status}}"
	DefaultContentTemplate    = `<h3>{{.ObjectType}} {{.Namespace}}/{{.Name}}</h3>
<dl>
	<dt>Status</dt>
	<dd>{{.Status}}</dd>
	<dt>Start Time</dt>
	<dd>{{.ObjectStartTime}}</dd>
	<dt>End Time</dt>
	<dd>{{.ObjectEndTime}}</dd>
	<dt>Duration</dt>
	<dd>{{.ObjectDuration}}</dd>
	<dt>Input</dt>
	<dd>{{.Input}}</dd>
	<dt>Output</dt>
	<dd>{{.Output}}</dd>
	<dt>Node</dt>
	<dd>{{.NodeName}}</dd>
	<dt>Logs</dt>
	<dd><pre>{{.Logs}}</pre></dd>
	<dt>Events</dt>
	<dd>
		<pre>
		{{range .Events}}
		{{.}}
		{{end}}
		</pre>
	</dd>
</dl>`
)

type FlowdockNotifier struct {
	Config          FlowdockNotifierConfig
	Channel         chan Event
	FlowdockClient  *flowdock.Client
	SubjectTemplate *template.Template
	ContentTemplate *template.Template
}

func NewFlowdockNotifier(config FlowdockNotifierConfig) (*FlowdockNotifier, error) {
	subjectTemplate, err := template.New("subject").Parse(config.SubjectTemplate)
	if err != nil {
		return nil, err
	}

	contentTemplate, err := template.New("content").Parse(config.ContentTemplate)
	if err != nil {
		return nil, err
	}

	notifier := &FlowdockNotifier{
		Config:          config,
		Channel:         make(chan Event),
		FlowdockClient:  flowdock.NewClient(nil),
		SubjectTemplate: subjectTemplate,
		ContentTemplate: contentTemplate,
	}
	return notifier, nil
}

func (notifier *FlowdockNotifier) Run() {
	for {
		event, open := <-notifier.Channel

		if !open {
			glog.Errorf("Flowdock Channel has been closed!")
			break
		}

		if err := notifier.sendNotification(event); err != nil {
			glog.Errorf("Failed to send an inbox message to Flowdock: %v", err)
		}
	}
}

func (notifier *FlowdockNotifier) sendNotification(event Event) error {
	subject, err := executeTemplate(notifier.SubjectTemplate, event)
	if err != nil {
		return err
	}
	content, err := executeTemplate(notifier.ContentTemplate, event)
	if err != nil {
		return err
	}

	fromAddress := notifier.Config.FromAddress
	switch {
	case event.IsSuccess():
		fromAddress = DefaultSuccessFromAddress
	case event.IsFailure():
		fromAddress = DefaultFailureFromAddress
	}

	glog.V(2).Infof("Sending an inbox message to Flowdock...")
	_, resp, err := notifier.FlowdockClient.Inbox.Create(notifier.Config.Token, &flowdock.InboxCreateOptions{
		Source:      notifier.Config.Source,
		Project:     event.Namespace(),
		FromAddress: fromAddress,
		FromName:    notifier.Config.FromName,
		Subject:     subject,
		Content:     content,
	})
	if err != nil {
		return err
	}
	glog.V(2).Infof("Successfully sent an inbox message to Flowdock. Response is: %+v", resp)
	return nil
}

func executeTemplate(tmpl *template.Template, event Event) (string, error) {
	buffer := &bytes.Buffer{}
	err := tmpl.Execute(buffer, event)
	return buffer.String(), err
}
