# User Application / Target

## TAPA Integration

### Label

The Spiffe label is required in the target pod so Spire will create an entry for it and the TAPA will be able to access certificates in order to communicate with other components (NSM, NSP...).

```yaml
"spiffe.io/spiffe-id": "true"
```

### Container

Here is the minimal TAPA container specification required:

```yaml
- name: tapa
  volumeMounts:
    - name: spire-agent-socket
      mountPath: /run/spire/sockets
      readOnly: true
    - name: nsm-socket
      mountPath: /var/lib/networkservicemesh
      readOnly: true
    - name: meridio-socket
      mountPath: /var/lib/meridio
      readOnly: false
    - name: tmp
      mountPath: /tmp
      readOnly: false
  securityContext:
    runAsNonRoot: true
    readOnlyRootFilesystem: true
    capabilities:
      drop:
      - all
      add:
      - DAC_OVERRIDE
```

Additional configuration via environment variables can be found on the [TAPA Configuration](tapa.md#configuration) documentation page.

### Volumes

Four Volumes must be added to the pod. Spire and NSM are required to access the socket files to communicate with the APIs. And the Meridio volume provides a socket file user container can use to communicate with the TAPA API.
If readOnlyRootFilesystem is enabled, the tmp volume provides a writable mount to create the health server socket that can be used by liveness, startup probes.

```yaml
volumes:
  - name: spire-agent-socket
    hostPath:
      path: /run/spire/sockets
      type: Directory
  - name: nsm-socket
    hostPath:
      path: /var/lib/networkservicemesh
      type: DirectoryOrCreate
  - name: meridio-socket
    emptyDir: {}
  - name: tmp
    emptyDir:
      medium: Memory
```

## Example

An example application Helm chart can be found [here](https://github.com/Nordix/Meridio/tree/master/examples/target).
