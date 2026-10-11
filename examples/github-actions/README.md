# GitHub Actions

Create repository or environment secrets named `ENVVAULT_TOKEN` and `ENVVAULT_KEY`, plus non-secret variables named `ENVVAULT_VAULT` and `ENVVAULT_SECRET`. Copy `pull.yml` to `.github/workflows/`.

Pin `github.com/SpaceTent/EnvSetter/cmd/envsetter` to a reviewed release rather than `@latest` before using this in production. The example uses `run`, so no plaintext workspace file is created. GitHub masking does not make it safe to print decrypted values; application and test logs still need normal secret hygiene.
