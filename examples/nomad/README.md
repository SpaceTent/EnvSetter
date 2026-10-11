# Nomad prestart task

The prestart task fetches the secret before the application starts. The example reads the token and key from Nomad Variables through an environment template and writes the decrypted file to a shared host volume.

Configure `envvault-tmpfs` as a memory-backed host volume on clients, replace both image placeholders and adapt the Nomad Variable path. If the app image can include EnvSetter, prefer one task using `envsetter run` so no shared plaintext file is required.
