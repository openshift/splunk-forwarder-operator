#!/usr/bin/env bash

cd /opt/splunkforwarder

SPLUNK_ARGS="--answer-yes --gen-and-print-passwd --nodaemon"

if [[ ${SPLUNK_ACCEPT_LICENSE} == "yes" ]]; then
    SPLUNK_ARGS="${SPLUNK_ARGS} --accept-license"
fi

cat >>etc/system/local/inputs.conf <<-EOF
    # disable forwarding splunk-internal logs
    [monitor://\$SPLUNK_HOME/var/log/splunk]
    disabled = true
    [monitor://\$SPLUNK_HOME/var/log/splunk/splunkd.log]
    disabled = true
    [monitor://\$SPLUNK_HOME/var/log/splunk/metrics.log]
    disabled = true
    [monitor://\$SPLUNK_HOME/var/log/introspection]
    disabled = true
    [monitor://\$SPLUNK_HOME/var/log/splunk/splunk_instrumentation_cloud.log*]
    disabled = true
    [batch://\$SPLUNK_HOME/var/run/splunk/search_telemetry/*search_telemetry.json]
    disabled = true
EOF

./bin/splunk start ${SPLUNK_ARGS}

# The above command still forks to the background even with --nodaemon so
# we do the tried and true while true sleep
SPLUNK_PID_FILE="/opt/splunkforwarder/var/run/splunk/splunkd.pid"
while true; do
    if [[ ! -f $SPLUNK_PID_FILE ]]; then
        exit 1
    fi

    SPLUNK_PID=$(head -1 ${SPLUNK_PID_FILE})
    ps -p $SPLUNK_PID > /dev/null || exit 1

    # Clean up old metric logs
    ls /opt/splunkforwarder/var/log/splunk/metrics.log.* && rm -f ls /opt/splunkforwarder/var/log/splunk/metrics.log.*

    sleep 5;
done
