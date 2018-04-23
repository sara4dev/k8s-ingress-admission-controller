package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/golang/glog"
	"k8s.io/api/admission/v1beta1"
	extbeta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

func main() {
	flag.Parse()
	http.HandleFunc("/", serve)
	http.HandleFunc("/healthz", healthz)
	clientset := getClient()
	server := &http.Server{
		Addr:      ":8000",
		TLSConfig: configTLS(clientset),
	}
	go selfRegistration(clientset, caCert)
	server.ListenAndServeTLS("", "")
}

func healthz(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "OK")
}

func serve(w http.ResponseWriter, r *http.Request) {
	glog.Info("Reviewing Ingress...")
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		glog.Errorf("contentType=%s, expect application/json", contentType)
		return
	}

	// glog.Info(body)

	admissionResponse := admit(body)
	ar := v1beta1.AdmissionReview{
		Response: admissionResponse,
	}

	resp, err := json.Marshal(ar)
	if err != nil {
		glog.Error(err)
	}
	if _, err := w.Write(resp); err != nil {
		glog.Error(err)
	}
}

// only allow pods to pull images from specific registry.
func admit(data []byte) *v1beta1.AdmissionResponse {
	admissionReview := v1beta1.AdmissionReview{}
	if err := json.Unmarshal(data, &admissionReview); err != nil {
		glog.Error(err)
		return nil
	}
	// The externalAdmissionHookConfiguration registered via selfRegistration
	// asks the kube-apiserver only sends admission request regarding ingress.
	ingressResource := metav1.GroupVersionResource{Group: "extensions", Version: "v1beta1", Resource: "ingresses"}
	if admissionReview.Request.Resource != ingressResource {
		glog.Errorf("expected resource to be %s", ingressResource)
		return nil
	}

	raw := admissionReview.Request.Object.Raw
	ingress := extbeta1.Ingress{}
	if err := json.Unmarshal(raw, &ingress); err != nil {
		glog.Error(err)
		return nil
	}

	admissionResponse := v1beta1.AdmissionResponse{}
	for _, rule := range ingress.Spec.Rules {
		// deny empty host
		if rule.Host == "" {
			admissionResponse.Allowed = false
			admissionResponse.Result = &metav1.Status{
				Message: "Empty hostname is not allowed in this cluster",
				Reason: metav1.StatusReasonForbidden,
			}
			glog.Info(ingress.Namespace + ":" + ingress.Name + " has empty host")
			return &admissionResponse
		}
		// deny wildcard host
		if rule.Host == "*" {
			admissionResponse.Allowed = false
			admissionResponse.Result = &metav1.Status{
				Message: "Wildcard hostname is not allowed in this cluster",
				Reason: metav1.StatusReasonForbidden,
			}
			glog.Info(ingress.Namespace + ":" + ingress.Name + " has wildcard host")
			return &admissionResponse
		}
		// deny localhost
		if strings.ToLower(rule.Host) == "localhost" {
			admissionResponse.Allowed = false
			admissionResponse.Result = &metav1.Status{
				Message: "Localhost hostname is not allowed in this cluster",
				Reason: metav1.StatusReasonForbidden,
			}
			glog.Info(ingress.Namespace + ":" + ingress.Name + " has localhost")
			return &admissionResponse
		}
		// deny dupliacte ingress hosts
		_ingressHostExists, _ingressNamespace, _ingressName :=
			ingressHostExists(rule, ingress.Namespace, ingress.Name)

		if _ingressHostExists {
			admissionResponse.Allowed = false
			admissionResponse.Result = &metav1.Status{
				Message: "Duplicate hostnames are not allowed in this cluster. Found a duplicate in " +
					_ingressNamespace + ":" + _ingressName,
				Reason: metav1.StatusReasonAlreadyExists,
			}
			glog.Info(ingress.Namespace + ":" + ingress.Name + " already exists")
			return &admissionResponse
		}

	}
	admissionResponse.Allowed = true
	return &admissionResponse
}

func ingressHostExists(ingressRule extbeta1.IngressRule, ingressNamespace string, ingressName string) (bool, string, string) {
	clientset := getClient()
	namespaces, error := clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
	if error != nil {
		glog.Error("Error while listing the namespaces")
	}
	for _, namespace := range namespaces.Items {
		ingresses, error := clientset.ExtensionsV1beta1().Ingresses(namespace.GetName()).List(metav1.ListOptions{})
		if error != nil {
			glog.Error("Error while getting the ingress")
		}
		for _, ingress := range ingresses.Items {
			for _, rule := range ingress.Spec.Rules {
				if rule.Host == ingressRule.Host {
					for _, ingressHostPath := range ingressRule.IngressRuleValue.HTTP.Paths {
						for _, ruleHostPath := range rule.IngressRuleValue.HTTP.Paths {
							// skip checking against the same ingress during updates
							if namespace.GetName() != ingressNamespace || ingress.GetName() != ingressName {
								if ruleHostPath.Path == ingressHostPath.Path {
									glog.Info("Found duplicate ingress host in " + namespace.GetName() + ":" + ingress.GetName() + ":" + rule.Host + ruleHostPath.Path)
									return true, namespace.GetName(), ingress.GetName()
								}
							}
						}
					}
				}
			}
		}
	}
	return false, "", ""
}
