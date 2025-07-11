# Solace Kubernetes Deployment (with Operator) Overview

This project/folder contains a series of scripts that will help ease deployment tasks/activities in Kuberenetes or equivalent container platforms.

These scripts are created so you do not need to handcraft deployment yaml and instead only need to specify the correct values for all necessary variables.

The scripts are also meant to be used in multi-environment × multi-broker deployment scenario.

The scripts are organized in a general cardinal order - please see file organization below.

# File Organization System

Scripts are prefixed with a 3-digit number system:

| Headers: | Category | Action | Sequence Number | 
| --------: | -------- | -------| --------------- |
| Explanation: | `0` Create/Deploy <br> `1` Delete/Remove | `0` Pre-deployment checks <br> `1` Pre-deployment actions <br> `2` Deployment actions <br> `3` Post-deployment checks <br> `5` Post-deployment configurations/steps <br> `6` Post-deployment verification/diagnostics | This represents (but not necessary always true) the general sequence of steps to execute. Some steps might be optional depending on environment |

> [!NOTE]
> Delete/Remove Category will have Action in reversed order


# How to use

## Prepare configuration file(s)

1. Duplicate `sample` file in `env` folder to an appropriate name e.g. `dev` or `dev.dns.hostname`. This will be used as the `--env` identifier (might want to choose something simpler for less typing)
2. Fill in the values for the relevant variables - see below "Configuration Variables" for explanation

## New Deployment

1. Run `001-check-env.sh` to ensure all mandatory variables are filled, and the values are expected (some variables have default values to make things work)
2. Run the scripts in this order - omit steps where not required:
   - `011-create-namespace.sh`
   - `012-create-secrets.sh`
   - (TODO include operator scripts)
   - `019-check-storage-class.sh`
   - `020-deploy-broker.sh`
   - `029-assert-leader.sh` (if HA deployment. Script will prompt either ways)
   - `050-load-server-cert.sh` (if SSL/TLS Certificates are not specified as part of deployment yaml)
   - `051-load-domain-certs.sh` (if required)
   - `059-execute-cli.sh`
   - `060-test-semp-login.sh`
   - `061-gather-configs.sh` (if you want to collect broker information and gather diagnostics)
  
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

(TODO)

## Solace Event broker

| Variable | Default Value | Mandatory | Description |
| - | - | - | - |
| `SOLBK_NAME` | (none) | Yes | Solace Event Broker Name. This will be prefixed to all pods and statefulset deployments in kubernetes. |
| `SOLBK_NS` | (none) | Yes | Kubernetes Namespace to deploy Solace Event Broker in. |
| `SOLBK_IMAGE` | (none) | Yes | Solace Event Broker Image Name. |
| `SOLBK_IMG_TAG` | (none) | Yes | Solace Event Broker Image Tag. e.g. 10.25.0.24 |
| `SOLBK_REDUNDANCY` | false | Yes | Solace Event Broker Redundancy Mode. true or false |
| `SOLBK_SCALING_MAXCONN`| 100 | No | Solace Event Broker Scaling Tier Parameter - Max Client Connections. |
| `SOLBK_SCALING_MAXPOOL` | 10000 | No | Solace Event Broker Scaling Tier Parameter - Max Spool Size (in MB). |
| `SOLBK_SCALING_MAXQMSG` | 100 | No | Solace Event Broker Scaling Tier Parameter - Max Number of Queued Messages (in millions). |
| `SOLBK_MSGNODE_CPU` | 2 | No | Number of CPUs for Solace Broker messaging nodes. <br> Please refer to the Broker Resource Calculator for numbers. |
| `SOLBK_MSGNODE_MEM` | 3410Mi | No | Amount of memory assigned for Solace Broker message node. Units are "Mi" "Gi". <br> Please refer to the Broker Resource Calculator for numbers. |
| `SOLBK_STORAGECLASS` | (none) | No | Specify the storage class to use if required. |
| `SOLBK_STORAGE_MSGNODE` | (none) | Yes | Specify the disk storage size for messaging nodes. |
| `SOLBK_STORAGE_MONNODE` | 5Gi | No | Specify the disk storage size for the monitor node. |
| `SOLBK_ADM_PASS` | adm1nPA@55w0rD | Yes | Admin password for SEMP. |
| `SOLBK_ADM_SECRET` | solace-admin-secret | Yes | Name of secret to store admin password for SEMP. |
| `SOLBK_SVR_SECRET` | (none) | No | Name of secret to store SSL/TLS Server Certificates. Leave blank if you do not want to enable SSL/TLS for the broker. |
| `SOLBK_TLS_CERT` | tls.crt | No | Full path + filename of SSL/TLS Server Certificate. This is required if `$SOLBK_SVR_SECRET` is specified. |
| `SOLBK_TLS_CERTKEY` | tls.key | No | Full path + filename of private key used to generate SSL/TLS Server Certificate. This is required if `$SOLBK_SVR_SECRET` is specified.  |
| `SOLBK_TLS_CERTCAS` | (none) | No | List of "full path + filenames" of Certificate Authority root/intermediate certificates. This is recommended to include as these will be appended as part of the SSL/TLS Server Certificate that will be loaded as part of `050-load-server-cert.sh` |
| `SOLBK_ANTIAFFINITY_NS` | (none) | No | List of namespace to apply Kubernete "Anti-Affinity" rules to. This script applies `preferredDuringSchedulingIgnoredDuringExecution` for pod allocation if this is specified. |
| `SOLBK_ANTIAFFINITY_WT` | 100 | No | Weight of Anti-Affinity rule. Values range from 1-100. |
| `SOLBK_DIAG_DIR` | diag-configs | No | Directory to store output of `061-gather-configs.sh`. |
| `SOLBK_LOADBALANCER_IP` | (none) | No | If static IP address is required to be specified for Metal LB |
| `SOLBK_DOMAINCERT_FOLDER` | (none) | No | Folder containing Certificate Authority certificates to be loaded. This is used in conjunction with `$SOLBK_DOMAINCERT_FILES` |
| `SOLBK_DOMAINCERT_FILES` | (none) | No | List of Certificate Authority certificates to be loaded into the broker. This will be applied by `051-load-domain-certs.sh`. |
| `SOLBK_CLISCRIPTS_FOLDER` | cli | No | Path + Folder name of directory containing CLI scripts to be executed. This is used in `059-execute-cli.sh` |

# Additional Helper Scripts

(TODO)
