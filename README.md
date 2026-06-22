# Solace Kubernetes Deployment (with Operator) Overview

> [!IMPORTANT]
> Please use the below scripts at your own risk. They are unsupported and is not a product of Solace. Subscription services from Solace Professional Services can purchased if required.

This project/folder contains a series of scripts that will help ease deployment tasks/activities in Kuberenetes or equivalent container platforms.

These scripts are created so you do not need to handcraft deployment yaml and instead only need to specify the correct values for all necessary variables.

The scripts are also meant to be used in multi-environment × multi-broker deployment scenario.

The scripts are organized in a general cardinal order - please see file organization below.

# File Organization System

Scripts are prefixed with a 3-digit number system:

| Headers: | Category | Action | Sequence Number | 
| --------: | -------- | -------| --------------- |
| Explanation: | `0` Create/Deploy <br> `1` Delete/Remove | `0` Pre-deployment checks <br> `1` Pre-deployment actions <br> `2` Deployment actions <br> `5` Post-deployment configurations/steps <br> `6` Post-deployment checks/verification | This represents (but not necessary always true) the general sequence of steps to execute. Some steps might be optional depending on environment |

> [!NOTE]
> Delete/Remove Category will have Action in reversed order


# How to use

## Prepare configuration file(s)

1. Duplicate `sample` file in `env` folder to an appropriate name e.g. `dev` or `dev.dns.hostname`. This will be used as the `--env` identifier (might want to choose something simpler for less typing)
2. Fill in the values for the relevant variables - see below "Configuration Variables" for explanation

## New Deployment

1. Run `001-check-env.sh` to ensure all mandatory variables are filled, and the values are expected (some variables have default values to make things work)
2. Run `009-check-storage-class.sh` to ensure storage class satisfy Solace storage requirements.
3. Run `010-deploy-operator.sh` to deploy Solace Event Broker Operator
4. Run the scripts in this order - omit steps where not required:
   - `011-create-namespace.sh`
   - `012-create-secrets.sh`
   - `020-deploy-broker.sh`
   - `050-assert-leader.sh` (if HA deployment. Script will prompt either ways)
   - `051-load-server-cert.sh` (loads/updates the TLS server certificate: updates the `$SOLBK_SVR_SECRET` secret if set, otherwise applies it via CLI)
   - `052-load-domain-certs.sh` (if required)
   - `059-execute-cli.sh`
   - `060-test-semp-login.sh`
   - `061-test-redundancy.sh`
   - `069-gather-configs.sh` (if you want to collect broker information and gather diagnostics)
  
## Redeployment / Delete Deployment

  1. Do the same as Step 1 & Step 2 as "New Deployment"
  2. Run the scripts in this order
     - `120-delete-broker.sh`
     - `112-delete-secrets.sh` (not required if you plan to execute `111-delete-namespace.sh`)
     - `111-delete-namespace.sh` (exercise caution as all resources e.g. pvc, secrets, deployments, services, etc. will be removed too)
    
# Configuration Variables

## Common / Kubernetes / Environment

| Variable | Default Value | Mandatory | Description |
| - | - | - | - |
| `KUBE` | `kubectl` | Yes | `kubectl` or `oc` command, can be full path or include other parameters required to execute `kubectl` as a profile |
| `IMAGEREPO_SECRET` | (none) | No | Name of secret to create if credentials are required to pull image from repository. <br> Leave this blank if not required. |
| `IMAGEREPO_HOST` | (none) | No | Hostname / URL of image repository to pull images from. Leave blank if images are loaded locally instead. |
| `IMAGEREPO_USER` | (none) | No | Username to login to image repository. Leave blank if not required. |
| `IMAGEREPO_PASS` | (none) | No | Password to login to image repository. Leave blank if not required. |

## Operator

| Variable | Default Value | Mandatory | Description |
| - | - | - | - |
| `SOLOP_IMAGE` | `docker.io/solace/pubsubplus-eventbroker-operator:1.4.0` | No | Full image repo + name for Solace Operator. Specify the local repository + image name here if unable to access docker.io. |
| `SOLOP_NS` | `pubsubplus-eventbroker-system` | No | Specify the Solace Operator's namespace, otherwise the script will try to detect the namespace the operator is running in. |
| `SOLOP_WATCH_NS` | (none) | No | Specify the namespace(s) (comma-separated) to watch. If empty and `${SOLOP_WATCH_SOLBK_NS}` != true, all namespaces on the cluster will be watched/monitored (please do this with caution). |
| `SOLOP_WATCH_SOLBK_NS` | `true` | No | Include Solace Event Broker's namespace as part of `${SOLOP_WATCH_NS}`. If unspecified, default value is `true`. |

## Solace Event broker

| Variable | Default Value | Mandatory | Description |
| - | - | - | - |
| `SOLBK_NAME` | (none) | Yes | Solace Event Broker Name. This will be prefixed to all pods and statefulset deployments in kubernetes. |
| `SOLBK_NS` | (none) | Yes | Kubernetes Namespace to deploy Solace Event Broker in. |
| `SOLBK_IMAGE` | (none) | Yes | Solace Event Broker Image Name. |
| `SOLBK_IMG_TAG` | (none) | Yes | Solace Event Broker Image Tag. e.g. 10.25.0.24 |
| `SOLBK_REDUNDANCY` | `false` | Yes | Solace Event Broker Redundancy Mode. true or false |
| `SOLBK_UPDATE_STRATEGY` | `automatedRolling` | No | Pod restart strategy on broker updates: `automatedRolling` (operator performs rolling restarts automatically) or `manualPodRestart` (operator waits for manual pod deletion / user intervention). Maps to CRD field `spec.updateStrategy`. |
| `SOLBK_SCALING_MAXCONN`| `100` | No | Max Client Connections. |
| `SOLBK_SCALING_MAXPOOL` | `10000` | No | Max Spool Size (in MB). |
| `SOLBK_SCALING_MAXQMSG` | `100` | No | Max Number of Queued Messages (in millions). |
| `SOLBK_SCALING_MAXKAFKABRIDGE` | `0` | No | Max Kafka Bridge Count. |
| `SOLBK_SCALING_MAXKAFKACONN` | `0` | No | Max Kafka Broker Connection Count. |
| `SOLBK_SCALING_MAXBRIDGE` | `25` | No | Max Bridge Count. |
| `SOLBK_SCALING_MAXSUB` | `50000` | No | Max Subscription Count. |
| `SOLBK_SCALING_MAXGMSSIZE` | `10` | No | Max Guaranteed Message Size (in MB). |
| `SOLBK_PRODUCTKEYS` | (none) | No | List of product keys to apply to Solace brokers. |
| `SOLBK_MSGNODE_CPU` | `2` | No | Number of CPUs for Solace Broker messaging nodes. <br> Please refer to the Broker Resource Calculator for numbers. |
| `SOLBK_MSGNODE_MEM` | `3410Mi` | No | Amount of memory assigned for Solace Broker message node. Units are "Mi" "Gi". <br> Please refer to the Broker Resource Calculator for numbers. |
| `SOLBK_STORAGECLASS` | (none) | No | Specify the storage class to use. If left blank, `009-check-storage-class.sh` auto-detects the cluster's default StorageClass (the one annotated `storageclass.kubernetes.io/is-default-class=true`) and validates that one. |
| `SOLBK_STORAGE_MSGNODE` | (none) | Yes | Specify the disk storage size for messaging nodes. |
| `SOLBK_STORAGE_MONNODE` | `5Gi` | No | Specify the disk storage size for the monitor node. |
| `SOLBK_ADM_PASS` | `adminpassword123` | Yes | Admin password for SEMP. |
| `SOLBK_MON_PASS` | `monitorpassword123` | Yes | Monitor (read-only) password for SEMP. |
| `SOLBK_USR_PASS` | (none) | No | List of key-value pair strings ("key=value") for additional CLI users. |
| `SOLBK_USR_SECRET` | `solace-admin-secret` | Yes | Name of secret to store admin password for SEMP. |
| `SOLBK_SVR_SECRET` | (none) | No | Name of secret to store SSL/TLS Server Certificates. Leave blank if you do not want to enable SSL/TLS for the broker. |
| `SOLBK_TLS_CERT` | `cert/tls.crt` | No | Full path + filename of SSL/TLS Server Certificate. This is required if `$SOLBK_SVR_SECRET` is specified. |
| `SOLBK_TLS_CERTKEY` | `cert/tls.key` | No | Full path + filename of private key used to generate SSL/TLS Server Certificate. This is required if `$SOLBK_SVR_SECRET` is specified.  |
| `SOLBK_TLS_CERTCAS` | (none) | No | List of "full path + filenames" of Certificate Authority root/intermediate certificates. This is recommended to include as these will be appended as part of the SSL/TLS Server Certificate that will be loaded as part of `051-load-server-cert.sh` |
| `SOLBK_NODETOL_PRI` | (none) | No | Worker Node Tolerations to apply during Node Selection at initialization for Primary Node. Use this for standalone broker deployment too.<br>Please comment the variable if not required. |
| `SOLBK_NODETOL_BKP` | (none) | No | Worker Node Tolerations to apply during Node Selection at initialization for Backup Node.<br>Please comment the variable if not required. |
| `SOLBK_NODETOL_MON` | (none) | No | Worker Node Tolerations to apply during Node Selection at initialization for Monitoring Node.<br>Please comment the variable if not required. |
| `SOLBK_NODELABEL_PRI` | (none) | No | Worker Node Labels to apply during Node Selection at initialization for Primary Node. Use this for standalone broker deployment too.<br>Please comment the variable if not required. |
| `SOLBK_NODELABEL_BKP` | (none) | No | Worker Node Labels to apply during Node Selection at initialization for Backup Node.<br>Please comment the variable if not required. |
| `SOLBK_NODELABEL_MON` | (none) | No | Worker Node Labels to apply during Node Selection at initialization for Monitoring Node.<br>Please comment the variable if not required. |
| `SOLBK_ANTIAFFINITY_NS` | (none) | No | List of namespace to apply Kubernete "Anti-Affinity" rules to. This script applies `preferredDuringSchedulingIgnoredDuringExecution` for pod allocation if this is specified. |
| `SOLBK_ANTIAFFINITY_WT` | `100` | No | Weight of Anti-Affinity rule. Values range from 1-100. |
| `SOLBK_DIAG_DIR` | `diag-configs` | No | Directory to store output of `069-gather-configs.sh`. |
| `SOLBK_IPPOOL` | (none) | No | IP Address Pool Name to be used for IP assignment for Metal LB. |
| `SOLBK_LOADBALANCER_IP` | (none) | No | If static IP address is required to be specified for Metal LB. |
| `SOLBK_LOADBALANCER_ANOTN` | (none) | No | Optional annotations required for loadbalancers association. |
| `SOLBK_PORTS` | 16 standard Solace ports | No | List of service ports to publish. Format per entry: `name=port[/protocol]` where `port` is either a single number (`containerPort` == `servicePort`) or `containerPort:servicePort`, and `/protocol` is optional (defaults to `TCP`). Override fully replaces the default list — omit defaults you don't need, append custom entries. |
| `SOLBK_DOMAINCERT_FOLDER` | (none) | No | Folder containing Certificate Authority certificates to be loaded. This is used in conjunction with `$SOLBK_DOMAINCERT_FILES` |
| `SOLBK_DOMAINCERT_FILES` | (none) | No | List of Certificate Authority certificates to be loaded into the broker. This will be applied by `052-load-domain-certs.sh`. |
| `SOLBK_CLISCRIPTS_FOLDER` | `cli` | No | Path + Folder name of directory containing CLI scripts to be executed. This is used in `059-execute-cli.sh` |

# Additional Helper Scripts

These are operational helpers — they do not participate in the deploy/delete sequence. All scripts still take `--env <name>`.

## Pod-role argument

Helpers that target a specific broker pod accept the role as the first positional argument. Both the short and long forms are accepted; if omitted, `primary` is used:

| Short | Long | Pod suffix |
| - | - | - |
| `p` | `primary` | `-pubsubplus-p-0` |
| `b` | `backup` | `-pubsubplus-b-0` |
| `m` | `monitor` | `-pubsubplus-m-0` |

```bash
./desc-broker.sh --env dev backup
./logs-broker.sh --env dev primary -f --tail=200
./enter-solace-cli.sh --env dev m
```

Normalization happens in the shared `pick_pod` function in `000-env.sh`.

## Inspection

| Script | Purpose |
| - | - |
| `desc-broker.sh [role]` | `kubectl describe pod` of a broker pod |
| `desc-op.sh` | `kubectl describe pod` of the operator pod |
| `desc-lb.sh` | `kubectl describe svc` of the broker LoadBalancer |
| `get-broker-status.sh` | Show broker pod/PVC/svc status |
| `get-op-status.sh` | Show operator deployment status |
| `show-all-brokers.sh` | List broker pods, services, statefulsets across all namespaces |
| `logs-broker.sh [role] [kubectl logs args...]` | Tail broker logs; extra args are forwarded to `kubectl logs` |
| `logs-op.sh [kubectl logs args...]` | Tail operator logs |
| `watch.sh` | Loop `get-op-status.sh` and `get-broker-status.sh` |

## Interactive access

| Script | Purpose |
| - | - |
| `enter-solace-cli.sh [role]` | Drop into the Solace CLI (`cli -A`) inside the chosen pod |
| `enter-solace-shell.sh [role]` | Drop into a bash shell inside the chosen pod |
| `copy-files-from-broker.sh [role] <path>...` | `kubectl cp` files out of the broker pod |
| `copy-files-into-broker.sh [role] <local-path>... <pod-dir>` | `kubectl cp` files into the broker pod |

## Lifecycle

| Script | Purpose |
| - | - |
| `replicas-start-broker.sh` | Scale broker statefulsets to 1 (start) |
| `replicas-stop-broker.sh` | Scale broker statefulsets to 0 (stop) — prompts for confirmation |
