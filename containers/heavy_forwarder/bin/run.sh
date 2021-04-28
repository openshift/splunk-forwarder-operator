#!/usr/bin/env bash

cd /opt/splunk

SPLUNK_ARGS="--answer-yes --gen-and-print-passwd"

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

# Switch to forwarder license
./bin/splunk edit licenser-groups Forwarder -is_active 1 ${SPLUNK_ARGS}

./bin/splunk start --nodaemon

# The above command still forks to the background even with --nodaemon so
# we do the tried and true while true sleep
SPLUNK_PID_FILE="/opt/splunk/var/run/splunk/splunkd.pid"
while true; do
    if [[ ! -f $SPLUNK_PID_FILE ]]; then
        exit 1
    fi

    SPLUNK_PID=$(head -1 ${SPLUNK_PID_FILE})
    ps -p $SPLUNK_PID > /dev/null || exit 1

    # Clean up old metric logs
    ls /opt/splunk/var/log/splunk/metrics.log.* && rm -f ls /opt/splunk/var/log/splunk/metrics.log.*

    sleep 5;
done
