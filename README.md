# OpenShift Flowdock Notifier

**Sends notifications to Flowdock about events happening in your OpenShift cluster.**

[![Travis](https://travis-ci.org/vbehar/openshift-flowdock-notifier.svg?branch=master)](https://travis-ci.org/vbehar/openshift-flowdock-notifier)
[![DockerHub](https://img.shields.io/badge/docker-vbehar%2Fopenshift--flowdock--notifier-008bb8.svg)](https://hub.docker.com/r/vbehar/openshift-flowdock-notifier/)

This [Go](http://golang.org/) application is an [OpenShift](http://www.openshift.org/) client that listen to events in the cluster, and post notifications to the inbox of your [Flowdock](https://flowdock.com/) flows.

## How It Works

This application can be deployed on OpenShift - or anywhere else, but I guess you will want to run it in your OpenShift cluster.

It should run with a [ServiceAccount](https://docs.openshift.org/latest/architecture/core_concepts/projects_and_users.html#users) that has enough rights to watch the events you want to forward to Flowdock. You have 2 solutions:

* use the [provided template](openshift-template-deploy-only.yml) that creates a specific ServiceAccount (`flowdock-notifier`) with the right role (`view`) - this is good if you want notifications for a single project, or for just a few projects (in this case you will need to add more rights to the ServiceAccount)
* create a ServiceAccount with the `cluster-reader` role - this solution should be used if you want notifications for the whole cluster, but it requires admin rights to setup.

Based on a configuration file (or environment variables), it will watch pre-defined types of resources, and use the [Flowdock API](https://www.flowdock.com/api) to send notifications to the [Team Inbox](https://www.flowdock.com/help/team_inbox) of one (or more) [flows](https://www.flowdock.com/help/flows).

It uses the [Flow Token](https://www.flowdock.com/api/authentication#source-token) to send mail-like messages to the team inbox of a flow using the [Team Inbox Push API](https://www.flowdock.com/api/team-inbox). You can find the flows tokens in your [account page](https://www.flowdock.com/account/tokens).

### Supported events

For the moment, the following events are supported:

* [Builds](https://docs.openshift.org/latest/architecture/core_concepts/builds_and_image_streams.html#builds) events: when a new build has been started, has successfully completed, has failed, has been cancelled, ...

More events are in the roadmap ;-)

### Configuration

You can either use a few environment variables for a basic and simple configuration - it is the fastest to setup, but more limited - or you can use a configuration file, but it requires more work to setup.

#### Environment variables based configuration

This way of configuring the application is recommanded if you want to send the notifications of a single project to a single flow.

The following environment variables are supported:

* `NOTIFIERS_DEFAULT_TOKEN` to configure the [Flow Token](https://www.flowdock.com/api/authentication#source-token) for the flow that will receive the notifications - Go to your [account page](https://www.flowdock.com/account/tokens) to retrieve the token.
* `NOTIFIERS_DEFAULT_SOURCE` if you want to overwrite the name of the source in the notification - defaults to `OpenShift`.
* `NOTIFIERS_DEFAULT_FROM_NAME` if you want to overwrite the name of the sender in the notification - defaults to `OpenShift`.
* `NOTIFIERS_DEFAULT_FROM_ADDRESS` if you want to overwrite the address of the sender in the notification - defaults to `build+ok@flowdock.com` for successful builds or `build+fail@flowdock.com` for failed builds, or `openshift@example.org` for all other events. Note that this address is used to display an avatar from the [Gravatar service](https://gravatar.com/) - see the [Flowdock Team Inbox API](https://www.flowdock.com/api/team-inbox) for more informations.
* `ENABLE_DEFAULT_BUILDS_WATCHER` to enable the default builds watcher, that will 

## Running on OpenShift

If you want to deploy this application on an OpenShift cluster, you need to:

* create a new application from the provided [openshift-template-deploy-only.yml](openshift-template-deploy-only.yml) template, and overwrite some parameters:

  ```
  oc new-app -f openshift-template-deploy-only.yml -p FLOW_TOKEN=xxx
  ```

  Of course, replace `xxx` by the value of your [GitHub Access Token](https://help.github.com/articles/creating-an-access-token-for-command-line-use/). To create such a token, go to your [GitHub Tokens Settings](https://github.com/settings/tokens) page, and create a new token with the `admin:repo_hook` scope.

* optional - if you want to get notifications from more than 1 project:

  * either add the `view` role to the `flowdock-notifier` ServiceAccount in a different project (you will need to be admin in the other project):

    ```
    oc policy add-role-to-user view system:serviceaccount:$(oc project -q):flowdock-notifier -n PROJECT_NAME
    ```

    The `$(oc project -q)` is used to get the name of the current project, in which the `flowdock-notifier` ServiceAccount has been created. Don't forget to replace `PROJECT_NAME` with the name of the other project. This can be repeated for any number of projects.

  * or give the `cluster-reader` role to the `flowdock-notifier` ServiceAccount (you will need to be cluster admin for that):

    ```
    oadm policy add-cluster-role-to-user cluster-reader system:serviceaccount:$(oc project -q):flowdock-notifier
    ```

    The `$(oc project -q)` is used to get the name of the current project, in which the `flowdock-notifier` ServiceAccount has been created.

You can use either of the following templates:

* [openshift-template-deploy-only.yml](openshift-template-deploy-only.yml) to just deploy from an existing Docker image - by default [vbehar/openshift-flowdock-notifier](https://hub.docker.com/r/vbehar/openshift-flowdock-notifier/)
* [openshift-template-full.yml](openshift-template-full.yml) to build from sources (by default the [vbehar/openshift-flowdock-notifier](https://github.com/vbehar/openshift-flowdock-notifier) github repository) and then deploy

## Running locally

If you want to run it on your laptop:

* Install [Go](http://golang.org/) (tested with 1.6) and [setup your GOPATH](https://golang.org/doc/code.html)
* clone the sources in your `GOPATH`

  ```
  git clone https://github.com/vbehar/openshift-flowdock-notifier.git $GOPATH/src/github.com/vbehar/openshift-flowdock-notifier
  ```

* install [godep](https://github.com/tools/godep) (to use the vendored dependencies)

  ```
  go get github.com/tools/godep
  ```

* build the binary with godep:

  ```
  cd $GOPATH/src/github.com/vbehar/openshift-flowdock-notifier
  godep go build
  ```

* configure the application, either with a `config.yml` file, or with some environment variables (see the configuration section)

* start the application in verbose mode

  ```
  ./openshift-flowdock-notifier --v=3
  ```

* enjoy!

## License

Copyright 2016 the original author or authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.