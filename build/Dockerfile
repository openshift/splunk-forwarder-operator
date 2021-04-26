# Currenlty the use of ${OPERATOR_PATH} is failing. Explicity pass the path for now. 
FROM registry.ci.openshift.org/openshift/release:golang-1.13 AS builder
COPY . /go/src/github.com/openshift/splunk-forwarder-operator
WORKDIR /go/src/github.com/openshift/splunk-forwarder-operator
RUN make gobuild

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest
ENV OPERATOR_PATH=/go/src/github.com/openshift/splunk-forwarder-operator \
    OPERATOR_BIN=splunk-forwarder-operator

WORKDIR /root/
COPY --from=builder /go/src/github.com/openshift/splunk-forwarder-operator/build/_output/bin/${OPERATOR_BIN} /usr/local/bin/${OPERATOR_BIN}
LABEL io.openshift.managed.name="splunk-forwarder-operator" \
      io.openshift.managed.description="This operator will be responsible for deploying the splunk forwarder."
