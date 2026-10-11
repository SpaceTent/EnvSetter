# Kubernetes init container

The init container writes `.env` to a memory-backed `emptyDir`. The app pod sees the file but never receives the EnvVault token or key. Both containers use the same non-root UID so the app can read the mode-`0600` file.

Create the credentials without writing a manifest to disk:

```sh
kubectl create secret generic envvault-token \
  --from-literal=token="$ENVVAULT_TOKEN"
kubectl create secret generic envvault-key \
  --from-literal=key="$ENVVAULT_KEY"
kubectl apply -f configmap.yaml
kubectl apply -f deployment.yaml
```

Replace both image placeholders and pin them by digest. Your app must read `DOTENV_PATH`; mounting a dotenv file does not automatically add its entries to an already-running process environment.
