/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"crypto/tls"
	"time"

	"k8s.io/api/admissionregistration/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/golang/glog"
)

// get a clientset with in-cluster config.
func getClient() *kubernetes.Clientset {
	// config, err := clientcmd.BuildConfigFromFlags("", "/Users/z0027kp/.kube/config")
	config, err := rest.InClusterConfig()
	if err != nil {
		glog.Fatal(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatal(err)
	}
	return clientset
}


func configTLS(clientset *kubernetes.Clientset) *tls.Config {
	sCert, err := tls.X509KeyPair(serverCert, serverKey)
	if err != nil {
		glog.Fatal(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{sCert},
	}
}

// register this example webhook admission controller with the kube-apiserver
// by creating externalAdmissionHookConfigurations.
func selfRegistration(clientset *kubernetes.Clientset, caCert []byte) {
	time.Sleep(10 * time.Second)
	client := clientset.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations()
	_, err := client.Get("k8s-ingress-admission-controller", metav1.GetOptions{})
	if err == nil {
		if err2 := client.Delete("k8s-ingress-admission-controller", nil); err2 != nil {
			glog.Fatal(err2)
		}
	}
	webhookConfig := &v1beta1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "k8s-ingress-admission-controller",
		},
		Webhooks: []v1beta1.Webhook{
			{
				Name: "k8s-ingress-admission-controller.target.k8s.io",
				Rules: []v1beta1.RuleWithOperations{{
					Operations: []v1beta1.OperationType{v1beta1.Create, v1beta1.Update},
					Rule: v1beta1.Rule{
						APIGroups:   []string{"extensions"},
						APIVersions: []string{"v1beta1"},
						Resources:   []string{"ingresses"},
					},
				}},
				ClientConfig: v1beta1.WebhookClientConfig{
					Service: &v1beta1.ServiceReference{
						Namespace: "kube-system",
						Name:      "k8s-ingress-admission-controller",
					},
					CABundle: caCert,
				},
			},
		},
	}
	if _, err := client.Create(webhookConfig); err != nil {
		glog.Fatal(err)
	}
}
