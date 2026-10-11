# Python

This wrapper delegates secret handling to EnvSetter instead of implementing cryptography in application code:

```sh
python examples/python/run.py python -m http.server 8080
```

The Python child receives decrypted variables. It does not receive `ENVVAULT_TOKEN` or `ENVVAULT_KEY`.
