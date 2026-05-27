# Graph Report - C:/Users/hamly/git/solace-k8-scripts  (2026-05-27)

## Corpus Check
- Corpus is ~29,577 words - fits in a single context window. You may not need a graph.

## Summary
- 155 nodes · 186 edges · 47 communities (21 shown, 26 thin omitted)
- Extraction: 87% EXTRACTED · 13% INFERRED · 0% AMBIGUOUS · INFERRED: 25 edges (avg confidence: 0.86)
- Token cost: 126,081 input · 14,010 output

## Community Hubs (Navigation)
- [[_COMMUNITY_Deploy + Post-Config Pipeline|Deploy + Post-Config Pipeline]]
- [[_COMMUNITY_000-env Bootstrap & Defaults|000-env Bootstrap & Defaults]]
- [[_COMMUNITY_Pod CLI  Operator Lifecycle|Pod CLI / Operator Lifecycle]]
- [[_COMMUNITY_gen_yaml Manifest Pattern|gen_yaml Manifest Pattern]]
- [[_COMMUNITY_Local Image Mirroring (Minikube)|Local Image Mirroring (Minikube)]]
- [[_COMMUNITY_Config Gather Helper (069)|Config Gather Helper (069)]]
- [[_COMMUNITY_Node Labeling Helper (013)|Node Labeling Helper (013)]]
- [[_COMMUNITY_Operator Deploy Script (010)|Operator Deploy Script (010)]]
- [[_COMMUNITY_Env Bootstrap Wrapper (000)|Env Bootstrap Wrapper (000)]]
- [[_COMMUNITY_Broker Deploy Script (020)|Broker Deploy Script (020)]]
- [[_COMMUNITY_Operator Delete Script (110)|Operator Delete Script (110)]]
- [[_COMMUNITY_Show All Brokers Helper|Show All Brokers Helper]]
- [[_COMMUNITY_Env Check Script (001)|Env Check Script (001)]]
- [[_COMMUNITY_Assert Leader Script (050)|Assert Leader Script (050)]]
- [[_COMMUNITY_Server Cert Loader (051)|Server Cert Loader (051)]]
- [[_COMMUNITY_Disable Default VPN (053)|Disable Default VPN (053)]]
- [[_COMMUNITY_Broker Delete Script (120)|Broker Delete Script (120)]]
- [[_COMMUNITY_Storage Class Check (009)|Storage Class Check (009)]]
- [[_COMMUNITY_Disable Default Users (054)|Disable Default Users (054)]]
- [[_COMMUNITY_Execute CLI Script (059)|Execute CLI Script (059)]]
- [[_COMMUNITY_SEMP Login Test (060)|SEMP Login Test (060)]]
- [[_COMMUNITY_Redundancy Test (061)|Redundancy Test (061)]]
- [[_COMMUNITY_desc-broker Helper|desc-broker Helper]]
- [[_COMMUNITY_enter-solace-shell Helper|enter-solace-shell Helper]]
- [[_COMMUNITY_replicas-start Helper|replicas-start Helper]]
- [[_COMMUNITY_Local Tar Image Concept|Local Tar Image Concept]]
- [[_COMMUNITY_enter-solace-cli Helper|enter-solace-cli Helper]]
- [[_COMMUNITY_logs-broker Helper|logs-broker Helper]]
- [[_COMMUNITY_Node Labeling Concept|Node Labeling Concept]]
- [[_COMMUNITY_Solace Product Keys Concept|Solace Product Keys Concept]]
- [[_COMMUNITY_Broker Replication Concept|Broker Replication Concept]]
- [[_COMMUNITY_VPN Replication Concept|VPN Replication Concept]]

## God Nodes (most connected - your core abstractions)
1. `000-env.sh bootstrap script` - 16 edges
2. `020-deploy-broker.sh - deploys PubSubPlusEventBroker CR` - 11 edges
3. `Pod naming convention: {SOLBK_NAME}-pubsubplus-{p|b|m}-0` - 7 edges
4. `010-deploy-operator.sh - deploys Solace operator` - 6 edges
5. `053-disable-default-vpn.sh - disables built-in 'default' VPN` - 6 edges
6. `054-disable-all-default-usernames.sh - disables 'default' client-username in every VPN` - 6 edges
7. `059-execute-cli.sh - executes a CLI script inside a broker pod` - 6 edges
8. `kubectl exec + cli -Apes pattern (run Solace CLI in pod)` - 6 edges
9. `110-delete-operator.sh - deletes Solace operator` - 5 edges
10. `050-assert-leader.sh - asserts config-sync leader on HA broker` - 5 edges

## Surprising Connections (you probably didn't know these)
- `001-check-env.sh - environment variable summary` --semantically_similar_to--> `009-check-storage-class.sh - StorageClass validation`  [INFERRED] [semantically similar]
  001-check-env.sh → 009-check-storage-class.sh
- `010-deploy-operator.sh - deploys Solace operator` --semantically_similar_to--> `110-delete-operator.sh - deletes Solace operator`  [INFERRED] [semantically similar]
  010-deploy-operator.sh → 110-delete-operator.sh
- `gen_yaml() in 010-deploy-operator.sh (operator manifest bundle)` --semantically_similar_to--> `gen_yaml() in 110-delete-operator.sh (mirror of 010)`  [INFERRED] [semantically similar]
  010-deploy-operator.sh → 110-delete-operator.sh
- `053-disable-default-vpn.sh - disables built-in 'default' VPN` --semantically_similar_to--> `054-disable-all-default-usernames.sh - disables 'default' client-username in every VPN`  [INFERRED] [semantically similar]
  053-disable-default-vpn.sh → 054-disable-all-default-usernames.sh
- `pick_pod() helper` --rationale_for--> `pick_pod helper concept`  [EXTRACTED]
  000-env.sh → CLAUDE.md

## Hyperedges (group relationships)
- **All scripts source 000-env.sh as bootstrap** — 001check_env, 009check_storage_class, 010deploy_operator, 020deploy_broker, 050assert_leader, 051load_server_cert, 053disable_default_vpn, 054disable_all_default_usernames, 059execute_cli, 060test_semp_login, 110delete_operator, 000env_bootstrap [EXTRACTED 1.00]
- **Scripts using the gen_yaml() heredoc-to-kubectl pattern** — 010deploy_operator, 020deploy_broker, 110delete_operator, concept_gen_yaml_pattern [EXTRACTED 1.00]
- **Post-config scripts that copy CLI into pod and exec it** — 050assert_leader, 051load_server_cert, 053disable_default_vpn, 054disable_all_default_usernames, 059execute_cli, concept_kubectl_exec_cli, concept_kubectl_cp [EXTRACTED 1.00]
- **Scripts that call pick_pod helper** — desc-broker_script, enter-solace-cli_script, enter-solace-shell_script, logs-broker_script [EXTRACTED 1.00]
- **Operational helper scripts (no number prefix)** — desc-broker_script, enter-solace-cli_script, enter-solace-shell_script, logs-broker_script, replicas-start-broker_script, show-all-brokers_script [EXTRACTED 1.00]
- **Post-check scripts (action band 6)** — 060-test-semp-login_script, 061-test-redundancy_script, 069-gather-configs_script [EXTRACTED 1.00]

## Communities (47 total, 26 thin omitted)

### Community 0 - "Deploy + Post-Config Pipeline"
Cohesion: 0.12
Nodes (10): PubSubPlusEventBroker CRD, Canonical Delete run order, File-naming convention (Category/Action/Sequence), Pod-role argument (p/b/m), Canonical New-Deployment run order, KUBE, SOLBK_DIAG_DIR, SOLBK_NAME (+2 more)

### Community 1 - "000-env Bootstrap & Defaults"
Cohesion: 0.23
Nodes (22): 000-env.sh bootstrap script, Default values block (KUBE, SOLOP_IMAGE, SOLBK_REDUNDANCY defaults), env/<name> file loader, Mandatory variable validation (SOLBK_NAME, SOLBK_NS, SOLBK_IMAGE, SOLBK_IMG_TAG, SOLBK_STORAGE_MSGNODE), pick_pod() helper, SOLOP_DERIVED_NS auto-detection via kubectl, 001-check-env.sh - environment variable summary, 009-check-storage-class.sh - StorageClass validation (+14 more)

### Community 2 - "Pod CLI / Operator Lifecycle"
Cohesion: 0.14
Nodes (5): pick_pod() helper, Solace EventBroker Operator, Env-file pattern (--env <name>), pick_pod helper concept, env/sample template

### Community 3 - "gen_yaml Manifest Pattern"
Cohesion: 0.53
Nodes (6): gen_yaml() in 010-deploy-operator.sh (operator manifest bundle), gen_yaml() in 020-deploy-broker.sh (broker CR YAML), gen_yaml() in 110-delete-operator.sh (mirror of 010), gen_yaml() heredoc-to-kubectl pattern, PubSubPlusEventBroker CRD (pubsubplus.solace.com/v1beta1), Solace operator manifest bundle (CRD + RBAC + Deployment)

### Community 4 - "Local Image Mirroring (Minikube)"
Cohesion: 0.50
Nodes (3): Minikube Steps, Steps, Using local tar images for kubectl setup

### Community 5 - "Config Gather Helper (069)"
Cohesion: 0.67
Nodes (3): 069-gather-configs.sh script, echoUsage(), nodes

## Knowledge Gaps
- **28 isolated node(s):** `Using local tar images for kubectl setup`, `Steps`, `Minikube Steps`, `Label Cluster Nodes for Broker Pods`, `Apply Solace Product Keys` (+23 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **26 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `gen_yaml() heredoc-to-kubectl pattern` connect `gen_yaml Manifest Pattern` to `Pod CLI / Operator Lifecycle`?**
  _High betweenness centrality (0.108) - this node is a cross-community bridge._
- **Why does `gen_yaml() in 020-deploy-broker.sh (broker CR YAML)` connect `gen_yaml Manifest Pattern` to `000-env Bootstrap & Defaults`?**
  _High betweenness centrality (0.065) - this node is a cross-community bridge._
- **Why does `020-deploy-broker.sh - deploys PubSubPlusEventBroker CR` connect `000-env Bootstrap & Defaults` to `gen_yaml Manifest Pattern`?**
  _High betweenness centrality (0.037) - this node is a cross-community bridge._
- **Are the 8 inferred relationships involving `020-deploy-broker.sh - deploys PubSubPlusEventBroker CR` (e.g. with `Mandatory variable validation (SOLBK_NAME, SOLBK_NS, SOLBK_IMAGE, SOLBK_IMG_TAG, SOLBK_STORAGE_MSGNODE)` and `010-deploy-operator.sh - deploys Solace operator`) actually correct?**
  _`020-deploy-broker.sh - deploys PubSubPlusEventBroker CR` has 8 INFERRED edges - model-reasoned connections that need verification._
- **Are the 2 inferred relationships involving `010-deploy-operator.sh - deploys Solace operator` (e.g. with `110-delete-operator.sh - deletes Solace operator` and `020-deploy-broker.sh - deploys PubSubPlusEventBroker CR`) actually correct?**
  _`010-deploy-operator.sh - deploys Solace operator` has 2 INFERRED edges - model-reasoned connections that need verification._
- **Are the 2 inferred relationships involving `053-disable-default-vpn.sh - disables built-in 'default' VPN` (e.g. with `020-deploy-broker.sh - deploys PubSubPlusEventBroker CR` and `054-disable-all-default-usernames.sh - disables 'default' client-username in every VPN`) actually correct?**
  _`053-disable-default-vpn.sh - disables built-in 'default' VPN` has 2 INFERRED edges - model-reasoned connections that need verification._
- **What connects `Using local tar images for kubectl setup`, `Steps`, `Minikube Steps` to the rest of the system?**
  _35 weakly-connected nodes found - possible documentation gaps or missing edges._