#!/usr/bin/env bash

cd /opt/splunkforwarder

SPLUNK_ARGS="--answer-yes --gen-and-print-passwd --nodaemon"

if [[ ${SPLUNK_ACCEPT_LICENSE} == "yes" ]]; then
    SPLUNK_ARGS="${SPLUNK_ARGS} --accept-license"
fi

./bin/splunk start ${SPLUNK_ARGS}

# The above command still forks to the background even with --nodaemon so 
# we do the tried and true while true sleep
SPLINK_PID_FILE="/opt/splunkforwarder/var/run/splunk/splunkd.pid"
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
