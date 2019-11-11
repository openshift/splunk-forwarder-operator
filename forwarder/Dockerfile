FROM registry.access.redhat.com/ubi8/ubi-minimal

ARG VERSION
ARG VERSION_HASH

ADD forwarder/bin/run.sh /
RUN microdnf install -y libsemanage shadow-utils findutils procps wget && \
    microdnf clean all && \
    wget -O /tmp/splunkforwarder-${VERSION_FULL}-linux-2.6-x86_64.rpm "https://www.splunk.com/bin/splunk/DownloadActivityServlet?architecture=x86_64&platform=linux&version=${VERSION}&product=universalforwarder&filename=splunkforwarder-${VERSION}-${VERSION_HASH}-linux-2.6-x86_64.rpm&wget=true" && \
    rpm -ivh /tmp/splunkforwarder-${VERSION_FULL}-linux-2.6-x86_64.rpm && \
    rm -f /tmp/splunkforwarder-${VERSION_FULL}-linux-2.6-x86_64.rpm && \
    mkdir -p /host && chown splunk:splunk /host && \
    mkdir -p /opt/splunkforwarder/var/{lib,run,spool}/splunk && \
    mkdir -p /opt/splunkforwarder/var/lib/splunk/{authDb,hashDb} && \
    mkdir -p /opt/splunkforwarder/var/run/splunk/appserver/{i18n,modules/static/css} && \
    mkdir -p /opt/splunkforwarder/var/run/splunk/{upload,search_telemetry} && \
    mkdir -p /opt/splunkforwarder/var/spool/dirmoncache && \
    mkdir -p /opt/splunkforwarder/etc/apps/osd_monitored_logs/{local,metadata} && \
    mkdir -p /opt/splunkforwarder/etc/apps/splunkauth/{default,local,metadata} && \
    chown -R splunk:splunk /opt/splunkforwarder && \
    chown -R splunk:splunk /run.sh && \
    chmod +x /run.sh

USER splunk
CMD [ "/run.sh" ]
