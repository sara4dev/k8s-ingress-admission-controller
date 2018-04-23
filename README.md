# k8s-ingress-admission-controller

[![Build Status](https://travis-ci.org/saravanakumar-periyasamy/k8s-ingress-admission-controller.svg?branch=master)](https://travis-ci.org/saravanakumar-periyasamy/k8s-ingress-admission-controller)

This [ValidatingAdmissionWebhook](https://kubernetes.io/docs/admin/admission-controllers/#validatingadmissionwebhook-alpha-in-18-beta-in-19) admission controller helps to stop creating bad ingress objects in the multi-tenant clusters. 

In particular it denies `ingress` with

* Empty Host
* Wilcard Host (*)
* Local Host (localhost)
* Duplicate Host/Path

This code is inspired from the [example-webhook-admission-controller](https://github.com/caesarxuchao/example-webhook-admission-controller), but made it to work with Kubernetes 1.9


## Prerequisite

* [minikube](https://github.com/kubernetes/minikube/releases)
* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/#install-kubectl)
* [helm](https://docs.helm.sh/using_helm/#installing-helm)

## Minikube

* Start minkube with dynamic admission controllers

```
minikube start \
--extra-config=apiserver.Admission.PluginNames=NamespaceLifecycle,LimitRanger,ServiceAccount,PersistentVolumeLabel,DefaultStorageClass,DefaultTolerationSeconds,MutatingAdmissionWebhook,ValidatingAdmissionWebhook,ResourceQuota \
--kubernetes-version=v1.9.0
```

* install `helm`

```
helm init
```

Wait until the tiller pod is running

* install the helm chart

```
helm install k8s-ingress-admission-controller-helm/ --debug  --namespace=kube-system
```

* apply `ingress` with no host

```
kubectl apply -f test/empty-host.yaml
```

and the admission controller should deny the ingress with 

```
Error from server (Forbidden): error when creating "test/empty-host.yaml": 
admission webhook "k8s-ingress-admission-controller.target.k8s.io" denied the request: 
Empty hostname is not allowed in this cluster
```