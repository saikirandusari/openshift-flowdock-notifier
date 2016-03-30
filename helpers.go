package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/openshift/origin/pkg/cmd/util/clientcmd"

	"github.com/emicklei/go-restful/swagger"
	"github.com/golang/glog"
)

// defaultOpenshiftPublicUrl returns the openshift public URL as defined
// in the master config - and exposed in the swagger API.
// Internally, this method will retrieve the swagger API to extract the public URL.
// If it can't be retrieved, it will either return an empty string,
// or the host of the server as defined by the client config.
func defaultOpenshiftPublicUrl(factory clientcmd.Factory) string {
	config, err := factory.OpenShiftClientConfig.ClientConfig()
	if err != nil {
		glog.Warningf("Failed to get Openshift Config: %v", err)
		return ""
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/swaggerapi/api/v1", config.Host), nil)
	if err != nil {
		glog.Warningf("Failed to build a request for the swagger API: %v", err)
		return config.Host
	}

	resp, err := client.Do(req)
	if err != nil {
		glog.Warningf("Failed to request the swagger API: %v", err)
		return config.Host
	}

	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Warningf("Failed to read the swagger API response: %v", err)
		return config.Host
	}

	swagger := &swagger.ApiDeclaration{}
	if err = json.Unmarshal(bytes, swagger); err != nil {
		glog.Warningf("Failed to unmarshall the swagger API response: %v", err)
		return config.Host
	}

	return swagger.BasePath
}
