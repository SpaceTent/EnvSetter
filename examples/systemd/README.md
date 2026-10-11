# systemd credentials

The unit uses systemd's credential directory for separate token and key files. It writes plaintext only beneath `/run`, whose default backing storage is temporary.

```sh
sudo install -o root -g root -m 0600 token /etc/envvault/example-app.token
sudo install -o root -g root -m 0600 key /etc/envvault/example-app.key
sudo install -m 0644 example-app.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now example-app
```

Install the application and EnvSetter at the paths used by the unit, create the `example-app` system user, and change the vault and secret names. The application must read `DOTENV_PATH`. For applications that accept environment variables directly, a small wrapper using `envsetter run` is simpler and avoids the file.
