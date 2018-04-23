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

	}
	admissionResponse.Allowed = true
	return &admissionResponse
}
