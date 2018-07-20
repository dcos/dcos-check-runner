package api

import (
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"

	"github.com/dcos/dcos-check-runner/runner"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

// NewRouter returns an API router for runner.
func NewRouter(runner *runner.Runner, baseURI string) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	rh := runnerHandler{runner: runner}

	base := router.PathPrefix(baseURI).Subrouter()
	base.Handle("/{check_type}/", withMiddlewares(http.HandlerFunc(rh.listChecks))).Methods("GET")
	base.Handle("/{check_type}/", withMiddlewares(http.HandlerFunc(rh.runChecks))).Methods("POST")

	return router
}

func withMiddlewares(h http.Handler) http.Handler {
	middlewares := [...]func(http.Handler) http.Handler{
		logRequestResponseMiddleware,
		loggerMiddleware,
	}

	for _, m := range middlewares {
		h = m(h)
	}
	return h
}

type runnerHandler struct {
	runner *runner.Runner
}

func (rh *runnerHandler) listChecks(w http.ResponseWriter, r *http.Request) {
	checkFunc, httpErr := rh.getCheckFuncFromReq(r)
	if httpErr != nil {
		http.Error(w, httpErr.Error(), httpErr.statusCode)
		return
	}

	rs, err := checkFunc(r.Context(), true, checksFromQueryParams(r)...)
	if err != nil {
		errMsg := "Error listing checks"
		reqLogger(r).Error(errors.Wrap(err, errMsg))
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	writeJSONResponse(w, r, rs)
}

func (rh *runnerHandler) runChecks(w http.ResponseWriter, r *http.Request) {
	checkFunc, httpErr := rh.getCheckFuncFromReq(r)
	if httpErr != nil {
		http.Error(w, httpErr.Error(), httpErr.statusCode)
		return
	}

	checks, httpErr := checksFromBody(r)
	if httpErr != nil {
		http.Error(w, httpErr.Error(), httpErr.statusCode)
		return
	}

	rs, err := checkFunc(r.Context(), false, checks...)
	if err != nil {
		errMsg := "Error running checks"
		reqLogger(r).Error(errors.Wrap(err, errMsg))
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}
	writeJSONResponse(w, r, rs)
}

// getCheckFuncFromReq returns the check function appropriate for r.
// The check function is determined from the check_type variable in the URI. If check_type is not "node" or "cluster",
// an httpError is returned with http.StatusNotFound. If check_type is not provided, an httpError is returned with
// http.StatusInternalServerError.
func (rh *runnerHandler) getCheckFuncFromReq(r *http.Request) (func(context.Context, bool, ...string) (*runner.CombinedResponse, error), *httpError) {
	checkType, ok := mux.Vars(r)["check_type"]
	if !ok {
		reqLogger(r).Error("check_type not provided in URI")
		return nil, &httpError{http.StatusInternalServerError, ""}
	}

	switch checkType {
	case "node":
		return rh.runner.PostStart, nil
	case "cluster":
		return rh.runner.Cluster, nil
	}
	return nil, &httpError{http.StatusNotFound, fmt.Sprintf("unrecognized check type: %s", checkType)}
}

// checksFromBody returns a slice of the check names from r's body.
// The request's body is decoded according to its Content-Type header.
func checksFromBody(r *http.Request) ([]string, *httpError) {
	ctHeader := r.Header.Get("Content-Type")
	if ctHeader == "" {
		return []string{}, nil
	}

	ct, _, err := mime.ParseMediaType(ctHeader)
	if err != nil {
		return nil, &httpError{http.StatusBadRequest, fmt.Sprintf("unable to parse Content-Type: %s", err)}
	}

	switch ct {
	case "application/x-www-form-urlencoded":
		checks, err := checksFromFormBody(r)
		if err != nil {
			return nil, &httpError{http.StatusBadRequest, err.Error()}
		}
		return checks, nil
	case "application/json":
		checks, err := checksFromJSONBody(r)
		if err != nil {
			return nil, &httpError{http.StatusBadRequest, err.Error()}
		}
		return checks, nil
	default:
		return nil, &httpError{http.StatusUnsupportedMediaType, fmt.Sprintf("unsupported Content-Type: %s", ct)}
	}
}

// checksFromJSONBody returns a slice of the check names from r's JSON-encoded body.
func checksFromJSONBody(r *http.Request) ([]string, error) {
	var t map[string]interface{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&t)
	if err != nil {
		return nil, err
	}

	checksVal, ok := t["check"]
	if !ok {
		return []string{}, nil
	}
	checksSlice, ok := checksVal.([]interface{})
	if !ok {
		return nil, fmt.Errorf("expected checks value of type []string, got %T", checksVal)
	}
	checks := make([]string, len(checksSlice))
	for i, check := range checksSlice {
		check, ok := check.(string)
		if !ok {
			return nil, fmt.Errorf("expected check type string, got %T", check)
		}
		checks[i] = check
	}
	return checks, nil
}

// checksFromFormBody returns a slice of the check names from r's url-encoded body.
func checksFromFormBody(r *http.Request) ([]string, error) {
	err := r.ParseForm()
	if err != nil {
		return nil, err
	}

	checks, ok := r.PostForm["check"]
	if !ok || checks == nil {
		return []string{}, nil
	}
	return checks, nil
}

// checksFromQueryParams returns a slice of the check names from r's query parameters.
func checksFromQueryParams(r *http.Request) []string {
	checks, ok := r.URL.Query()["check"]
	if !ok || checks == nil {
		return []string{}
	}
	return checks
}

// writeJSONResponse writes the JSON encoding of bodyObj to w.
func writeJSONResponse(w http.ResponseWriter, r *http.Request, bodyObj interface{}) {
	body, err := json.Marshal(bodyObj)
	if err != nil {
		reqLogger(r).Error(errors.Wrap(err, "failed to serialize JSON response"))
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(body)
}

type httpError struct {
	statusCode int
	err        string
}

func (e *httpError) Error() string {
	return e.err
}
