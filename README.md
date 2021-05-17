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
  imageDigest: "sha256:8c4a2a8bb186c6b0ea744fd4c05df61d2c50053ebf42a0a6ec7aef8170be4c55"
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

To use the current version, `8.0.5-a1a6394cc5ae`, specify the following:
- For [splunk-forwarder](https://quay.io/repository/app-sre/splunk-forwarder?tag=8.0.5-a1a6394cc5ae&tab=tags):
  ```yaml
  image: quay.io/app-sre/splunk-forwarder
  imageDigest: sha256:2452a3f01e840661ee1194777ed5a9185ceaaa9ec7329ed364fa2f02be22a701
  ```
- For [splunk-heavyforwarder](https://quay.io/repository/app-sre/splunk-heavyforwarder?tag=8.0.5-a1a6394cc5ae&tab=tags):
  ```yaml
  heavyForwarderImage: quay.io/app-sre/splunk-heavyforwarder
  heavyForwarderDigest: sha256:49b40c2c5d79913efb7eff9f3bf9c7348e322f619df10173e551b2596913d52a
  ```

## Upgrading Splunk Universal Forwarder
To use a different version of Splunk Universal Forwarder
1. Make sure the [splunk-forwarder-images](https://github.com/openshift/splunk-forwarder-images/) repository has [built the desired version](https://github.com/openshift/splunk-forwarder-images/#versioning-and-tagging).
2. Edit the [version](.splunk-version) and [hash](.splunk-version-hash) files to register the desired version.
3. Run `make image-digests`.
   This will populate the [OLM template](hack/olm-registry/olm-artifacts-template.yaml) with the by-digest URIs for the registered version.
4. Edit the version and digest strings in the [section above](#splunk-forwarder-operator) to keep them in sync with the version files and the OLM template.
5. Commit and propose the changes as usual.

## Testing the app-sre pipeline

This repository is configured to support the testing strategy documented
[here](https://github.com/openshift/boilerplate/blob/cc252374715df1910c8f4a8846d38e7b5d00f94f/boilerplate/openshift/golang-osd-operator/app-sre.md).

Note that, in addition to creating personal repositories for the operator and
OLM registry, you must also create them for `splunk-forwarder` and
`splunk-heavyforwarder`.
