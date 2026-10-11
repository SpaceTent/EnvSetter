# Security policy

## Reporting a vulnerability

Please report security problems privately to the repository owner rather than opening a public issue. Include the affected version, reproduction steps and the impact you observed. Do not include live EnvVault tokens, keys or decrypted values.

## Credential handling

- Treat both the EnvVault token and decryption key as secrets.
- Store them separately where the deployment platform permits it.
- Never place either value in a URL, command-line argument, image layer or repository file.
- Rotate the vault token after accidental disclosure. Replace the encrypted secret with a newly generated key after key disclosure.
- Prefer `envsetter run` when an application can consume environment variables directly.
- If a file is required, keep it on memory-backed storage with mode `0600` and delete it when the process stops.

EnvSetter deliberately avoids logging request headers, credentials, ciphertext or plaintext. Errors name fields and line numbers, not secret values.
