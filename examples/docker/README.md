# Application image

The example copies EnvSetter and an application into one final image. Its entrypoint decrypts the EnvVault file in memory and starts the application without a plaintext layer or runtime file.

Replace `@latest` with an immutable EnvSetter release before production use.

```sh
docker build -t example-app -f examples/docker/Dockerfile .
docker run --rm \
  -e ENVVAULT_VAULT \
  -e ENVVAULT_SECRET \
  -e ENVVAULT_TOKEN \
  -e ENVVAULT_KEY \
  example-app
```

For Docker Swarm, mount the token and key as separate secrets and add `--token-file` and `--key-file` to the entrypoint.
