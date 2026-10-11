# Fetch ciphertext with curl

`fetch-envelope.sh` is a connectivity and authorization example. It returns the encrypted JSON envelope and does not use `ENVVAULT_KEY`.

```sh
ENVVAULT_VAULT=my-vault \
ENVVAULT_SECRET=production-api \
ENVVAULT_TOKEN=mvt_replace_me \
./fetch-envelope.sh
```

Vault and secret names copied from EnvVault are URL-safe. Do not interpolate arbitrary user input into this example and never add a token or key to the URL.
