package fetch

import "net/http"

// RequestInterceptorFn is called just before an HTTP request is sent.
// It may mutate or replace the request. Returning a non-nil error aborts
// the request and surfaces the error from Do / Scan.
type RequestInterceptorFn func(req *http.Request) (*http.Request, error)

// ResponseInterceptorFn is called immediately after a response is received
// (but before the body is read / decoded). It may mutate or replace the
// response. Returning a non-nil error surfaces the error from Do / Scan.
type ResponseInterceptorFn func(resp *http.Response) (*http.Response, error)

// interceptors holds ordered slices of request and response hooks.
type interceptors struct {
	request  []RequestInterceptorFn
	response []ResponseInterceptorFn
}

func (i *interceptors) applyRequest(req *http.Request) (*http.Request, error) {
	var err error
	for _, fn := range i.request {
		req, err = fn(req)
		if err != nil {
			return nil, err
		}
	}
	return req, nil
}

func (i *interceptors) applyResponse(resp *http.Response) (*http.Response, error) {
	var err error
	for _, fn := range i.response {
		resp, err = fn(resp)
		if err != nil {
			return nil, err
		}
	}
	return resp, nil
}
