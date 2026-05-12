# Sub2API Docker Deployment

Sub2API publishes a container image that is intended to be used with the compose files in this directory.

## Image

The default image used by `deploy/docker-compose*.yml` is:

```text
ghcr.io/gitfot/sub2api:latest
```

If you want to pin a tag or use a mirror, set `SUB2API_IMAGE` in `.env` before starting compose.

## Recommended Usage

Use the checked-in compose files instead of a handwritten `docker run` command so PostgreSQL, Redis, health checks, and persisted data stay aligned with the application defaults.

From the `deploy/` directory:

```bash
cp .env.example .env
docker compose -f docker-compose.local.yml up -d
```

Available compose variants:

- `docker-compose.yml`: named Docker volumes for app, PostgreSQL, and Redis.
- `docker-compose.local.yml`: bind-mounted local directories for easier backup and migration.
- `docker-compose.standalone.yml`: only runs Sub2API and expects external PostgreSQL and Redis.
- `docker-compose.dev.yml`: builds the image from local source for development verification.

## Important Environment Variables

These are the main variables most deployments care about. See `.env.example` for the full list.

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `SUB2API_IMAGE` | Image to pull for the `sub2api` service | No | `ghcr.io/gitfot/sub2api:latest` |
| `POSTGRES_PASSWORD` | PostgreSQL password for bundled database deployments | Yes | - |
| `SERVER_PORT` | Host port mapped to container port `8080` | No | `8080` |
| `SERVER_MODE` | Application mode (`release` or `debug`) | No | `release` |
| `JWT_SECRET` | Fixed JWT secret to preserve sessions across restarts | Recommended | auto-generated if empty |
| `TOTP_ENCRYPTION_KEY` | Fixed key for persisted 2FA secrets | Recommended | auto-generated if empty |
| `ADMIN_EMAIL` | Initial admin email | No | `admin@sub2api.local` |
| `ADMIN_PASSWORD` | Initial admin password | No | auto-generated if empty |

## Notes

- The root `Dockerfile` is the single source of truth for container builds.
- `docker-compose.dev.yml` builds from local source with that root `Dockerfile`.
- Production compose files pull a prebuilt image instead of building locally.
