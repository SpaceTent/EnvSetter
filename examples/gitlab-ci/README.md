# GitLab CI

Add `ENVVAULT_TOKEN` and `ENVVAULT_KEY` as masked, protected CI/CD variables. Add `ENVVAULT_VAULT` and `ENVVAULT_SECRET` as protected variables, then copy `.gitlab-ci.yml` into your pipeline configuration.

Pin the `go install` target to a reviewed EnvSetter release. Restrict protected variables to protected branches and environments. The job uses `run`, leaving no plaintext `.env` artifact or cache entry.
