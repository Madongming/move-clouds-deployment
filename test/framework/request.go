package framework

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"net/http"

	"github.com/go-resty/resty/v2"
	. "github.com/onsi/gomega"
)

// RequestOptionFunc function to add new options to request before returning
type RequestOptionFunc func(*resty.Request) *resty.Request

// WithFileRequest loads a HTTP request file from the file system and generates a resty.Request
// the contents of the file are basically in the following format:
//
// POST /api/v1/webhook/gitlab HTTP/1.1
// Host: localhost:30001
// Connection: close
// Accept: */*
// Accept-Encoding: gzip, deflate
// content-type: application/json
// Content-Length: 2458
//
// {"hello":"world"}
//
// A few important things:
// for posting body it is necessary to specify content-length and content-type otherwise the request
// will return with an empty body
// this is required by the http.ReadRequest method which makes this validation
func WithFileRequest(f *Framework, client *resty.Client, file string, opts ...RequestOptionFunc) *resty.Request {
	var (
		req *http.Request
		err error
	)
	req, err = LoadRequest(file)
	Expect(err).To(Succeed(), "should have loaded request file "+file)

	// returning as a restyReq
	restyReq := client.NewRequest()
	if req.Body != nil {
		body, readErr := ioutil.ReadAll(req.Body)
		Expect(readErr).To(Succeed(), "should read the body successfuly")
		restyReq = restyReq.SetBody(body)
	}

	if len(req.Header) > 0 {
		for k, v := range req.Header {
			restyReq = restyReq.SetHeader(k, v[0])
		}
	}
	for _, f := range opts {
		restyReq = f(restyReq)
	}
	return restyReq
}

// LoadRequest loads a file as a request
// the file contents must a plain text request
func LoadRequest(file string) (req *http.Request, err error) {
	var data []byte
	if data, err = ioutil.ReadFile(file); err != nil {
		return
	}
	reader := bufio.NewReader(bytes.NewBuffer(data))
	req, err = http.ReadRequest(reader)
	return
}
