// Contains code for managing VICE applciations running
// outside of the k8s cluster, namely in HTCondor

package external

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/cyverse-de/app-exposer/common"
	"github.com/labstack/echo/v4"
	extv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/kubernetes"
)

var log = common.Log

// External contains the support for running VICE apps outside of k8s.
type External struct {
	namespace          string
	clientset          kubernetes.Interface
	ServiceController  ServiceCrudder
	EndpointController EndpointCrudder
	IngressController  IngressCrudder
}

// New returns a new *External.
func New(cs kubernetes.Interface, namespace, ingressClass string) *External {
	return &External{
		clientset:          cs,
		namespace:          namespace,
		ServiceController:  NewServicer(cs.CoreV1().Services(namespace)),
		EndpointController: NewEndpointer(cs.CoreV1().Endpoints(namespace)),
		IngressController:  NewIngresser(cs.ExtensionsV1beta1().Ingresses(namespace), ingressClass),
	}
}

// CreateService is an http handler for creating a Service object in a k8s cluster.
//
// Expects JSON in the request body in the following format:
// 	{
// 		"target_port" : integer,
// 		"listen_port" : integer
// 	}
//
// The name of the Service comes from the URL the request is sent to and the
// namespace is a daemon-wide configuration setting.
func (e *External) CreateService(c echo.Context) error {
	var (
		service string
		err     error
	)

	service = c.Param("name")
	if service == "" {
		return &echo.HTTPError{
			Code:     http.StatusBadRequest,
			Internal: errors.New("missing service name in the URL"),
		}
	}

	log.Printf("CreateService: creating a service named %s", service)

	opts := &ServiceOptions{}
	if err = c.Bind(opts); err != nil {
		return err
	}

	if opts.TargetPort == 0 {
		return &echo.HTTPError{
			Code:     http.StatusBadRequest,
			Internal: errors.New("TargetPort was either not set or set to 0"),
		}
	}
	log.Printf("CreateService: target port for service %s will be %d", service, opts.TargetPort)

	if opts.ListenPort == 0 {
		return &echo.HTTPError{
			Code:     http.StatusBadRequest,
			Internal: errors.New("ListenPort was either not set or set to 0"),
		}
	}
	log.Printf("CreateService: listen port for service %s will be %d", service, opts.ListenPort)

	opts.Name = service
	opts.Namespace = e.namespace

	log.Printf("CreateService: namespace for service %s will be %s", service, opts.Namespace)

	svc, err := e.ServiceController.Create(opts)
	if err != nil {
		return err
	}

	log.Printf("CreateService: finished creating service %s", service)

	returnOpts := &ServiceOptions{
		Name:       svc.Name,
		Namespace:  svc.Namespace,
		ListenPort: svc.Spec.Ports[0].Port,
		TargetPort: svc.Spec.Ports[0].TargetPort.IntValue(),
	}

	return c.JSON(http.StatusOK, returnOpts)
}

// UpdateService is an http handler for updating a Service object in a k8s cluster.
//
// Expects JSON in the request body in the following format:
// 	{
// 		"target_port" : integer,
// 		"listen_port" : integer
// 	}
//
// The name of the Service comes from the URL the request is sent to and the
// namespace is a daemon-wide configuration setting.
func (e *External) UpdateService(c echo.Context) error {

	var (
		service string
		err     error
	)

	service = c.Param("name")
	if service == "" {
		return &echo.HTTPError{
			Code:     http.StatusBadRequest,
			Internal: errors.New("missing service name in the URL"),
		}
	}

	log.Printf("UpdateService: updating service %s", service)

	opts := &ServiceOptions{}
	if err = c.Bind(opts); err != nil {
		return err
	}

	if opts.TargetPort == 0 {
		return &echo.HTTPError{
			Code:     http.StatusBadRequest,
			Internal: errors.New("TargetPort was either not set or set to 0"),
		}
	}
	log.Printf("UpdateService: target port for %s should be %d", service, opts.TargetPort)

	if opts.ListenPort == 0 {
		return &echo.HTTPError{
			Code:     http.StatusBadRequest,
			Internal: errors.New("ListenPort was either not set or set to 0"),
		}
	}
	log.Printf("UpdateService: listen port for %s should be %d", service, opts.ListenPort)

	opts.Name = service
	opts.Namespace = e.namespace

	log.Printf("UpdateService: namespace for %s will be %s", service, opts.Namespace)

	svc, err := e.ServiceController.Update(opts)
	if err != nil {
		return err
	}

	log.Printf("UpdateService: finished updating service %s", service)

	returnOpts := &ServiceOptions{
		Name:       svc.Name,
		Namespace:  svc.Namespace,
		ListenPort: svc.Spec.Ports[0].Port,
		TargetPort: svc.Spec.Ports[0].TargetPort.IntValue(),
	}

	return c.JSON(http.StatusOK, returnOpts)
}

// GetService is an http handler for getting information about a Service object from
// a k8s cluster.
//
// Expects no body in the requests and will return a JSON encoded body in the
// response in the following format:
// 	{
// 		"name" : "The name of the service as a string.",
// 		"namespace" : "The namespace that the service is in, as a string",
// 		"target_port" : integer,
// 		"listen_port" : integer
// 	}
//
// The namespace of the Service comes from the daemon configuration setting.
func (e *External) GetService(c echo.Context) error {
	var (
		service string
	)

	service = c.Param("name")
	if service == "" {
		return &echo.HTTPError{
			Code:     http.StatusBadRequest,
			Internal: errors.New("missing service name in the URL"),
		}
	}

	log.Printf("GetService: getting info for service %s", service)

	svc, err := e.ServiceController.Get(service)
	if err != nil {
		return &echo.HTTPError{
			Code:     http.StatusBadRequest,
			Internal: err,
		}
	}

	log.Printf("GetService: finished getting info for service %s", service)

	returnOpts := &ServiceOptions{
		Name:       svc.Name,
		Namespace:  svc.Namespace,
		ListenPort: svc.Spec.Ports[0].Port,
		TargetPort: svc.Spec.Ports[0].TargetPort.IntValue(),
	}

	return c.JSON(http.StatusOK, returnOpts)
}

// DeleteService is an http handler for deleting a Service object in a k8s cluster.
//
// Expects no body in the request and returns no body in the response. Returns
// a 200 status if you try to delete a Service that doesn't exist.
func (e *External) DeleteService(c echo.Context) error {
	var service string

	service = c.Param("name")
	if service == "" {
		return &echo.HTTPError{
			Code:     http.StatusBadRequest,
			Internal: errors.New("missing service name in the URL"),
		}
	}

	log.Printf("DeleteService: deleting service %s", service)

	return e.ServiceController.Delete(service)
}

// CreateEndpoint is an http handler for creating an Endpoints object in a k8s cluster.
//
// Expects JSON in the request body in the following format:
// 	{
// 		"ip" : "IP address of the external process as a string.",
// 		"port" : The target port of the external process as an integer
// 	}
//
// The name of the Endpoint is derived from the URL the request was sent to and
// the namespace comes from the daemon-wide configuration value.
func (e *External) CreateEndpoint(c echo.Context) error {
	var (
		endpoint string
		err      error
	)

	endpoint = c.Param("name")
	if endpoint == "" {
		return &echo.HTTPError{
			Code:     http.StatusBadRequest,
			Internal: errors.New("missing endpoint name in the URL"),
		}
	}

	log.Printf("CreateEndpoint: creating an endpoint named %s", endpoint)

	opts := &EndpointOptions{}
	if err = c.Bind(opts); err != nil {
		return err
	}

	if opts.IP == "" {
		return &echo.HTTPError{
			Code:     http.StatusBadRequest,
			Internal: errors.New("IP field is blank"),
		}
	}
	log.Printf("CreateEndpoint: ip for endpoint %s will be %s", endpoint, opts.IP)

	if opts.Port == 0 {
		return &echo.HTTPError{
			Code:     http.StatusBadRequest,
			Internal: errors.New("Port field is blank"),
		}
	}
	log.Printf("CreateEndpoint: port for endpoint %s will be %d", endpoint, opts.Port)

	opts.Name = endpoint
	opts.Namespace = e.namespace

	log.Printf("CreateEndpoint: namespace for endpoint %s will be %s", endpoint, opts.Namespace)

	ept, err := e.EndpointController.Create(opts)
	if err != nil {
		return err
	}

	log.Printf("CreateEndpoint: finished creating endpoint %s", endpoint)

	returnOpts := &EndpointOptions{
		Name:      ept.Name,
		Namespace: ept.Namespace,
		IP:        ept.Subsets[0].Addresses[0].IP,
		Port:      ept.Subsets[0].Ports[0].Port,
	}

	return c.JSON(http.StatusOK, returnOpts)
}

// UpdateEndpoint is an http handler for updating an Endpoints object in a k8s cluster.
//
// Expects JSON in the request body in the following format:
// 	{
// 		"ip" : "IP address of the external process as a string.",
// 		"port" : The target port of the external process as an integer
// 	}
//
// The name of the Endpoint is derived from the URL the request was sent to and
// the namespace comes from the daemon-wide configuration value.
func (e *External) UpdateEndpoint(c echo.Context) error {
	var err error

	endpoint := c.Param("name")

	if endpoint == "" {
		return &echo.HTTPError{
			Code:     http.StatusBadRequest,
			Internal: errors.New("missing endpoint name in the URL"),
		}
	}

	log.Printf("UpdateEndpoint: updating endpoint %s", endpoint)

	opts := &EndpointOptions{}
	if err = c.Bind(opts); err != nil {
		return err
	}

	if opts.IP == "" {
		return &echo.HTTPError{
			Code:     http.StatusBadRequest,
			Internal: errors.New("IP field is blank"),
		}
	}
	log.Printf("UpdateEndpoint: ip for endpoint %s should be %s", endpoint, opts.IP)

	if opts.Port == 0 {
		return &echo.HTTPError{
			Code:     http.StatusBadRequest,
			Internal: errors.New("Port field is blank"),
		}
	}
	log.Printf("UpdateEndpoint: port for endpoint %s should be %d", endpoint, opts.Port)

	opts.Name = endpoint
	opts.Namespace = e.namespace

	log.Printf("UpdateEndpoint: namespace for endpoint %s should be %s", endpoint, opts.Namespace)

	ept, err := e.EndpointController.Update(opts)
	if err != nil {
		return err
	}

	log.Printf("UpdateEndpoint: finished updating endpoint %s", endpoint)

	returnOpts := &EndpointOptions{
		Name:      ept.Name,
		Namespace: ept.Namespace,
		IP:        ept.Subsets[0].Addresses[0].IP,
		Port:      ept.Subsets[0].Ports[0].Port,
	}

	return c.JSON(http.StatusOK, returnOpts)
}

// GetEndpoint is an http handler for getting an Endpoints object from a k8s cluster.
//
// Expects no body in the request and returns JSON in the response body in the
// following format:
// 	{
// 		"name" : "The name of the Endpoints object in Kubernetes, as a string.",
// 		"namespace" : "The namespace of the Endpoints object in Kubernetes, as a string.",
// 		"ip" : "IP address of the external process as a string.",
// 		"port" : The target port of the external process as an integer
// 	}
//
// The name of the Endpoint is derived from the URL the request was sent to and
// the namespace comes from the daemon-wide configuration value.
func (e *External) GetEndpoint(c echo.Context) error {
	var (
		endpoint string
		err      error
	)

	endpoint = c.Param("name")
	if endpoint == "" {
		return &echo.HTTPError{
			Code:     http.StatusBadRequest,
			Internal: errors.New("missing endpoint name in the URL"),
		}
	}

	log.Printf("GetEndpoint: getting info on endpoint %s", endpoint)

	ept, err := e.EndpointController.Get(endpoint)
	if err != nil {
		return err
	}

	log.Printf("GetEndpoint: done getting info on endpoint %s", endpoint)

	returnOpts := &EndpointOptions{
		Name:      ept.Name,
		Namespace: ept.Namespace,
		IP:        ept.Subsets[0].Addresses[0].IP,
		Port:      ept.Subsets[0].Ports[0].Port,
	}

	return c.JSON(http.StatusOK, returnOpts)
}

// DeleteEndpoint is an http handler for deleting an Endpoints object from a k8s cluster.
//
// Expects no request body and returns no body in the response. Returns a 200
// if you attempt to delete an Endpoints object that doesn't exist.
func (e *External) DeleteEndpoint(c echo.Context) error {
	var endpoint string

	endpoint = c.Param("name")
	if endpoint == "" {
		return &echo.HTTPError{
			Code:     http.StatusBadRequest,
			Internal: errors.New("missing endpoint name in the URL"),
		}
	}

	log.Printf("DeleteEndpoint: deleting endpoint %s", endpoint)

	return e.EndpointController.Delete(endpoint)
}

// WriteIngress uses the provided writer to write a version of the provided
// *typed_extv1beta1.Ingress object out as JSON in the response body.
func WriteIngress(ing *extv1beta1.Ingress, writer http.ResponseWriter) {
	returnOpts := &IngressOptions{
		Name:      ing.Name,
		Namespace: ing.Namespace,
		Service:   ing.Spec.Backend.ServiceName,
		Port:      ing.Spec.Backend.ServicePort.IntValue(),
	}

	outbuf, err := json.Marshal(returnOpts)
	if err != nil {
		common.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	writer.Write(outbuf)
}

// CreateIngress is an http handler for creating an Ingress object in a k8s cluster.
//
// Expects a JSON encoded request body in the following format:
// 	{
// 		"service" : "The name of the Service that the Ingress is configured for, as a string.",
// 		"port" : The port of the Service that the Ingress is configured for, as an integer
// 	}
//
// The name of the Ingress is extracted from the URL that the request is sent to.
// The namespace for the Ingress object comes from the daemon configuration setting.
func (e *External) CreateIngress(c echo.Context) error {
	var ingress string
	var err error

	ingress = c.Param("name")
	if ingress == "" {
		return &echo.HTTPError{
			Code:     http.StatusBadRequest,
			Internal: errors.New("missing ingress name in the URL"),
		}
	}

	log.Printf("CreateIngress: create an ingress named %s", ingress)

	opts := &IngressOptions{}
	if err = c.Bind(opts); err != nil {
		return err
	}

	if opts.Service == "" {
		return &echo.HTTPError{
			Code:     http.StatusBadRequest,
			Internal: errors.New("missing service from the ingress JSON"),
		}
	}
	log.Printf("CreateIngress: service name for ingress %s will be %s", ingress, opts.Service)

	if opts.Port == 0 {
		return &echo.HTTPError{
			Code:     http.StatusBadRequest,
			Internal: errors.New("Port was either not set or set to 0"),
		}
	}
	log.Printf("CreateIngress: port for ingress %s will be %d", ingress, opts.Port)

	opts.Name = ingress
	opts.Namespace = e.namespace

	log.Printf("CreateIngress: namespace for ingress %s will be %s", ingress, opts.Namespace)

	ing, err := e.IngressController.Create(opts)
	if err != nil {
		return err
	}

	log.Printf("CreateIngress: done creating ingress %s", ingress)

	returnOpts := &IngressOptions{
		Name:      ing.Name,
		Namespace: ing.Namespace,
		Service:   ing.Spec.Backend.ServiceName,
		Port:      ing.Spec.Backend.ServicePort.IntValue(),
	}

	return c.JSON(http.StatusOK, returnOpts)
}

// UpdateIngress is an http handler for updating an Ingress object in a k8s cluster.
//
// Expects a JSON encoded request body in the following format:
// 	{
// 		"service" : "The name of the Service that the Ingress is configured for, as a string.",
// 		"port" : The port of the Service that the Ingress is configured for, as an integer
// 	}
//
// The name of the Ingress is extracted from the URL that the request is sent to.
// The namespace for the Ingress object comes from the daemon configuration setting.
func (e *External) UpdateIngress(c echo.Context) error {
	var (
		ingress string
		err     error
	)

	ingress = c.Param("name")
	if ingress == "" {
		return &echo.HTTPError{
			Code:     http.StatusBadRequest,
			Internal: errors.New("missing ingress name in the URL"),
		}
	}

	log.Printf("UpdateIngress: updating ingress %s", ingress)

	opts := &IngressOptions{}
	if err = c.Bind(opts); err != nil {
		return nil
	}

	if opts.Service == "" {
		return &echo.HTTPError{
			Code:     http.StatusBadRequest,
			Internal: errors.New("missing service from the ingress JSON"),
		}
	}
	log.Printf("UpdateIngress: service for ingress %s should be %s", ingress, opts.Service)

	if opts.Port == 0 {
		return &echo.HTTPError{
			Code:     http.StatusBadRequest,
			Internal: errors.New("Port was either not set or set to 0"),
		}
	}
	log.Printf("UpdateIngress: port for ingress %s should be %d", ingress, opts.Port)

	opts.Name = ingress
	opts.Namespace = e.namespace

	log.Printf("UpdateIngress: namespace for ingress %s should be %s", ingress, opts.Namespace)

	ing, err := e.IngressController.Update(opts)
	if err != nil {
		return err
	}

	log.Printf("UpdateIngress: finished updating ingress %s", ingress)

	returnOpts := &IngressOptions{
		Name:      ing.Name,
		Namespace: ing.Namespace,
		Service:   ing.Spec.Backend.ServiceName,
		Port:      ing.Spec.Backend.ServicePort.IntValue(),
	}

	return c.JSON(http.StatusOK, returnOpts)
}

// GetIngress is an http handler for getting an Ingress object from a k8s cluster.
//
// Expects no request body and returns a JSON-encoded body in the response in the
// following format:
// 	{
// 		"name" : "The name of the Ingress, as a string.",
// 		"namespace" : "The Kubernetes namespace that the Ingress exists in, as a string.",
// 		"service" : "The name of the Service that the Ingress is configured for, as a string.",
// 		"port" : The port of the Service that the Ingress is configured for, as an integer
// 	}
func (e *External) GetIngress(c echo.Context) error {
	var (
		ingress string
		err     error
	)

	ingress = c.Param("name")
	if ingress == "" {
		return &echo.HTTPError{
			Code:     http.StatusBadRequest,
			Internal: errors.New("missing ingress name in the URL"),
		}
	}

	log.Printf("GetIngress: getting ingress %s", ingress)

	ing, err := e.IngressController.Get(ingress)
	if err != nil {
		return err
	}

	log.Printf("GetIngress: done getting ingress %s", ingress)

	returnOpts := &IngressOptions{
		Name:      ing.Name,
		Namespace: ing.Namespace,
		Service:   ing.Spec.Backend.ServiceName,
		Port:      ing.Spec.Backend.ServicePort.IntValue(),
	}

	return c.JSON(http.StatusOK, returnOpts)
}

// DeleteIngress is an http handler for deleting an Ingress object from a k8s cluster.
//
// Expects no request body and returns no body in the response. Returns a 200
// if you attempt to delete an Endpoints object that doesn't exist.
func (e *External) DeleteIngress(c echo.Context) error {
	var ingress string

	ingress = c.Param("name")
	if ingress == "" {
		return &echo.HTTPError{
			Code:     http.StatusBadRequest,
			Internal: errors.New("missing ingress name in the URL"),
		}
	}

	log.Printf("DeleteIngress: deleting ingress %s", ingress)

	return e.IngressController.Delete(ingress)
}
