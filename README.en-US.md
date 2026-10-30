# MinIO Open

[🇧🇷 Português](README.md) · 🇺🇸 English

**MinIO with the full admin console back**, packaged to run on a small VPS in a
few clicks with **CapRover**, and ready for **Typebot**.

A fork of [minio/minio](https://github.com/minio/minio), which went into
maintenance mode in 2025: the community lost the admin console and stopped
getting official Docker images and binaries. This fork brings both back.

[![license](https://img.shields.io/badge/license-AGPL%20V3-blue)](LICENSE)
[![image](https://img.shields.io/badge/image-ghcr.io%2Fchr1sred%2Fminio--open-2ea44f)](https://github.com/chr1sred/minio-open/pkgs/container/minio-open)

---

## Why this fork

| Feature | Upstream MinIO (2025) | MinIO Open |
|---|---|---|
| Console: users, groups, policies, access keys | ❌ removed (only the Object Browser is left) | ✅ |
| Console: OpenID / LDAP | ❌ removed | ✅ |
| Console: Lifecycle (expire/transition objects) | ❌ removed since Nov 2024 | ✅ |
| Console: Tiering, Site Replication | ❌ removed since Nov 2024 | ✅ |
| Console: events, notifications, quotas, bucket replication | ❌ removed | ✅ |
| Console: server settings, monitoring, logs, trace | ❌ removed | ✅ |
| Ready-to-use Docker image | ❌ no longer published | ✅ `ghcr.io/chr1sred/minio-open` |
| 2025 server fixes | ✅ | ✅ (latest `master` server) |
| Automatic setup for Typebot | — | ✅ bucket, public folder and dedicated user |
| Console login behind a proxy (CapRover, Traefik) | ⚠️ fails if the public domain isn't reachable from inside the VPS | ✅ `MINIO_CONSOLE_LOCAL_API=on` |
| Defaults for 2–4 GB VPS | — | ✅ ~160 MB idle RAM, configurable ceiling |
| Working Helm chart and Docker Compose | ❌ point to images that no longer exist | ✅ use this fork's image |
| Build and tests independent from MinIO Inc. | ❌ need `dl.min.io`, Docker Hub and private runners | ✅ everything builds from source on GitHub Actions |

**How it's done:** the server is the latest MinIO `master`. Only the console
(`github.com/minio/console`) is pinned to **v1.7.3**, the last release with the
full admin UI. It lives in [`third_party/console`](third_party/console), with
a minimal compatibility patch for the current `minio-go`.

---

## Quick start

### CapRover (recommended)

1. **CapRover → Apps → One-Click Apps/Databases → `>> TEMPLATE <<`**.
2. Paste the contents of [`deploy/caprover/minio-open-one-click.yml`](deploy/caprover/minio-open-one-click.yml)
   ([raw](https://raw.githubusercontent.com/chr1sred/minio-open/master/deploy/caprover/minio-open-one-click.yml)).
3. Name the app (e.g. `minio-open`), write down the generated passwords and click **Deploy**.
4. Enable **HTTPS** on both apps created (`minio-open` and `minio-open-console`).
5. Copy the `S3_*` variables from the `minio-open` **App Logs** into Typebot's builder and viewer.

Full step-by-step guide (in Portuguese), with variables and troubleshooting:
**[deploy/README.md](deploy/README.md)**.

### Docker

```bash
docker run -d --name minio -p 9000:9000 -p 9001:9001 -v minio-data:/data -e MINIO_ROOT_USER=admin -e MINIO_ROOT_PASSWORD=change-this-password ghcr.io/chr1sred/minio-open:latest
```

S3 API at `http://localhost:9000` and console at `http://localhost:9001`.

### Docker Compose

```bash
cp deploy/.env.example deploy/.env
```

```bash
docker compose -f deploy/docker-compose.yml --env-file deploy/.env up -d
```

### Kubernetes (Helm)

The chart in [`helm/minio`](helm/minio) already uses `ghcr.io/chr1sred/minio-open`:

```bash
helm install minio ./helm/minio --set mode=standalone,replicas=1,rootUser=admin,rootPassword=change-this-password
```

---

## Updating an existing install

Every push to `master` publishes a new `ghcr.io/chr1sred/minio-open:latest`.

| Where it runs | How to update |
|---|---|
| CapRover | App `minio-open` → **Deployment** → **Deploy via ImageName** → `ghcr.io/chr1sred/minio-open:latest` → **Deploy** |
| Docker | `docker pull ghcr.io/chr1sred/minio-open:latest` and recreate the container |
| Docker Compose | `docker compose -f deploy/docker-compose.yml pull`, then `up -d` |

Data in `/data`, users and policies are preserved. The bootstrap never
overwrites anything that already exists.

---

## What's in the image

| Component | Description |
|---|---|
| `minio` | S3 server + full admin console (port 9000 API, 9001 console) |
| `minio-bootstrap` | Runs on every start, never overwriting anything: creates buckets, grants anonymous read on prefixes, creates an application user with a restricted policy and prints the Typebot setup guide to the logs. Does nothing without `MINIO_INIT_*` variables |
| `mc` | MinIO Client, for command-line administration |

Alpine base, ~160 MB, built-in healthcheck at `/minio/health/live`.

### Main variables

| Variable | Default | Description |
|---|---|---|
| `MINIO_ROOT_USER` / `MINIO_ROOT_PASSWORD` | — | Admin credentials (required) |
| `MINIO_SERVER_URL` | — | Public S3 API URL |
| `MINIO_BROWSER_REDIRECT_URL` | — | Public console URL |
| `MINIO_CONSOLE_LOCAL_API` | `on` | Console talks to the API over 127.0.0.1 (new in this fork) |
| `MINIO_MEMLIMIT` | `1GiB` | Memory ceiling; also tunes max concurrency |
| `MINIO_SCANNER_SPEED` | `slow` | Lighter background scanner |
| `MINIO_INIT_BUCKETS` | — | Buckets created automatically (the CapRover template uses `typebot`) |
| `MINIO_INIT_PUBLIC_PATHS` | — | Prefixes with anonymous, download-only access (the template uses `typebot/public`) |
| `MINIO_INIT_APP_USER` / `MINIO_INIT_APP_PASSWORD` | — | Application user to create |
| `MINIO_INIT_ENABLE` | `on` | `off` disables the bootstrap |

Full list in [deploy/README.md](deploy/README.md#variáveis-de-ambiente).

---

## Typebot integration

The bootstrap sets MinIO up the way Typebot expects: a `typebot` bucket, a
`public/` prefix with anonymous read and a user restricted to that bucket.
In Typebot's builder **and** viewer:

| Variable | Value |
|---|---|
| `S3_ACCESS_KEY` | Typebot user (e.g. `typebot`) |
| `S3_SECRET_KEY` | Typebot password |
| `S3_BUCKET` | `typebot` |
| `S3_ENDPOINT` | `minio-open.yourdomain.com` (host only, no `https://`) |
| `S3_SSL` | `true` |
| `S3_REGION` | `us-east-1` |

Ready-made template in [`deploy/typebot.env.example`](deploy/typebot.env.example).
Validated in production with the current Typebot: uploads from the builder and
public image display.

---

## Building from source

Requires Go 1.26+.

```bash
CGO_ENABLED=0 go build -trimpath -o minio .
```

```bash
CGO_ENABLED=0 go build -trimpath -o minio-bootstrap ./deploy/bootstrap
```

Docker image:

```bash
docker build -t minio-open .
```

On every push to `master`, the [`open-image.yml`](.github/workflows/open-image.yml)
workflow publishes `ghcr.io/chr1sred/minio-open:latest`.

### Tests and CI

All upstream MinIO test suites run on this fork's GitHub Actions, with no
dependency on MinIO Inc. servers:

| Suite | What it checks |
|---|---|
| Lint, unit tests, cross-build | Go code (Go 1.26) |
| IAM (LDAP, OpenID, etcd) | Login and policies |
| Healing, resiliency, replication | Disk and site recovery |
| Upgrade | Upgrading from a 2019 release, built from source |
| Mint (manual, from the Actions tab) | Full S3 compatibility |
| Dependency review, vulncheck | Vulnerable dependencies |

---

## Fork layout

| Path | Purpose |
|---|---|
| `third_party/console/` | Full console v1.7.3 (+ compatibility patch) |
| `cmd/common-main.go` | `MINIO_CONSOLE_LOCAL_API` option |
| `deploy/bootstrap/` | `minio-bootstrap` source |
| `deploy/caprover/` | CapRover One-Click template |
| `deploy/docker-compose.yml`, `deploy/.env.example` | Running outside CapRover |
| `deploy/typebot.env.example` | Ready-made Typebot variables |
| `deploy/README.md` | Full deployment guide (Portuguese) |
| `Dockerfile`, `dockerscripts/open-entrypoint.sh` | Docker image |
| `helm/minio/` | Helm chart using this fork's image |
| `buildscripts/build-release-image.sh` | Builds old MinIO releases for upgrade tests |
| `captain-definition` | Deploy straight from GitHub on CapRover (builds on the VPS) |
| `.github/workflows/open-image.yml` | Publishes the image to GHCR |

The rest of the repository is upstream MinIO, with fixes to build and pass
tests on Go 1.26.

---

## Known limitations

- The console is from **Nov 2024 (v1.7.3)**. Releases 1.7.4–1.7.6 mostly
  brought the feature removals. Always use HTTPS and strong passwords.
- The console's **Subnet** menu (License, Health, Call Home…) belongs to
  MinIO's commercial service and does nothing here; ignore it.
- Designed for single-server/single-disk installs (`/data`). For distributed
  clusters, see the upstream MinIO documentation.

---

## License and credits

- Source code: [GNU AGPLv3](LICENSE), same as MinIO. If you offer this
  modified service to others, the source of the modifications must remain
  available; this public repository already does that.
- Original MinIO documentation: [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/).
- [License compliance](COMPLIANCE.md) · [Credits](CREDITS)

MinIO is a registered trademark of MinIO, Inc. This is an independent
community fork, not affiliated with MinIO, Inc.
