# Stream

The Stream reflects a logical grouping of traffic flows. The stream points out the conduit the traffic will pass before it can be consumed by the target application.

Stream is a logical configuration entity, which cannot directly be found in the payload traffic.

The stream is the "network service entity" to be known and referred in the target application. The different target application pods can sign up for consumption of traffic from different streams. When more target pods are signed up for the same stream the traffic will be load-balanced between the pods.

Notice that a target pod concurrently only can sign up for streams belonging to the same trench.

This resource must be created with label `metadata.labels.trench` to specify its owner reference trench.

## API

- [v1](https://github.com/Nordix/Meridio/blob/master/api/v1/stream_types.go)
- [v1alpha1 (deprecated)](https://github.com/Nordix/Meridio/blob/master/api/v1alpha1/stream_types.go)

## Example

Here is an example of a Stream object:

```yaml
apiVersion: meridio.nordix.org/v1
kind: Stream
metadata:
  name: stream-a-i
  labels:
    trench: trench-a
spec:
  conduit: conduit-a-1
  max-targets: 100
```

The stream will be configured and running in `conduit-a-1`. Defined by `.spec.max-targets`, only 100 targets will be able to open to this stream.

## Deployment

After deploying the example from the previous section, the following resources have been created in Kubernetes:

```sh
$ kubectl get streams
NAME         CONDUIT       TRENCH     MAX-TARGETS
stream-a-i   conduit-a-1   trench-a   100
```

No new resource has been deployed while deploying the VIPs, but the `meridio-configuration-<trench-name>` configmap has been configured.

The picture below represents a Kubernetes cluster with Stream applied and highlighted in red:
![Installation-Stream](../resources/Installation-Stream.svg)

## Limitations

* `.metadata.labels.trench` property is mandatory and immutable.
* A stream belonging to a `stateless-lb` conduit will consume memory. Here is the formula to calculate how much memory a stream will consume per pod: `(n*100*max)*4`. The shared memory files can be found in `/dev/shm/`.
   * `(n*100)`: in reality is a prime number close to this number.
   * `4`: sizeof(int).
   * e.g. With `max-targets` set to 100 (default), the stream will take 4000000 bytes (4mb) (`(100*100*100)*4`).

## Configuration

TODO: Update

Name | Type | Description | Required | Default
--- | --- | --- | --- | ---
name | string | Name of the Stream | yes |
conduit | string | Name of the Conduit the Stream belongs to | yes | 
max-targets | int | Max number of targets the stream supports | yes | 
