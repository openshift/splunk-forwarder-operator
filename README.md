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
  imageDigest: sha256:e4c55593996c7b0f3303ff31f391aa296a9af6b1da945d9c63a1807ea257839d
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

The `image` and `imageDigest` are for the splunk-forwarder image.
If `useHeavyForwarder` is `true`, `heavyForwarderImage` and `heavyForwarderDigest` are used for the splunk-heavyforwarder image.
(The CRD supports `imageTag` for both, but this is deprecated.)

To use the current version, `8.2.3-cd0848707637-661ed09`, specify the following:
- For [splunk-forwarder](https://quay.io/repository/app-sre/splunk-forwarder?tag=8.2.3-cd0848707637-661ed09&tab=tags):
  ```yaml
  image: quay.io/app-sre/splunk-forwarder
  imageDigest: sha256:e4c55593996c7b0f3303ff31f391aa296a9af6b1da945d9c63a1807ea257839d
  ```
- For [splunk-heavyforwarder](https://quay.io/repository/app-sre/splunk-heavyforwarder?tag=8.2.3-cd0848707637-661ed09&tab=tags):
  ```yaml
  heavyForwarderImage: quay.io/app-sre/splunk-heavyforwarder
  heavyForwarderDigest: sha256:ab8bfc8a9fd41db2e40d12eef2de6a9a7b5b0d319d5ecf67c62bfb18ba1501b3
  ```

## Upgrading Splunk Universal Forwarder

Run `make image-update` to update to the current master branch commit of [splunk-forwarder-images](https://github.com/openshift/splunk-forwarder-images/).

This process will update the Makefile with a new value for `FORWARDER_IMAGE_TAG` (from the [forwarder version](https://github.com/openshift/splunk-forwarder-images/blob/master/.splunk-version), [forwarder hash](https://github.com/openshift/splunk-forwarder-images/blob/master/.splunk-version-hash) and [commit hash](https://github.com/openshift/splunk-forwarder-images/blob/fa50892e3ea29cb19e34b287ac4a5dd42aab45ec/Makefile#L14)) and populate the [OLM template](hack/olm-registry/olm-artifacts-template.yaml) with the by-digest URIs for [that version](https://github.com/openshift/splunk-forwarder-images/#versioning-and-tagging).

To use a specific version, use `make SFI_UPDATE=<commit/branch/etc> image-update` or edit the Makefile by hand and run `make image-digests` to update the OLM template.

Commit and propose the changes as usual.

## Testing the app-sre pipeline

This repository is configured to support the testing strategy documented
[here](https://github.com/openshift/boilerplate/blob/cc252374715df1910c8f4a8846d38e7b5d00f94f/boilerplate/openshift/golang-osd-operator/app-sre.md).

Note that, in addition to creating personal repositories for the operator and
OLM registry, you must also create them for `splunk-forwarder` and
`splunk-heavyforwarder`.
