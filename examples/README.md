# EnvVault integration examples

The examples use the same five inputs:

| Variable | Meaning |
| --- | --- |
| `ENVVAULT_URL` | API origin; defaults to `https://www.envvault.net` |
| `ENVVAULT_VAULT` | URL-safe vault name, such as `my-vault` |
| `ENVVAULT_SECRET` | URL-safe secret name, such as `production-api` |
| `ENVVAULT_TOKEN` | Vault read token beginning `mvt_` |
| `ENVVAULT_KEY` | 32-byte AES key encoded as unpadded base64url |

Start locally with fake-free values from your EnvVault Account URL page:

```sh
cp .env.example .env.local
chmod 600 .env.local
set -a
. ./.env.local
set +a
```

Never commit `.env.local`, rendered Kubernetes Secrets, CI variables, token files, key files or fetched plaintext. In production, pin container images and downloaded binaries by immutable version or digest.

Choose the integration based on where plaintext should exist:

- `envsetter run` keeps plaintext off disk and is the preferred pattern for a single process.
- `envsetter pull` is appropriate when an application requires a dotenv file. Use a mode-`0600` file on memory-backed storage when possible.
- `curl` proves API connectivity but returns ciphertext only; it is not a replacement for authenticated decryption.
