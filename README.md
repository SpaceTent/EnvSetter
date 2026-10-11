# EnvSetter

EnvSetter is a small, dependency-free client for [EnvVault.net](https://www.envvault.net). It downloads an encrypted `.env` envelope, verifies its authenticated metadata and decrypts it locally with AES-256-GCM. The decryption key is never sent to EnvVault.

This repository contains:

- the `envsetter` command for writing a protected `.env` file or running a process without writing plaintext to disk;
- the importable Go package `pkg/envvaultclient`;
- deployment examples for local shells, Docker, Kubernetes, CI systems, systemd and Nomad;
- wrapper examples for Go, Python and Node.js.

The client has no third-party Go dependencies.

## Security model

EnvVault stores ciphertext. EnvSetter needs two independent credentials:

- `ENVVAULT_TOKEN` (`mvt_...`) authorizes a read of the encrypted envelope;
- `ENVVAULT_KEY` is the unpadded base64url encoding of the 32-byte AES key and decrypts the envelope locally.

The token is sent only in the `Authorization: Bearer` header. The key is never placed in the URL, request body or headers. Possession of only one credential is not enough to recover the plaintext.

The response must have exactly the expected v1 envelope fields and an `X-Envvault-Team-ID` header. EnvSetter authenticates `envvault:v1:<team-id>:<secret-name>` as AES-GCM additional data, preventing an envelope from being substituted under another secret name.

## Install

Build from source with Go 1.24 or newer:

```sh
go install github.com/SpaceTent/EnvSetter/cmd/envsetter@latest
```

Or clone the repository and build a local binary:

```sh
make build
./bin/envsetter version
```

Container builds are also supported:

```sh
docker build -t envsetter:local .
```

## Get your EnvVault values

In EnvVault, open the **Account URL** page for the vault. It shows the vault URL name and token. Open the secret to obtain the secret URL name and use the decryption key that was shown when the file was encrypted.

```sh
export ENVVAULT_URL=https://www.envvault.net
export ENVVAULT_VAULT=my-vault
export ENVVAULT_SECRET=production-api
export ENVVAULT_TOKEN=mvt_replace_me
export ENVVAULT_KEY=replace_with_your_base64url_key
```

Do not commit these values. `.env` and `.env.*` are ignored by this repository, except for files named `.env.example`.

## Pull a `.env` file

```sh
envsetter pull --output .env
```

Vault and secret names can be passed explicitly:

```sh
envsetter pull \
  --vault my-vault \
  --secret production-api \
  --output .env
```

The destination is created with mode `0600`. EnvSetter refuses to overwrite it unless `--force` is present and refuses to write into a world-writable directory.

The original positional interface remains available for existing deployments. The old `envvault:` selector prefix is optional:

```sh
envsetter my-vault production-api .env
envsetter envvault:my-vault production-api .env
```

## Run without writing plaintext

`run` loads the variables directly into a child process. Values from the decrypted file replace same-named variables in the child. `ENVVAULT_TOKEN` and `ENVVAULT_KEY` are removed before it starts.

```sh
envsetter run --vault my-vault --secret production-api -- ./server
envsetter run -- npm start
envsetter run -- python -m myapp
```

The command's exit code is returned unchanged.

## Read credentials from files

For mounted secrets, read the token and key from separate mode-`0600` files:

```sh
envsetter run \
  --token-file /run/secrets/envvault-token \
  --key-file /run/secrets/envvault-key \
  -- ./server
```

EnvSetter rejects credential files that group or other users can read, write or execute.

## Use the Go package

```go
ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
defer cancel()

vars, err := envvaultclient.Load(ctx, envvaultclient.FromEnv(envvaultclient.Options{}))
if errors.Is(err, envvaultclient.ErrWrongKey) {
    log.Fatal("the configured key cannot open this secret")
}
if err != nil {
    log.Fatal(err)
}

dbURL := vars["DATABASE_URL"]
```

`Fetch` returns the plaintext `.env` bytes, `Load` returns a map, `Setenv` updates the current process environment and `FetchEnvelope` returns only the ciphertext. See [`examples/go/main.go`](examples/go/main.go) for a runnable program.

## Examples

All examples use placeholders and are safe to commit. Put real credentials only in the secret store provided by the target platform.

| Example | Pattern |
| --- | --- |
| [`examples/curl`](examples/curl) | Inspect the read-only ciphertext API with `curl` |
| [`examples/shell`](examples/shell) | Pull a file or run a command from POSIX shell |
| [`examples/go`](examples/go) | Import `envvaultclient` directly |
| [`examples/python`](examples/python) | Start a Python app through EnvSetter |
| [`examples/node`](examples/node) | Start a Node.js app through EnvSetter |
| [`examples/docker`](examples/docker) | Bake EnvSetter into a multi-stage application image |
| [`examples/docker-compose`](examples/docker-compose) | Supply credentials as mounted files and run an app in memory |
| [`examples/kubernetes`](examples/kubernetes) | Init container with a memory-backed `emptyDir` |
| [`examples/github-actions`](examples/github-actions) | Pull during a GitHub Actions job |
| [`examples/gitlab-ci`](examples/gitlab-ci) | Pull during a GitLab CI job |
| [`examples/systemd`](examples/systemd) | Prepare `/run` before a service starts |
| [`examples/nomad`](examples/nomad) | Prestart task writing to a task-group tmpfs volume |

See [`examples/README.md`](examples/README.md) for the shared rules and copy-paste commands.

## Direct HTTP API

To fetch ciphertext without decrypting it:

```sh
curl --fail-with-body \
  --header "Authorization: Bearer ${ENVVAULT_TOKEN}" \
  --header 'Accept: application/json' \
  "https://www.envvault.net/v1/api/my-vault/production-api"
```

Never put a token or key in the URL. The successful response is an encrypted envelope:

```json
{
  "v": 1,
  "alg": "AES-256-GCM",
  "iv": "base64url-without-padding",
  "ciphertext": "base64url-ciphertext-and-tag",
  "aad": "envvault:v1:team-id:production-api"
}
```

Writing a client correctly means validating the version, algorithm, exact field set, IV length, GCM tag, AAD and `X-Envvault-Team-ID` response header before accepting plaintext. Prefer the command or Go package over ad-hoc shell decryption.

## Development

```sh
make fmt
make check
make build
```

The tests use an in-process HTTP server and generated AES keys. They never contact EnvVault.net and contain no real credentials.

## Publishing checklist

- Select the repository visibility and enable secret scanning.
- Review the copyright name in [`LICENSE`](LICENSE).
- Publish a signed release and attach checksums for binaries.
- Build the container with an immutable version tag, not only `latest`.
- Add the repository's final GHCR image name to the container examples.
- Keep tokens and keys in separate secret records where the platform permits it.

## License

MIT. See [`LICENSE`](LICENSE).
