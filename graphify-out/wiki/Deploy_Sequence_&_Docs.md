# Deploy Sequence & Docs

> 20 nodes

## Key Concepts

- **README.md** (20 connections) — `README.md`
- **Environment Loader (000-env.sh)** (20 connections) — `000-env.sh`
- **Numbered Script Convention (Category/Action/Sequence)** (3 connections) — `CLAUDE.md`
- **Canonical Deploy Sequence** (2 connections) — `README.md`
- **Verify StorageClass Settings** (2 connections) — `009-check-storage-class.sh`
- **Create Broker Namespace** (2 connections) — `011-create-namespace.sh`
- **Assert HA Config-Sync Leader** (2 connections) — `050-assert-leader.sh`
- **Load TLS Server Certificate** (2 connections) — `051-load-server-cert.sh`
- **Load Domain Certificate Authorities** (2 connections) — `052-load-domain-certs.sh`
- **Test SEMP API Login** (2 connections) — `060-test-semp-login.sh`
- **Test HA Redundancy Failover** (2 connections) — `061-test-redundancy.sh`
- **Gather Configs and Diagnostics** (2 connections) — `069-gather-configs.sh`
- **Delete Broker Namespace** (2 connections) — `111-delete-namespace.sh`
- **Delete PubSubPlusEventBroker CR and PVCs** (2 connections) — `120-delete-broker.sh`
- **Create Broker Secrets** (1 connections) — `012-create-secrets.sh`
- **Label Cluster Nodes for Broker Pods** (1 connections) — `013-label-nodes.sh`
- **Disable Default Message VPN Services** (1 connections) — `053-disable-default-vpn.sh`
- **Disable All Default Client-Usernames** (1 connections) — `054-disable-all-default-usernames.sh`
- **Apply Solace Product Keys** (1 connections) — `057-configure-product-keys.sh`
- **Delete Broker Secrets** (1 connections) — `112-delete-secrets.sh`

## Relationships

- No strong cross-community connections detected

## Source Files

- `000-env.sh`
- `009-check-storage-class.sh`
- `011-create-namespace.sh`
- `012-create-secrets.sh`
- `013-label-nodes.sh`
- `050-assert-leader.sh`
- `051-load-server-cert.sh`
- `052-load-domain-certs.sh`
- `053-disable-default-vpn.sh`
- `054-disable-all-default-usernames.sh`
- `057-configure-product-keys.sh`
- `060-test-semp-login.sh`
- `061-test-redundancy.sh`
- `069-gather-configs.sh`
- `111-delete-namespace.sh`
- `112-delete-secrets.sh`
- `120-delete-broker.sh`
- `CLAUDE.md`
- `README.md`

## Audit Trail

- EXTRACTED: 67 (94%)
- INFERRED: 3 (4%)
- AMBIGUOUS: 1 (1%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*