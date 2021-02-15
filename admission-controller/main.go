/*
Copyright 2018 The Kubernetes Authors.

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
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"

	. "k8s.io/api/admission/v1"
	"k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	// TODO: try this library to see if it generates correct json patch
	// https://github.com/mattbaird/jsonpatch
)

var (
	certFile     string
	keyFile      string
	port         int
	sidecarImage string
)

func init() {
	klog.Info(certFile)
	flag.StringVar(&certFile, "tls-cert-file", "",
		"File containing the default x509 Certificate for HTTPS. (CA cert, if any, concatenated after server cert).")
	klog.Info(keyFile)
	flag.StringVar(&keyFile, "tls-private-key-file", "",
		"File containing the default x509 private key matching --tls-cert-file.")
	klog.Info(port)
	flag.IntVar(&port, "port", 443,
		"Secure port that the webhook listens on")
	klog.Info(sidecarImage)
	flag.StringVar(&sidecarImage, "sidecar-image", "",
		"Image to be used as the injected sidecar")

}

// admitv1beta1Func handles a v1beta1 admission
type admitv1beta1Func func(v1beta1.AdmissionReview) *v1beta1.AdmissionResponse

// admitv1beta1Func handles a v1 admission
type admitv1Func func(AdmissionReview) *AdmissionResponse

// admitHandler is a handler, for both validators and mutators, that supports multiple admission review versions
type admitHandler struct {
	v1beta1 admitv1beta1Func
	v1      admitv1Func
}

func newDelegateToV1AdmitHandler(f admitv1Func) admitHandler {
	klog.Info("newDelegateToV1AdmitHandler()")
	return admitHandler{
		v1beta1: delegateV1beta1AdmitToV1(f),
		v1:      f,
	}
}

/**
convert v1beta to v1 withAdmissionReview which describes an admission review request/response.
*/
func delegateV1beta1AdmitToV1(f admitv1Func) admitv1beta1Func {
	klog.Info("delegateV1beta1AdmitToV1()")
	return func(review v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
		in := AdmissionReview{Request: convertAdmissionRequestToV1(review.Request)}
		out := f(in)
		return convertAdmissionResponseToV1beta1(out)
	}
}

/**
serve handles the http portion of a request prior to handing to an admit
*/
func serve(w http.ResponseWriter, r *http.Request, admit admitHandler) {
	klog.Info("serve()")
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		klog.Errorf("contentType=%s, expect application/json", contentType)
		return
	}

	klog.Info(fmt.Sprintf("handling request: %s", body))

	deserializer := codecs.UniversalDeserializer()
	obj, gvk, err := deserializer.Decode(body, nil, nil)
	if err != nil {
		msg := fmt.Sprintf("Request could not be decoded: %v", err)
		klog.Error(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	var responseObj runtime.Object
	switch *gvk {
	case v1beta1.SchemeGroupVersion.WithKind("AdmissionReview"):
		requestedAdmissionReview, ok := obj.(*v1beta1.AdmissionReview)
		if !ok {
			klog.Errorf("Expected v1beta1.AdmissionReview but got: %T", obj)
			return
		}
		responseAdmissionReview := &v1beta1.AdmissionReview{}
		responseAdmissionReview.SetGroupVersionKind(*gvk)
		responseAdmissionReview.Response = admit.v1beta1(*requestedAdmissionReview)
		responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID
		responseObj = responseAdmissionReview
	case SchemeGroupVersion.WithKind("AdmissionReview"):
		requestedAdmissionReview, ok := obj.(*AdmissionReview)
		if !ok {
			klog.Errorf("Expected v1.AdmissionReview but got: %T", obj)
			return
		}
		responseAdmissionReview := &AdmissionReview{}
		responseAdmissionReview.SetGroupVersionKind(*gvk)
		responseAdmissionReview.Response = admit.v1(*requestedAdmissionReview)
		responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID
		responseObj = responseAdmissionReview
	default:
		msg := fmt.Sprintf("Unsupported group version kind: %v", gvk)
		klog.Error(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	klog.V(2).Info(fmt.Sprintf("sending response: %v", responseObj))
	respBytes, err := json.Marshal(responseObj)
	if err != nil {
		klog.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(respBytes); err != nil {
		klog.Error(err)
	}
}

func serveMutatePods(w http.ResponseWriter, r *http.Request) {
	klog.Info("serveMutatePods()")
	serve(w, r, newDelegateToV1AdmitHandler(mutatePods))
}

func serveMutatePodsSidecar(w http.ResponseWriter, r *http.Request) {
	klog.Info("serveMutatePodsSidecar()")
	serve(w, r, newDelegateToV1AdmitHandler(mutatePodsSidecar))
}

func main() {
	klog.Info("Enter into adm-controller main()")
	loggingFlags := &flag.FlagSet{}
	klog.InitFlags(loggingFlags)
	flag.Parse()
	config := Config{
		CertFile: certFile,
		KeyFile:  keyFile,
	}

	klog.Info(config)

	http.HandleFunc("/mutating-pods", serveMutatePods)
	http.HandleFunc("/mutating-pods-sidecar", serveMutatePodsSidecar)
	http.HandleFunc("/readyz", func(w http.ResponseWriter, req *http.Request) { w.Write([]byte("ok")) })
	server := &http.Server{
		Addr:      fmt.Sprintf(":%d", port),
		TLSConfig: configTLS(config),
	}

	klog.Info("server adm-controller is listening to requests on: ", server.Addr, server.TLSConfig)

	err := server.ListenAndServeTLS("", "")
	if err != nil {
		panic(err)
	}
}
