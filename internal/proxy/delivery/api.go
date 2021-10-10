package delivery

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"http-proxy/internal/proxy/repositiory"
)

type Api struct {
	rep    *repositiory.DB
	proxy  *Proxy
	router *mux.Router
}

func NewApi(rep *repositiory.DB, proxy *Proxy) *Api {

	router := mux.NewRouter()

	api := &Api{
		rep:    rep,
		proxy:  proxy,
		router: router,
	}

	router.HandleFunc("/repeat/{request_id:[0-9]+}", api.RepeatRequest)
	router.HandleFunc("/requests/{request_id:[0-9]+}", api.GetRequest)
	router.HandleFunc("/requests", api.GetAllRequests)
	router.HandleFunc("/scan/{request_id:[0-9]+}", api.Scan)

	return api
}

func (api *Api) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	api.router.ServeHTTP(w, r)
}

func (api *Api) RepeatRequest(w http.ResponseWriter, r *http.Request) {
	requestID, err := strconv.Atoi(mux.Vars(r)["request_id"])
	if err != nil {
		logrus.Error(err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	request, err := api.rep.GetRequest(requestID)
	if err != nil {
		logrus.Error(err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	req := &http.Request{
		Method: request.Method,
		URL: &url.URL{
			Scheme: "http",
			Host:   request.Host,
			Path:   request.Path,
		},
		Body:   ioutil.NopCloser(strings.NewReader(request.Body)),
		Host:   request.Host,
		Header: request.Headers,
	}

	api.proxy.HandleHTTPRequest(w, req)
}

func (api *Api) GetRequest(w http.ResponseWriter, r *http.Request) {
	requestID, err := strconv.Atoi(mux.Vars(r)["request_id"])
	if err != nil {
		logrus.Error(err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	request, err := api.rep.GetRequest(requestID)
	if err != nil {
		logrus.Error(err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	bytes, err := json.Marshal(request)
	if err != nil {
		logrus.Error(err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Write(bytes)
}

func (api *Api) GetAllRequests(w http.ResponseWriter, r *http.Request) {
	requests, err := api.rep.GetAllRequests()
	if err != nil {
		logrus.Error(err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	for _, request := range requests {
		bytes, err := json.Marshal(request)
		if err != nil {
			logrus.Error(err)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Write(bytes)
		w.Write([]byte("\n\n"))
	}
}

func (api *Api) Scan(w http.ResponseWriter, r *http.Request) {
	requestID, err := strconv.Atoi(mux.Vars(r)["request_id"])
	if err != nil {
		logrus.Error(err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	req, err := api.rep.GetRequest(requestID)
	if err != nil {
		logrus.Error(err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	request := &http.Request{
		Method: req.Method,
		URL: &url.URL{
			Scheme: "http",
			Host:   req.Host,
			Path:   req.Path,
		},
		Body:   ioutil.NopCloser(strings.NewReader(req.Body)),
		Host:   req.Host,
		Header: req.Headers,
	}

	for _, symbol := range []string{";", "|", "`"} {
		if request.Method == http.MethodGet {
			request.Host = request.Host + symbol + "cat /etc/passwd" + symbol
		}

		if request.Method == http.MethodPost {
			bodyBytes, err := ioutil.ReadAll(request.Body)

			request.Body = ioutil.NopCloser(strings.NewReader(string(bodyBytes) + symbol + "cat /etc/passwd" + symbol))
			if err != nil {
				logrus.Error(err)
			}
		}

		response, err := api.proxy.HandleHTTPRequest(w, request)
		if err != nil {
			logrus.Error(err)
		}

		if strings.Contains(response, ":root") {
			w.Write([]byte("vulnerable request"))
			logrus.Info("vulnerable request")
			return
		}
	}

	logrus.Info("the request contains no vulnerabilities")
	w.Write([]byte("the request contains no vulnerabilities"))
}
