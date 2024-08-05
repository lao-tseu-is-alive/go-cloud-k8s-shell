package main

import (
	"encoding/json"
	"fmt"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/gohttp"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/golog"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/info"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/tools"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-shell/pkg/version"
	"github.com/stretchr/testify/assert"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	DEBUG                           = false
	assertCorrectStatusCodeExpected = "expected status code should be returned"
	fmtErrNewRequest                = "### ERROR http.NewRequest %s on [%s] error is :%v\n"
	fmtTraceInfo                    = "### %s : %s on %s\n"
	fmtErr                          = "### GOT ERROR : %s\n%s"
	msgRespNotExpected              = "Response should contain what was expected."
)

var l golog.MyLogger

type TestMainStruct struct {
	name                         string
	contentType                  string
	wantStatusCode               int
	wantBody                     string
	paramKeyValues               map[string]string
	httpMethod                   string
	url                          string
	useFormUrlencodedContentType bool
	body                         string
}

func setPortEnv(t *testing.T, port int) {
	err := os.Setenv("PORT", fmt.Sprintf("%d", port))
	if err != nil {
		t.Errorf("💥💥 ERROR: Unable to set env variable PORT")
		t.FailNow()
	}
}

func startMainServer(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		main()
	}()
}

func newRequest(t *testing.T, method, url, body string, useFormUrlencodedContentType bool) *http.Request {
	fmt.Printf("INFO: 🚀🚀'newRequest %s on %s ##BODY : %+v'\n", method, url, body)
	r, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf(fmtErrNewRequest, method, url, err)
	}
	if method == http.MethodPost && useFormUrlencodedContentType {
		r.Header.Set(gohttp.HeaderContentType, "Application/x-www-form-urlencoded")
	} else {
		r.Header.Set(gohttp.HeaderContentType, gohttp.MIMEAppJSON)
	}
	return r
}

func executeTest(t *testing.T, tt TestMainStruct, listenAddr string, l golog.MyLogger) {
	t.Run(tt.name, func(t *testing.T) {
		// prepare the request for this test case
		r := newRequest(t, tt.httpMethod, listenAddr+tt.url, tt.body, tt.useFormUrlencodedContentType)
		if len(tt.paramKeyValues) > 0 {
			parameters := r.URL.Query()
			for paramName, paramValue := range tt.paramKeyValues {
				parameters.Add(paramName, paramValue)
			}
			r.URL.RawQuery = parameters.Encode()
		}
		l.Debug(fmtTraceInfo, tt.name, tt.httpMethod, tt.url)
		resp, err := http.DefaultClient.Do(r)
		if err != nil {
			fmt.Printf(fmtErr, err, resp.Body)
			t.Fatal(err)
		}
		defer tools.CloseBody(resp.Body, tt.name, l)
		assert.Equal(t, tt.wantStatusCode, resp.StatusCode, assertCorrectStatusCodeExpected)
		receivedJson, _ := io.ReadAll(resp.Body)
		rInfo := &info.RuntimeInfo{}
		tools.PrintWantedReceived(tt.wantBody, receivedJson, l)
		if tt.wantStatusCode == http.StatusOK {
			if tt.contentType == gohttp.MIMEAppJSON {
				err = json.Unmarshal(receivedJson, rInfo)
				assert.Nil(t, err, "the output should be a valid json")
			}
		}
		// check that receivedJson contains the specified tt.wantBody substring
		assert.Contains(t, string(receivedJson), tt.wantBody, msgRespNotExpected)
	})
}

func TestMainExecution(t *testing.T) {
	defaultPort := 9999
	listenAddr := fmt.Sprintf("%s://%s:%d", "http", "127.0.0.1", defaultPort)
	setPortEnv(t, defaultPort)
	fmt.Printf("INFO: 'Will start HTTP server listening on port %s'\n", listenAddr)
	// starting main in his own go routine
	var wg sync.WaitGroup
	startMainServer(&wg)
	gohttp.WaitForHttpServer(listenAddr, 1*time.Second, 10)

	tests := []TestMainStruct{
		{
			name:                         "Get on default get handler should contain the App name",
			wantStatusCode:               http.StatusOK,
			contentType:                  gohttp.MIMEHtml,
			wantBody:                     version.APP,
			paramKeyValues:               make(map[string]string),
			httpMethod:                   http.MethodGet,
			url:                          "/",
			useFormUrlencodedContentType: false,
			body:                         "",
		},
		{
			name:                         "Post on default get handler should return an http error method not allowed ",
			wantStatusCode:               http.StatusMethodNotAllowed,
			contentType:                  gohttp.MIMEAppJSON,
			wantBody:                     "Method Not Allowed",
			paramKeyValues:               make(map[string]string),
			httpMethod:                   http.MethodPost,
			url:                          "/",
			useFormUrlencodedContentType: true,
			body:                         `{"junk":"test with junk text"}`,
		},
		{
			name:                         "Get on nonexistent route should return an http error not found ",
			wantStatusCode:               http.StatusNotFound,
			contentType:                  gohttp.MIMEAppJSON,
			wantBody:                     "page not found",
			paramKeyValues:               make(map[string]string),
			httpMethod:                   http.MethodGet,
			url:                          "/aroutethatwillneverexisthere",
			useFormUrlencodedContentType: false,
			body:                         "",
		},
		{
			name:                         "Get on default Server Path should return a valid html containing app",
			wantStatusCode:               http.StatusOK,
			contentType:                  gohttp.MIMEHtml,
			wantBody:                     version.APP,
			paramKeyValues:               make(map[string]string),
			httpMethod:                   http.MethodGet,
			url:                          "/",
			useFormUrlencodedContentType: false,
			body:                         "",
		},
		{
			name:                         "/health Post should return an http error method not allowed ",
			wantStatusCode:               http.StatusMethodNotAllowed,
			contentType:                  gohttp.MIMEAppJSON,
			wantBody:                     "",
			paramKeyValues:               make(map[string]string),
			httpMethod:                   http.MethodPost,
			url:                          "/health",
			useFormUrlencodedContentType: false,
			body:                         `{"task":"test not allowed method "}`,
		},
		{
			name:                         "/readiness Post should return an http error method not allowed ",
			wantStatusCode:               http.StatusMethodNotAllowed,
			contentType:                  gohttp.MIMEAppJSON,
			wantBody:                     "",
			paramKeyValues:               make(map[string]string),
			httpMethod:                   http.MethodPost,
			url:                          "/readiness",
			useFormUrlencodedContentType: false,
			body:                         `{"task":"test not allowed method "}`,
		},
		{
			name:                         "/time Post should return an http error method not allowed ",
			wantStatusCode:               http.StatusMethodNotAllowed,
			contentType:                  gohttp.MIMEAppJSON,
			wantBody:                     "",
			paramKeyValues:               make(map[string]string),
			httpMethod:                   http.MethodPost,
			url:                          "/time",
			useFormUrlencodedContentType: false,
			body:                         `{"task":"test not allowed method "}`,
		},
	}

	for _, tt := range tests {
		executeTest(t, tt, listenAddr, l)
	}
}

func init() {
	var err error
	if DEBUG {
		l, err = golog.NewLogger("zap", golog.DebugLevel, fmt.Sprintf("test_%s ", version.APP))
		if err != nil {
			log.Fatalf("💥💥 error golog.NewLogger error: %v'\n", err)
		}
	} else {
		l, err = golog.NewLogger("zap", golog.ErrorLevel, fmt.Sprintf("test_%s ", version.APP))
		if err != nil {
			log.Fatalf("💥💥 error golog.NewLogger error: %v'\n", err)
		}
	}
}
