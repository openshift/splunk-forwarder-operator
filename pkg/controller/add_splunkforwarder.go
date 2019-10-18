package controller

import (
	"github.com/openshift/splunk-forwarder-operator/pkg/controller/splunkforwarder"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, splunkforwarder.Add)
}
