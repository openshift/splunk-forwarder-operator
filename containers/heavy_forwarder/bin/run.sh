#!/usr/bin/env bash

set -o xtrace

id

ls -la /opt/

cd /opt/splunk

SPLUNK_ARGS="--answer-yes --gen-and-print-passwd"

if [[ ${SPLUNK_ACCEPT_LICENSE} == "yes" ]]; then
    SPLUNK_ARGS="${SPLUNK_ARGS} --accept-license"
fi

# Switch to forwarder license
./bin/splunk edit licenser-groups Forwarder -is_active 1 ${SPLUNK_ARGS}

./bin/splunk start --nodaemon

# The above command still forks to the background even with --nodaemon so 
# we do the tried and true while true sleep
SPLINK_PID_FILE="/opt/splunk/var/run/splunk/splunkd.pid"
while true; do
    if [[ ! -f $SPLUNK_PID_FILE ]]; then
        exit 1
    fi

    SPLUNK_PID=$(head -1 ${SPLUNK_PID_FILE})
    ps -p $SPLUNK_PID > /dev/null || exit 1
    sleep 5;
done

echo "past the loop: unexpected"
sleep 120

