FROM registry.access.redhat.com/ubi8/ubi-minimal

ADD containers/heavy_forwarder/bin/run.sh /
ADD .splunk-version /
ADD .splunk-version-hash /
RUN export VERSION=$(cat /.splunk-version) && export VERSION_HASH=$(cat /.splunk-version-hash) && \
    curl -Lo /tmp/splunk.rpm "https://www.splunk.com/bin/splunk/DownloadActivityServlet?architecture=x86_64&platform=linux&version=${VERSION}&product=splunk&filename=splunk-${VERSION}-${VERSION_HASH}-linux-2.6-x86_64.rpm&wget=true" && \
    microdnf install -y findutils libsemanage procps-ng shadow-utils && \
    rpm -ivh /tmp/splunk.rpm && \
    microdnf clean all && \
    rm -f /tmp/splunk.rpm && \
    mkdir -p /host && chown splunk:splunk /host && \
    mkdir -p /opt/splunk/var/{lib,run,spool}/splunk && \
    mkdir -p /opt/splunk/var/lib/splunk/{authDb,hashDb} && \
    mkdir -p /opt/splunk/var/run/splunk/appserver/{i18n,modules/static/css} && \
    mkdir -p /opt/splunk/var/run/splunk/{upload,search_telemetry} && \
    mkdir -p /opt/splunk/var/spool/dirmoncache && \
    mkdir -p /opt/splunk/etc/apps/osd_monitored_logs/{local,metadata} && \
    mkdir -p /opt/splunk/etc/apps/splunkauth/{default,local,metadata} && \
    chown -R splunk:splunk /opt/splunk && \
    chown -R splunk:splunk /run.sh && \
    chmod +x /run.sh && \
    echo -e "\nOPTIMISTIC_ABOUT_FILE_LOCKING = 1" >>  /opt/splunk/etc/splunk-launch.conf && \
    sed -i 's/startwebserver = 1/startwebserver = 0/' /opt/splunk/etc/system/default/web.conf && \
    sed -i 's/\.maxBackupIndex=5/\.maxBackupIndex=1/g' /opt/splunk/etc/log.cfg && \
    sed -i 's/\.maxFileSize=25000000/\.maxFileSize=250000/g' /opt/splunk/etc/log.cfg

USER splunk
CMD [ "/run.sh" ]
