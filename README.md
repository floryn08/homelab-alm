# Homelab ALM

Kubernetes operator that automates certificate and ingress creation by fetching domain configuration from Vault.

## What It Does

Creates **cert-manager Certificates** and **Traefik IngressRoutes** using domain names stored in HashiCorp Vault.

Instead of hardcoding domains everywhere, you store them once in Vault and reference them by key.

## Quick Start

### Prerequisites

- Kubernetes with cert-manager and Traefik v3
- HashiCorp Vault with domain config at `kv/data/domains`
- ClusterIssuer named `ca-issuer` (or specify your own)

### Install

```bash
helm install homelab-alm ./deployment/helm \
  --set image=ghcr.io/floryn08/homelab-alm:v1.4.0 \
  --namespace homelab-alm-system \
  --create-namespace
```

### Configure Vault

```bash
vault kv put kv/domains \
  prodDomain=example.com \
  stagingDomain=staging.example.com
```

Set environment variables for the operator:
- `VAULT_ADDR=https://vault.example.com:8200`
- `VAULT_TOKEN=your-token`

## Usage

### Create a Certificate

```yaml
apiVersion: networking.alm.homelab/v1
kind: CertificateRequest
metadata:
  name: myapp-cert
spec:
  domainKey: prodDomain      # Looks up in Vault
  subdomain: myapp           # Creates myapp.example.com
  secretName: myapp-tls
  issuerName: ca-issuer      # Optional, defaults to ca-issuer
  issuerKind: ClusterIssuer  # Optional, defaults to ClusterIssuer
```

### Create an Ingress

```yaml
apiVersion: networking.alm.homelab/v1
kind: IngressRequest
metadata:
  name: myapp-ingress
spec:
  domainKey: prodDomain
  subdomain: myapp
  serviceName: myapp-service
  servicePort: "80"
  entrypoints: [websecure]
  tls:
    secretName: myapp-tls
```

## CRD Reference

### CertificateRequest

| Field | Required | Description |
|-------|----------|-------------|
| `domainKey` | Yes | Key to lookup in Vault |
| `subdomain` | No | Subdomain to prepend to domain |
| `secretName` | Yes | K8s secret name for certificate |
| `vaultPath` | No | Vault path (default: `kv/data/domains`) |
| `issuerName` | No | cert-manager issuer (default: `ca-issuer`) |
| `issuerKind` | No | `Issuer` or `ClusterIssuer` (default: `ClusterIssuer`) |

### IngressRequest

| Field | Required | Description |
|-------|----------|-------------|
| `domainKey` | Yes | Key to lookup in Vault |
| `subdomain` | Yes | Subdomain to prepend to domain |
| `serviceName` | Yes | Target Kubernetes service |
| `servicePort` | Yes | Target service port |
| `vaultPath` | No | Vault path (default: `kv/data/domains`) |
| `entrypoints` | No | Traefik entrypoints (default: `[web]`) |
| `tls.secretName` | No | TLS secret reference |
| `tls.certResolver` | No | Traefik cert resolver |
| `middlewares` | No | List of Traefik middlewares |

## Development

```bash
# Generate manifests after changing CRDs
make manifests generate

# Run tests
make test

# Run locally (requires VAULT_ADDR and VAULT_TOKEN)
make run

# Build and deploy
make docker-build docker-push IMG=your-registry/homelab-alm:tag
make deploy IMG=your-registry/homelab-alm:tag
```

## Troubleshooting

**Operator not working?**
```bash
kubectl logs -n homelab-alm-system deployment/homelab-alm-controller-manager
```

**Domain not found?**
```bash
vault kv get kv/domains
```

**Certificate not issued?**
```bash
kubectl get certificate -A
kubectl describe certificaterequest <name>
```