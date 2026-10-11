# Docker Compose

Compose mounts the token and key as separate files. EnvSetter validates that the mounted files are private, decrypts the `.env` in memory and starts the container command.

```sh
mkdir -p secrets
printf '%s' "$ENVVAULT_TOKEN" > secrets/token
printf '%s' "$ENVVAULT_KEY" > secrets/key
chmod 600 secrets/token secrets/key
cp .env.example .env
docker compose up --build
```

The repository ignores `.env` files but does not globally ignore a directory called `secrets`; do not run `git add` on real credential files. Prefer your deployment platform's managed secret facility outside local development.
