# Native Go client

Run this example from the repository root after exporting the five `ENVVAULT_*` variables:

```sh
go run ./examples/go
```

It imports `pkg/envvaultclient`, fetches and decrypts the secret in memory, and prints variable names only. A separate application can use `Fetch` for raw dotenv bytes, `Load` for a map or `Setenv` to update the current process.
