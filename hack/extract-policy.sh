#!/bin/bash

gojq --yaml-input --yaml-output '{apiVersion:"audit.k8s.io/v1",kind:"Policy",rules:[.objects[]?.spec?.resources[]?.spec?.auditPolicy]?|.[]|select(.==null|not)}' <hack/olm-registry/olm-artifacts-template.yaml
