FROM registry.svc.ci.openshift.org/openshift/release:golang-1.12 AS builder
ENV OPERATOR_PATH=/go/src/github.com/openshift/splunk-forwarder-operator
COPY . ${OPERATOR_PATH}
WORKDIR ${OPERATOR_PATH}
RUN make gobuild

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest
ENV OPERATOR_PATH=/go/src/github.com/openshift/splunk-forwarder-operator \
    OPERATOR_BIN=splunk-forwarder-operator

WORKDIR /root/
COPY --from=builder ${OPERATOR_PATH}/build/_output/bin/${OPERATOR_BIN} /usr/local/bin/${OPERATOR_BIN}
LABEL io.openshift.managed.name="splunk-forwarder-operator" \
      io.openshift.managed.description="This operator will be responsible for deploying the splunk forwarder."