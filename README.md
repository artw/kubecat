# README

## Kubecat

Kubecat is a utility for directly manipulating the etcd state of a Kubernetes cluster. This is a **dangerous operation** and should only be done by experienced operators who understand the risks. Directly modifying etcd can lead to inconsistencies in the cluster state and potential data loss.

**Use this utility at your own risk. Always take a backup of your etcd cluster before making any changes.**

## Installation

The project can be built with `make`.

## Usage

Kubecat can be used to extract Kubernetes objects stored in etcd to YAML, and vice versa.

### Reading from etcd

To read an object from etcd, use the following command:

```
kubecat <etcd-key> --etcd-endpoint=<etcd-endpoint> --certfile=<path-to-cert> --keyfile=<path-to-key> --cafile=<path-to-ca> > output.yaml
```

### Writing to etcd

To write an object to etcd, use the following command:

```
kubecat <etcd-key> --etcd-endpoint=<etcd-endpoint> --certfile=<path-to-cert> --keyfile=<path-to-key> --cafile=<path-to-ca> --write < input.yaml
```

### Environment Variables

Kubecat supports the following environment variables, you can use them instead of flags to configure etcd client:

- `ETCDCTL_ENDPOINTS`: Comma-separated list of etcd endpoints (default: `localhost:2379`)
- `ETCDCTL_CACERT`: Path to the CA certificate file
- `ETCDCTL_CERT`: Path to the client certificate file
- `ETCDCTL_KEY`: Path to the client key file

## Example: Changing a PersistentVolume's CSI driver

Let's say we want to change a PersistentVolume's CSI driver from "cephfs.csi.ceph.com" to "cephfs.kernel.csi.ceph.com", and the volumeAttributes mounter from "fuse" to "kernel".

First, we extract the PersistentVolume's manifest:

```
kubecat /registry/persistentvolumes/<pv-name> --certfile=<path-to-cert> --keyfile=<path-to-key> --cafile=<path-to-ca> > pv.yaml
```

Then, we use `jq` to change the `spec.csi.driver` and `spec.csi.volumeAttributes.mounter` fields in the manifest:

```bash
cat pv.yaml | jq '.spec.csi.driver="cephfs.kernel.csi.ceph.com"' | jq '.spec.csi.volumeAttributes.mounter="kernel"' > modified_pv.yaml
```

(Note: This requires [jq](https://stedolan.github.io/jq/download/) installed on your system.)

Finally, we write the modified manifest back to etcd:

```
kubecat /registry/persistentvolumes/<pv-name> --certfile=<path-to-cert> --keyfile=<path-to-key> --cafile=<path-to-ca> --write < modified_pv.yaml
```

After these steps, the PersistentVolume's CSI driver and mounter will be updated in the Kubernetes etcd state.

**Again, please use this utility with extreme caution. Making direct modifications to the Kubernetes etcd state can lead to data inconsistencies and potential data loss. Always take a backup of your etcd cluster before making any changes.**
