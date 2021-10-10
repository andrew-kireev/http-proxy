package delivery

import (
	"compress/gzip"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/sirupsen/logrus"

	"http-proxy/internal/proxy/models"
	"http-proxy/internal/proxy/repositiory"
)

type Proxy struct {
	rep *repositiory.DB
}

func NewProxy(rep *repositiory.DB) *Proxy {
	return &Proxy{
		rep: rep,
	}
}

func (p *Proxy) HandleProxyRequest(w http.ResponseWriter, r *http.Request) {
	body, _ := ioutil.ReadAll(r.Body)

	req := &models.Request{
		Method:  r.Method,
		Host:    r.Host,
		Path:    r.URL.Path,
		Headers: r.Header,
		Body:    string(body),
	}

	logrus.Info(req)

	err := p.rep.SaveRequest(req)
	if err != nil {
		logrus.Error(err)
	}

	_, err = p.HandleHTTPRequest(w, r)
	if err != nil {
		logrus.Info(err)
	}
}

func (p *Proxy) HandleHTTPRequest(w http.ResponseWriter, r *http.Request) (string, error) {
	proxyResponse, err := p.DoHttpRequest(r)
	if err != nil {
		logrus.Info(err)
	}
	for header, values := range proxyResponse.Header {
		for _, value := range values {
			w.Header().Add(header, value)
		}
	}
	w.WriteHeader(proxyResponse.StatusCode)
	_, err = io.Copy(w, proxyResponse.Body)
	if err != nil {
		logrus.Info(err)
	}
	defer proxyResponse.Body.Close()

	decodedResponse, err := DecodeResponse(proxyResponse)
	if err != nil {
		return "", err
	}

	return string(decodedResponse), nil
}

func (p *Proxy) DoHttpRequest(r *http.Request) (*http.Response, error) {
	request, err := http.NewRequest(r.Method, r.URL.String(), r.Body)
	if err != nil {
		return nil, err
	}

	request.Header = r.Header

	proxyResponse, err := http.DefaultTransport.RoundTrip(request)
	if err != nil {
		return nil, err
	}

	return proxyResponse, nil
}

func DecodeResponse(response *http.Response) ([]byte, error) {
	var body io.ReadCloser

	switch response.Header.Get("Content-Encoding") {
	case "gzip":
		var err error
		body, err = gzip.NewReader(response.Body)
		if err != nil {
			body = response.Body
		}
	default:
		body = response.Body
	}

	bodyByte, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, err
	}

	lineBreak := []byte("\n")
	bodyByte = append(bodyByte, lineBreak...)
	bodyByte = bodyByte[0:500]

	defer body.Close()

	return bodyByte, nil
}
