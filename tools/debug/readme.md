# Debugging

## Deploy 

```
kubectl apply -f tools/debug/debug-daemont.yaml
```

## Build

```
docker build -t debug-meridio -f tools/debug/Dockerfile .
docker tag debug-meridio:latest registry.nordix.org/cloud-native/meridio/debug-meridio:latest
docker push registry.nordix.org/cloud-native/meridio/debug-meridio:latest
```

## Commands
List netns::
```
ls -1i /var/run/netn
```

List netns (more details):
```
lsns -t net
```

Check the processes running in the network namespace:
```
ls -l /proc/[1-9]*/ns/net | grep <NS> | cut -f3 -d"/" | xargs ps -p
```

Find pid from container ID:
```
crictl inspect --output go-template --template '{{.info.pid}}' <CONTAINER-ID>
```

List containers:
```
crictl ps
```

Find network namespace from pod ID:
```
crictl inspectp <POD-ID> | jq -r '.info.runtimeSpec.linux.namespaces[] |select(.type=="network") | .path'
```
