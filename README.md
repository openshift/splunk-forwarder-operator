# splunk-forwarder-operator

This operator manages [Splunk Universal Forwarder](https://docs.splunk.com/Documentation/Forwarder/latest/Forwarder/Abouttheuniversalforwarder). It deploys a daemonset which 
deploys a pod on each node including the masters. It expects the service account
for the namespace can deploy privileged pods. It also needs a secret that holds
the forwarder auth.

If you are using [Splunk Cloud](https://www.splunk.com/en_us/software/splunk-cloud.html), credentials can be obtained by
downloading a credentials package from the specific Splunk application being used, such as the Universal Forwarder app.
The credentials package is a tarball, so first extract the contents with `tar xvf splunkclouduf.spl`, then add the
following fields in outputs.conf
```
sslCertPath = $SPLUNK_HOME/etc/apps/splunkauth/default/server.pem
sslRootCAPath = $SPLUNK_HOME/etc/apps/splunkauth/default/cacert.pem
sslPassword = <Your SSL Password>
```

Then create a secret named "splunk-auth" using the extracted spl files and modified outputs.conf:
```
oc create secret generic splunk-auth --dry-run=client -o yaml \
  --from-file=cacert.pem=/path/to/spl/cacert.pem \
  --from-file=limits.conf=/path/to/spl/limits.conf \
  --from-file=outputs.conf=/path/to/spl/outputs.conf \
  --from-file=server.pem=/path/to/spl/server.pem
```

The SplunkForwarder CRD explicitly points to the files you want to monitor (currently only supports monitor://).

```yaml
apiVersion: splunkforwarder.managed.openshift.io/v1alpha1
kind: SplunkForwarder
metadata:
  name: example-splunkforwarder
spec:
  image: dockerimageurl
  imageDigest: sha256:afc413cd2504b586b6f28c16355db243c8bfdef536cc80e44f61e03c01e12235
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
(The CRD supports `imageTag` for both, but this is deprecated in favor of `imageDigest`.)

To use the current version, `9.0.0.1-9e907cedecb1-72787d0`, specify the following:
- For [splunk-forwarder](https://quay.io/repository/app-sre/splunk-forwarder?tag=8.2.5-77015bc7a462-f4d16f7):
  ```yaml
  image: quay.io/app-sre/splunk-forwarder
  imageDigest: sha256:afc413cd2504b586b6f28c16355db243c8bfdef536cc80e44f61e03c01e12235
  ```
- For [splunk-heavyforwarder](https://quay.io/repository/app-sre/splunk-heavyforwarder?tag=8.2.5-77015bc7a462-f4d16f7):
  ```yaml
  heavyForwarderImage: quay.io/app-sre/splunk-heavyforwarder
  heavyForwarderDigest: sha256:083847a013e4e29db923b03d5c3c1f21f3b250063b51794e5cc00c24ceb3a8b2
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
OLM registry, you must also create them for `splunk-forwarder` and `splunk-heavyforwarder`.
