FROM registry.access.redhat.com/ubi8/ubi-minimal

ADD bin/run.sh /
RUN microdnf install -y libsemanage shadow-utils findutils procps wget && \
    microdnf clean all && \
    wget -O /tmp/splunkforwarder-7.3.2-c60db69f8e32-linux-2.6-x86_64.rpm 'https://www.splunk.com/bin/splunk/DownloadActivityServlet?architecture=x86_64&platform=linux&version=7.3.2&product=universalforwarder&filename=splunkforwarder-7.3.2-c60db69f8e32-linux-2.6-x86_64.rpm&wget=true' && \
    rpm -ivh /tmp/splunkforwarder-7.3.2-c60db69f8e32-linux-2.6-x86_64.rpm && \
    rm -f /tmp/splunkforwarder-7.3.2-c60db69f8e32-linux-2.6-x86_64.rpm && \
    mkdir -p /host && chown splunk:splunk /host && \
    mkdir -p /opt/splunkforwarder/var/{lib,run,spool}/splunk && \
    mkdir -p /opt/splunkforwarder/var/lib/splunk/{authDb,hashDb} && \
    mkdir -p /opt/splunkforwarder/var/run/splunk/appserver/{i18n,modules/static/css} && \
    mkdir -p /opt/splunkforwarder/var/run/splunk/{upload,search_telemetry} && \
    mkdir -p /opt/splunkforwarder/var/spool/dirmoncache && \
    mkdir -p /opt/splunkforwarder/etc/splunk/apps/osd_monitored_logs/{local,metadata} && \
    mkdir -p /opt/splunkforwarder/etc/splunk/apps/splunkauth/{default,local,metadata} && \
    chown -R splunk:splunk /opt/splunkforwarder && \
    chown -R splunk:splunk /run.sh && \
    chmod +x /run.sh

USER splunk
CMD [ "/run.sh" ]
