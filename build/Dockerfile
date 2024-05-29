FROM quay.io/app-sre/boilerplate:image-v4.0.2 AS builder
COPY . /go/src/github.com/openshift/splunk-forwarder-operator
WORKDIR /go/src/github.com/openshift/splunk-forwarder-operator
RUN make go-build

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.10-896.1716497715
ENV OPERATOR_PATH=/go/src/github.com/openshift/splunk-forwarder-operator \
    OPERATOR_BIN=splunk-forwarder-operator

WORKDIR /root/
COPY --from=builder /go/src/github.com/openshift/splunk-forwarder-operator/build/_output/bin/${OPERATOR_BIN} /usr/local/bin/${OPERATOR_BIN}
LABEL io.openshift.managed.name="splunk-forwarder-operator" \
      io.openshift.managed.description="This operator will be responsible for deploying the splunk forwarder."
