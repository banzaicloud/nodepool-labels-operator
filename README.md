# Nodepool Labels Operator

 >A **node pool** is a subset of node instances within a cluster with the same configuration, however, the overall cluster can contain multiple node pools as well as heterogeneous nodes/configurations. The Pipeline platform can manage any number of node pools on a Kubernetes cluster, each with different configurations - e.g. node pool 1 is local SSD, node pool 2 is spot or preemptible-based, node pool 3 contains GPUs - these configurations are turned into actual cloud-specific instances.

This operator watches node events to catch when a node joins the cluster. It uses a [Custom Resource](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) to keep track of the desired list of node labels for the nodes of a node pool. There is one such CR per node pool.
[Pipeline](https://beta.banzaicloud.io/) creates the CRs with the list of desired labels for each node pool and updates these when the user updates the node pool labels. The operator takes care of placing the labels listed in the CR to all the nodes that belong to the corresponding node pool. Since the concept of a node pool doesn't exists in Kubernetes, [Pipeline](https://beta.banzaicloud.io/) tracks what node pool a node belongs to via the `node.banzaicloud.io/nodepool: <node pool name>` node label. The `operator` relies on this label to identify the nodes of a node pool. If `node.banzaicloud.io/nodepool` label is not available than it falls back to cloud specific node labels to identify the node pool a node belongs to:

* AKS: `agent: <node pool name>`
* GKE: `cloud.google.com/gke-nodepool: <node pool name>`

As the desired labels descibred in the CR for a nodepool only contains labels which should be set on the related nodes the operator uses an annotation (`nodepool.banzaicloud.io/managed-labels`) on each node to keep track of the managed labels and it will removed those managed labels which are not present in the desired state.

## Installing the operator

```bash
helm repo add  banzai-stable http://kubernetes-charts.banzaicloud.com/branch/master
helm install banzai-stable/nodepool-labels-operator
```

## Example

```bash
cat <<EOF | kubectl create -f -
apiVersion: labels.banzaicloud.io/v1alpha1
kind: NodePoolLabelSet
metadata:
  name: test-pool-2
spec:
  labels:
    environment: "testing"
    team: "rnd"
EOF
```

```bash
# kubectl describe node gke-standard-cluster-1-test-pool-2-1be3b06d-42wd

Name:               gke-standard-cluster-1-test-pool-2-1be3b06d-42wd
Roles:              <none>
Labels:             environment=testing
                    team=rnd
Annotations:        nodepool.banzaicloud.io/managed-labels: ["environment","team"]
```

## Contributing

If you find this project useful here's how you can help:

* Send a pull request with your new features and bug fixes
* Help new users with issues they may encounter
* Support the development of this project and star this repo!
