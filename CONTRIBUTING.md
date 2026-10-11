# Contributing

Run these checks before opening a pull request:

```sh
make fmt
make check
make build
```

Tests must not call the live EnvVault service or contain real credentials. Use `httptest.Server`, generated keys and clearly fake `mvt_` tokens. New errors and logs must not include URLs with credentials, authorization headers, decryption keys, ciphertext or plaintext values.

Keep the Go client dependency-free unless a dependency provides a compelling security benefit that cannot reasonably be implemented with the standard library.
