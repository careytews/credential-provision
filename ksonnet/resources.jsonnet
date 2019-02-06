
//
// Definition for Cassandra resources on Kubernetes.  This creates a Cassandra
// cluster.
//

// Import KSonnet library.
local k = import "ksonnet.beta.2/k.libsonnet";
local tnw = import 'lib/tnw-common.libsonnet';

// Short-cuts to various objects in the KSonnet library.
local depl = k.extensions.v1beta1.deployment;
local container = depl.mixin.spec.template.spec.containersType;
local mount = container.volumeMountsType;
local volume = depl.mixin.spec.template.spec.volumesType;
local resources = container.resourcesType;
local env = container.envType;
local pvcVol = volume.mixin.persistentVolumeClaim;
local secretDisk = volume.mixin.secret;
local svc = k.core.v1.service;
local pvc = k.core.v1.persistentVolumeClaim;
local sc = k.storage.v1.storageClass;

local credentialMgmt(config) = {

    local version = import "version.jsonnet",

    name: "credential-mgmt",
    images: [
        config.containerBase + "/credential-mgmt:" + version
    ],

    // Environment
    envs:: [
        env.new("WEB_CA", "/web_ca"),
        env.new("WEB_CA_CERT", "/web_ca_cert"),
        env.new("PROBE_CA", "/probe_ca"),
        env.new("PROBE_CA_CERT", "/probe_ca_cert"),
        env.new("VPN_CA", "/vpn_ca"),
        env.new("VPN_CA_CERT", "/vpn_ca_cert"),
        env.new("KEY", "/key/private.json"),

        // We write to the trust-networks-credentials bucket in the
        // trust-networks project.
        env.new("PROJECT_ID", "trust-networks"),
        env.new("BUCKET", "trust-networks-credentials"),
        env.new("KEY_RING", "user-secrets"),

        env.new("SERVICE_ACCOUNT", config.accounts["credential-mgmt"]),
        env.new("CRL_BUCKET", "%s" % [config.urls.crlDistPointAddress]),

        env.new("PUBSUB_PROJECT", config.project),
        env.new("PUBSUB_REQUEST_TOPIC", config.credential_request_topic),
        env.new("PUBSUB_RESPONSE_TOPIC", config.credential_response_topic),
        env.new("PUBSUB_SUBSCRIPTION", config.credential_subscription)

    ],

    // Volume mount points
    volumeMounts:: [

        // Cloud account private key, used to access GCP services.
        mount.new("keys", "/key") + mount.readOnly(true),

        // CA credentials for VPN, web, probe
        mount.new("vpn-ca-creds", "/vpn_ca_cert") + mount.readOnly(true),
        mount.new("web-ca-creds", "/web_ca_cert") + mount.readOnly(true),
        mount.new("probe-ca-creds", "/probe_ca_cert") + mount.readOnly(true),

        // CA records i.e. certs which have been allocated.
        mount.new("ca-data", "/vpn_ca") + mount.subPath("vpn_ca"),
        mount.new("ca-data", "/web_ca") + mount.subPath("web_ca"),
        mount.new("ca-data", "/probe_ca") + mount.subPath("probe_ca")

    ],

    // Container definition.
    containers:: [

        container.new("credential-mgmt", self.images[0]) +
            container.env(self.envs) +
            container.volumeMounts(self.volumeMounts) +
            container.mixin.resources.limits({
                memory: "64M", cpu: "1.0"
            }) +
            container.mixin.resources.requests({
                memory: "64M", cpu: "0.05"
            })

    ],

    // Volumes - this invokes a secret containing the cert/key
    volumes:: [

        // Cloud account private key, used to access GCP services.
        volume.name("keys") + secretDisk.secretName("credential-mgmt-keys"),

        // CA credentials for VPN, web, probe
        volume.name("vpn-ca-creds") + secretDisk.secretName("vpn-ca-creds"),
        volume.name("web-ca-creds") + secretDisk.secretName("web-ca-creds"),
        volume.name("probe-ca-creds") + secretDisk.secretName("probe-ca-creds"),

        // CA records i.e. certs which have been allocated.  These are stored
        // on permanent disks.
        volume.name("ca-data") + pvcVol.claimName("ca-data")

    ],

    // Deployment definition.  id is the node ID.
    deployments:: [
        depl.new("credential-mgmt", 1, self.containers,
                 {app: "credential-mgmt", component: "frontend"}) +
            depl.mixin.spec.template.spec.volumes(self.volumes) +
            depl.mixin.metadata.namespace(config.namespace)
    ],

    storageClasses:: [
        sc.new() + sc.mixin.metadata.name("credential-mgmt") +
            config.storageParams.hot +
            { reclaimPolicy: "Retain" } +
            sc.mixin.metadata.namespace(config.namespace)
            
    ],

    pvcs:: [
        tnw.pvc("ca-data", "credential-mgmt", 20, config.namespace)
    ],

    // Function which returns resource definitions - deployments and services.
    resources:
        if config.options.includeAnalytics then
            self.deployments + self.pvcs + self.storageClasses
        else [],

};

// Return the function which creates resources.
[credentialMgmt]
