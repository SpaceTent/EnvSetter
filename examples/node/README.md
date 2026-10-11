# Node.js

Start any Node command through EnvSetter:

```sh
node examples/node/run.mjs npm start
```

This wrapper inherits the current `ENVVAULT_*` configuration. EnvSetter fetches and decrypts the file, removes its token and key, and then starts `npm` with the resulting application variables.
