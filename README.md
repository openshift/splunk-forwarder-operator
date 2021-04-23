# splunk-forwarder-operator

This operator manages [Splunk Universal Forwarder](https://docs.splunk.com/Documentation/Forwarder/latest/Forwarder/Abouttheuniversalforwarder). It deploys a daemonset which 
deploys a pod on each node including the masters. It expects the service account
for the namespace can deploy privileged pods. It also needs a secret that holds
the forwarder auth.

If you are using [splunk cloud](https://www.splunk.com/en_us/software/splunk-cloud.html) you can download the spl file, extract it with
`tar xvf splunkclouduf.spl` then edit outputs.conf and change sslCertPath and
sslRootCAPath to point to the directory `$SPLUNK_HOME/etc/apps/splunkauth/default/`
create a secret with the files as is and they will be mounted in the correct place. 

The CRD is very to point to the files you want to ship(currently only supports
monitor://).

```json
apiVersion: splunkforwarder.managed.openshift.io/v1alpha1
kind: SplunkForwarder
metadata:
  name: example-splunkforwarder
spec:
  image: dockerimageurl
  imageTag: "versiontag"
  splunkLicenseAccepted: true
  clusterID: optional-cluster-name
  splunkInputs:
  - path: /host/var/log/openshift-apiserver/audit.log
    index: openshift_managed_audit
    whitelist: \.log$
    sourcetype: _json
  - path: /host/var/log/containers/ip-*-*-*-*ec2internal-debug*.log
    index: openshift_managed_debug_node
    whitelist: \.log$
    sourcetype: _json
```

The image and imageTag are for the image in /forwarder (currently version 
8.0.5-a1a6394cc5ae)

## Testing the app-sre pipeline

This repository is configured to support the testing strategy documented
[here](https://github.com/openshift/boilerplate/blob/cc252374715df1910c8f4a8846d38e7b5d00f94f/boilerplate/openshift/golang-osd-operator/app-sre.md).

Note that, in addition to creating personal repositories for the operator and
OLM registry, you must also create them for `splunk-forwarder` and
`splunk-heavyforwarder`.
