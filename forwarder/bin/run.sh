#!/usr/bin/env bash

cd /opt/splunkforwarder

SPLUNK_ARGS="--answer-yes --gen-and-print-passwd --nodaemon"

if [[ ${SPLUNK_ACCEPT_LICENSE} == "yes" ]]; then
    SPLUNK_ARGS="${SPLUNK_ARGS} --accept-license"
fi

./bin/splunk start ${SPLUNK_ARGS}

# The above command still forks to the background even with --nodaemon so 
# we do the tried and true while true sleep
while true; do
    SPLUNK_PID=$(head -1 /opt/splunkforwarder/var/run/splunk/splunkd.pid)
    ps -p $SPLUNK_PID > /dev/null || exit -1
    sleep 5;
done
