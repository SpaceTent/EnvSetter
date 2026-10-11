# POSIX shell

With the five `ENVVAULT_*` variables exported:

```sh
./pull.sh .env
./run.sh ./server --listen=:8080
```

`pull.sh` sets `umask 077`; EnvSetter also explicitly creates the destination as mode `0600`. `run.sh` replaces itself with EnvSetter, which then starts the requested process with decrypted variables and without the token or key.
