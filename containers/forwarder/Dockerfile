FROM registry.access.redhat.com/ubi8/ubi-minimal

ADD containers/forwarder/bin/run.sh /
ADD .splunk-version /
ADD .splunk-version-hash /
RUN export VERSION=$(cat /.splunk-version) && export VERSION_HASH=$(cat /.splunk-version-hash) && \
    microdnf install -y libsemanage shadow-utils findutils procps wget && \
    microdnf clean all && \
    wget -O /tmp/splunkforwarder.rpm "https://www.splunk.com/bin/splunk/DownloadActivityServlet?architecture=x86_64&platform=linux&version=${VERSION}&product=universalforwarder&filename=splunkforwarder-${VERSION}-${VERSION_HASH}-linux-2.6-x86_64.rpm&wget=true" && \
    rpm -ivh /tmp/splunkforwarder.rpm && \
    rm -f /tmp/splunkforwarder.rpm && \
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
    chmod +x /run.sh && \
    sed -i 's/\.maxBackupIndex=5/\.maxBackupIndex=1/g' /opt/splunkforwarder/etc/log.cfg && \
    sed -i 's/\.maxFileSize=25000000/\.maxFileSize=250000/g' /opt/splunkforwarder/etc/log.cfg

USER splunk
CMD [ "/run.sh" ]
