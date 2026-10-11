job "example-app" {
  datacenters = ["dc1"]
  type        = "service"

  group "app" {
    volume "envvault" {
      type      = "host"
      source    = "envvault-tmpfs"
      read_only = false
    }

    task "envvault-pull" {
      lifecycle {
        hook    = "prestart"
        sidecar = false
      }

      driver = "docker"
      config {
        image = "ghcr.io/spacetent/envsetter:REPLACE_WITH_VERSION"
        args  = ["pull", "--output", "/envvault/.env", "--force"]
      }

      env {
        ENVVAULT_URL    = "https://www.envvault.net"
        ENVVAULT_VAULT  = "my-vault"
        ENVVAULT_SECRET = "production-api"
      }

      # Populate these from Nomad Variables or Vault templates in a real job.
      template {
        destination = "secrets/envvault.env"
        env         = true
        perms       = "0600"
        data        = <<EOH
ENVVAULT_TOKEN={{ with nomadVar "nomad/jobs/example-app" }}{{ .envvault_token }}{{ end }}
ENVVAULT_KEY={{ with nomadVar "nomad/jobs/example-app" }}{{ .envvault_key }}{{ end }}
EOH
      }

      volume_mount {
        volume      = "envvault"
        destination = "/envvault"
        read_only   = false
      }
    }

    task "app" {
      driver = "docker"
      config {
        image = "ghcr.io/example/application:REPLACE_WITH_VERSION"
      }
      env {
        DOTENV_PATH = "/envvault/.env"
      }
      volume_mount {
        volume      = "envvault"
        destination = "/envvault"
        read_only   = true
      }
    }
  }
}
