FROM quay.io/redhat-services-prod/openshift/boilerplate:image-v8.1.0 AS builder
COPY . /go/src/github.com/openshift/splunk-forwarder-operator
WORKDIR /go/src/github.com/openshift/splunk-forwarder-operator
RUN make go-build

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.6-1758184547
ENV OPERATOR_PATH=/go/src/github.com/openshift/splunk-forwarder-operator \
    OPERATOR_BIN=splunk-forwarder-operator

WORKDIR /root/
COPY --from=builder /go/src/github.com/openshift/splunk-forwarder-operator/build/_output/bin/${OPERATOR_BIN} /usr/local/bin/${OPERATOR_BIN}
LABEL io.openshift.managed.name="splunk-forwarder-operator" \
      io.openshift.managed.description="This operator will be responsible for deploying the splunk forwarder."
