package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	DEBUG                           = false
	assertCorrectStatusCodeExpected = "expected status code should be returned"
	expectedJsonString              = `{
  "hostname": "pulsar2021",
  "pid": 1,
  "ppid": 0,
  "uid": 1000,
  "appname": "go-info-server",
  "version": "0.3.0",
  "param_name": "_NO_PARAMETER_NAME_",
  "remote_addr": "127.0.0.1:56670",
  "goos": "linux",
  "goarch": "amd64",
  "runtime": "go1.18.3",
  "num_goroutine": "1",
  "num_cpu": "4",
  "env_vars": [
    "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
    "HOME=/home/gouser"
  ],
  "headers": {
    "Accept": [
      "*/*"
    ],
    "User-Agent": [
      "curl/7.68.0"
    ]
  }
}`
)

type testStruct struct {
	name           string
	wantStatusCode int
	wantBody       string
	paramKeyValues map[string]string
	r              *http.Request
}

func TestErrorConfigError(t *testing.T) {
	err := ErrorConfig{
		err: errors.New("a brand ne error test"),
		msg: "ERROR: This a test error.",
	}
	tests := []struct {
		name string
		e    ErrorConfig
		want string
	}{
		{
			name: "",
			e:    err,
			want: fmt.Sprintf("%s : %v", err.msg, err.err),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := err
			if got := e.Error(); got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetPortFromEnv(t *testing.T) {
	type args struct {
		defaultPort int
	}
	tests := []struct {
		name          string
		args          args
		envPORT       string
		want          string
		wantErr       bool
		wantErrPrefix string
	}{
		{
			name: "should return the default values when env variables are not set",
			args: args{
				defaultPort: defaultPort,
			},
			envPORT:       "",
			want:          ":8080",
			wantErr:       false,
			wantErrPrefix: "",
		},
		{
			name: "should return SERVERIP:PORT when env variables are set to valid values",
			args: args{
				defaultPort: 8080,
			},
			envPORT:       "3333",
			want:          ":3333",
			wantErr:       false,
			wantErrPrefix: "",
		},
		{
			name: "should return an empty string and report an error when PORT is not a number",
			args: args{
				defaultPort: 8080,
			},
			envPORT:       "aBigOne",
			want:          "",
			wantErr:       true,
			wantErrPrefix: "ERROR: CONFIG ENV PORT should contain a valid integer.",
		},
		{
			name: "should return an empty string and report an error when PORT is < 1",
			args: args{
				defaultPort: 8080,
			},
			envPORT:       "0",
			want:          "",
			wantErr:       true,
			wantErrPrefix: "ERROR: CONFIG ENV PORT should contain an integer between 1 and 65535",
		},
		{
			name: "should return an empty string and report an error when PORT is > 65535",
			args: args{
				defaultPort: 8080,
			},
			envPORT:       "70000",
			want:          "",
			wantErr:       true,
			wantErrPrefix: "ERROR: CONFIG ENV PORT should contain an integer between 1 and 65535",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.envPORT) > 0 {
				err := os.Setenv("PORT", tt.envPORT)
				if err != nil {
					t.Errorf("Unable to set env variable PORT")
					return
				}
			}
			got, err := GetPortFromEnv(tt.args.defaultPort)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetPortFromEnv() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				// check that error contains the ERROR keyword
				if strings.HasPrefix(err.Error(), "ERROR:") == false {
					t.Errorf("GetPortFromEnv() error = %v, wantErrPrefix %v", err, tt.wantErrPrefix)
				}
			}
			if got != tt.want {
				t.Errorf("GetPortFromEnv() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGoHttpServerMyDefaultHandler(t *testing.T) {
	var l *log.Logger
	var nameParameter string
	listenAddr := fmt.Sprintf(":%d", defaultPort)
	if DEBUG {
		l = log.New(os.Stdout, fmt.Sprintf("HTTP_SERVER_%s ", APP), log.Ldate|log.Ltime|log.Lshortfile)
	} else {
		l = log.New(ioutil.Discard, APP, 0)
	}

	myServer := NewGoHttpServer(listenAddr, l)
	ts := httptest.NewServer(myServer.getMyDefaultHandler())
	defer ts.Close()

	newRequest := func(method, url string, body string) *http.Request {
		r, err := http.NewRequest(method, ts.URL+url, strings.NewReader(body))
		if err != nil {
			t.Fatalf("### ERROR http.NewRequest %s on [%s] error is :%v\n", method, url, err)
		}
		return r
	}

	tests := []testStruct{
		{
			name:           "1: Get on default Server Path should return a valid json containing param value",
			wantStatusCode: http.StatusOK,
			wantBody:       `"param_name": "╚»☯💥⚡✌ℂ𝔾𝕀𝕃✌⚡💥☯«╝"`,
			paramKeyValues: map[string]string{"name": "╚»☯💥⚡✌ℂ𝔾𝕀𝕃✌⚡💥☯«╝"},
			r:              newRequest(http.MethodGet, defaultServerPath, ""),
		},
		{
			name:           "2: Get on default Server Path should return Http Status Ok",
			wantStatusCode: http.StatusOK,
			wantBody:       "",
			paramKeyValues: make(map[string]string, 0),
			r:              newRequest(http.MethodGet, defaultServerPath, ""),
		},
		{
			name:           "3: Post should return an http error method not allowed ",
			wantStatusCode: http.StatusMethodNotAllowed,
			wantBody:       "",
			paramKeyValues: make(map[string]string, 0),
			r:              newRequest(http.MethodPost, defaultServerPath, `{"task":"test not allowed method "}`),
		},
		{
			name:           "4: Get on unhandled path should return an http 404 Not Found",
			wantStatusCode: http.StatusNotFound,
			wantBody:       defaultNotFound,
			paramKeyValues: make(map[string]string, 0),
			r:              newRequest(http.MethodGet, "/a_funny_path_that_does_not_exist", ""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Header.Set("Content-Type", "application/json")
			if len(tt.paramKeyValues) > 0 {
				parameters := tt.r.URL.Query()
				for paramName, paramValue := range tt.paramKeyValues {
					parameters.Add(paramName, paramValue)
					if paramName == "name" {
						nameParameter = paramValue
					}
				}
				tt.r.URL.RawQuery = parameters.Encode()
			}
			resp, err := http.DefaultClient.Do(tt.r)
			if DEBUG {
				fmt.Printf("### %s : %s on %s\n", tt.name, tt.r.Method, tt.r.URL)
			}
			defer resp.Body.Close()
			if err != nil {
				fmt.Printf("### GOT ERROR : %s\n%s", err, resp.Body)
				t.Fatal(err)
			}
			assert.Equal(t, tt.wantStatusCode, resp.StatusCode, assertCorrectStatusCodeExpected)
			receivedJson, _ := ioutil.ReadAll(resp.Body)
			rInfo := &RuntimeInfo{}
			if DEBUG {
				fmt.Println("param name : % v", nameParameter)
				fmt.Printf("WANTED   :%T - %#v\n", tt.wantBody, tt.wantBody)
				fmt.Printf("RECEIVED :%T - %#v\n", receivedJson, string(receivedJson))
			}
			if tt.wantStatusCode == http.StatusOK {
				err = json.Unmarshal(receivedJson, rInfo)
				assert.Nil(t, err, "the output should be a valid json")
			}
			// check that receivedJson contains the specified tt.wantBody substring . https://pkg.go.dev/github.com/stretchr/testify/assert#Contains
			assert.Contains(t, string(receivedJson), tt.wantBody, "Response should contain what was expected.")
		})
	}
}

func TestGoHttpServerReadinessHandler(t *testing.T) {
	myServer := NewGoHttpServer(fmt.Sprintf(":%d", defaultPort), log.New(ioutil.Discard, APP, 0))
	ts := httptest.NewServer(myServer.getReadinessHandler())
	defer ts.Close()

	newRequest := func(method, url string, body string) *http.Request {
		r, err := http.NewRequest(method, ts.URL+url, strings.NewReader(body))
		if err != nil {
			t.Fatalf("### ERROR http.NewRequest %s on [%s] error is :%v\n", method, url, err)
		}
		return r
	}

	tests := []testStruct{
		{
			name:           "5: Get on readiness should return Http Status Ok",
			wantStatusCode: http.StatusOK,
			wantBody:       "",
			paramKeyValues: make(map[string]string, 0),
			r:              newRequest(http.MethodGet, "/readiness", ""),
		},
		{
			name:           "6: Post  on readiness should return an http error method not allowed ",
			wantStatusCode: http.StatusMethodNotAllowed,
			wantBody:       "",
			paramKeyValues: make(map[string]string, 0),
			r:              newRequest(http.MethodPost, "/readiness", `{"task":"test not allowed method "}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Header.Set(HeaderContentType, MIMEAppJSONCharsetUTF8)
			resp, err := http.DefaultClient.Do(tt.r)
			if DEBUG {
				fmt.Printf("### %s : %s on %s\n", tt.name, tt.r.Method, tt.r.URL)
			}
			defer resp.Body.Close()
			if err != nil {
				fmt.Printf("### GOT ERROR : %s\n%s", err, resp.Body)
				t.Fatal(err)
			}
			assert.Equal(t, tt.wantStatusCode, resp.StatusCode, assertCorrectStatusCodeExpected)
			receivedJson, _ := ioutil.ReadAll(resp.Body)

			if DEBUG {
				printWantedReceived(tt, receivedJson)
			}
			// check that receivedJson contains the specified tt.wantBody substring . https://pkg.go.dev/github.com/stretchr/testify/assert#Contains
			assert.Contains(t, string(receivedJson), tt.wantBody, "Response should contain what was expected.")
		})
	}
}

func printWantedReceived(tt testStruct, receivedJson []byte) {
	fmt.Printf("WANTED   :%T - %#v\n", tt.wantBody, tt.wantBody)
	fmt.Printf("RECEIVED :%T - %#v\n", receivedJson, string(receivedJson))
}

func TestGoHttpServerHealthHandler(t *testing.T) {
	myServer := NewGoHttpServer(fmt.Sprintf(":%d", defaultPort), log.New(ioutil.Discard, APP, 0))
	ts := httptest.NewServer(myServer.getHealthHandler())
	defer ts.Close()

	newRequest := func(method, url string, body string) *http.Request {
		r, err := http.NewRequest(method, ts.URL+url, strings.NewReader(body))
		if err != nil {
			t.Fatalf("### ERROR http.NewRequest %s on [%s] error is :%v\n", method, url, err)
		}
		return r
	}

	tests := []testStruct{
		{
			name:           "1: Get on health should return Http Status Ok",
			wantStatusCode: http.StatusOK,
			wantBody:       "",
			paramKeyValues: make(map[string]string, 0),
			r:              newRequest(http.MethodGet, "/health", ""),
		},
		{
			name:           "2: Post on health should return an http error method not allowed ",
			wantStatusCode: http.StatusMethodNotAllowed,
			wantBody:       "",
			paramKeyValues: make(map[string]string, 0),
			r:              newRequest(http.MethodPost, "/health", `{"task":"test not allowed method "}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Header.Set(HeaderContentType, MIMEAppJSONCharsetUTF8)
			resp, err := http.DefaultClient.Do(tt.r)
			if DEBUG {
				fmt.Printf("### %s : %s on %s\n", tt.name, tt.r.Method, tt.r.URL)
			}
			defer resp.Body.Close()
			if err != nil {
				fmt.Printf("### GOT ERROR : %s\n%s", err, resp.Body)
				t.Fatal(err)
			}
			assert.Equal(t, tt.wantStatusCode, resp.StatusCode, assertCorrectStatusCodeExpected)
			receivedJson, _ := ioutil.ReadAll(resp.Body)

			if DEBUG {
				fmt.Printf("WANTED   :%T - %#v\n", tt.wantBody, tt.wantBody)
				fmt.Printf("RECEIVED :%T - %#v\n", receivedJson, string(receivedJson))
			}
			// check that receivedJson contains the specified tt.wantBody substring . https://pkg.go.dev/github.com/stretchr/testify/assert#Contains
			assert.Contains(t, string(receivedJson), tt.wantBody, "Response should contain what was expected.")
		})
	}
}

func TestGoHttpServerTimeHandler(t *testing.T) {
	myServer := NewGoHttpServer(fmt.Sprintf(":%d", defaultPort), log.New(ioutil.Discard, APP, 0))
	ts := httptest.NewServer(myServer.getTimeHandler())
	defer ts.Close()
	now := time.Now()
	expectedResult := fmt.Sprintf("{\"time\":\"%s\"}", now.Format(time.RFC3339))

	newRequest := func(method, url string, body string) *http.Request {
		r, err := http.NewRequest(method, ts.URL+url, strings.NewReader(body))
		if err != nil {
			t.Fatalf("### ERROR http.NewRequest %s on [%s] error is :%v\n", method, url, err)
		}
		return r
	}

	tests := []testStruct{
		{
			name:           "1: Get on time should return Http Status Ok",
			wantStatusCode: http.StatusOK,
			wantBody:       expectedResult,
			paramKeyValues: make(map[string]string, 0),
			r:              newRequest(http.MethodGet, "/time", ""),
		},
		{
			name:           "2: Post on time should return an http error method not allowed ",
			wantStatusCode: http.StatusMethodNotAllowed,
			wantBody:       "",
			paramKeyValues: make(map[string]string, 0),
			r:              newRequest(http.MethodPost, "/time", `{"task":"test not allowed method "}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Header.Set(HeaderContentType, MIMEAppJSONCharsetUTF8)
			resp, err := http.DefaultClient.Do(tt.r)
			if DEBUG {
				fmt.Printf("### %s : %s on %s\n", tt.name, tt.r.Method, tt.r.URL)
			}
			defer resp.Body.Close()
			if err != nil {
				fmt.Printf("### GOT ERROR : %s\n%s", err, resp.Body)
				t.Fatal(err)
			}
			assert.Equal(t, tt.wantStatusCode, resp.StatusCode, assertCorrectStatusCodeExpected)
			receivedJson, _ := ioutil.ReadAll(resp.Body)

			if DEBUG {
				fmt.Printf("WANTED   :%T - %#v\n", tt.wantBody, tt.wantBody)
				fmt.Printf("RECEIVED :%T - %#v\n", receivedJson, string(receivedJson))
			}
			// check that receivedJson contains the specified tt.wantBody substring . https://pkg.go.dev/github.com/stretchr/testify/assert#Contains
			assert.Contains(t, string(receivedJson), tt.wantBody, "Response should contain what was expected.")
		})
	}
}

func TestGoHttpServerWaitHandler(t *testing.T) {
	myServer := NewGoHttpServer(fmt.Sprintf(":%d", defaultPort), log.New(ioutil.Discard, APP, 0))
	ts := httptest.NewServer(myServer.getWaitHandler(1))
	defer ts.Close()
	expectedResult := fmt.Sprintf("{\"waited\":\"%v seconds\"}", 1)

	newRequest := func(method, url string, body string) *http.Request {
		r, err := http.NewRequest(method, ts.URL+url, strings.NewReader(body))
		if err != nil {
			t.Fatalf("### ERROR http.NewRequest %s on [%s] error is :%v\n", method, url, err)
		}
		return r
	}

	tests := []testStruct{
		{
			name:           "1: Get on /wait should return Http Status Ok",
			wantStatusCode: http.StatusOK,
			wantBody:       expectedResult,
			paramKeyValues: make(map[string]string, 0),
			r:              newRequest(http.MethodGet, "/wait", ""),
		},
		{
			name:           "2: Post on /wait should return an http error method not allowed ",
			wantStatusCode: http.StatusMethodNotAllowed,
			wantBody:       "",
			paramKeyValues: make(map[string]string, 0),
			r:              newRequest(http.MethodPost, "/wait", `{"task":"test not allowed method "}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.r.Header.Set(HeaderContentType, MIMEAppJSONCharsetUTF8)
			resp, err := http.DefaultClient.Do(tt.r)
			if DEBUG {
				fmt.Printf("### %s : %s on %s\n", tt.name, tt.r.Method, tt.r.URL)
			}
			defer resp.Body.Close()
			if err != nil {
				fmt.Printf("### GOT ERROR : %s\n%s", err, resp.Body)
				t.Fatal(err)
			}
			assert.Equal(t, tt.wantStatusCode, resp.StatusCode, assertCorrectStatusCodeExpected)
			receivedJson, _ := ioutil.ReadAll(resp.Body)

			if DEBUG {
				fmt.Printf("WANTED   :%T - %#v\n", tt.wantBody, tt.wantBody)
				fmt.Printf("RECEIVED :%T - %#v\n", receivedJson, string(receivedJson))
			}
			// check that receivedJson contains the specified tt.wantBody substring . https://pkg.go.dev/github.com/stretchr/testify/assert#Contains
			assert.Contains(t, string(receivedJson), tt.wantBody, "Response should contain what was expected.")
		})
	}
}

func TestMainExecution(t *testing.T) {
	listenAddr := fmt.Sprintf("%s://%s:%d%s", defaultProtocol, defaultServerIp, defaultPort, defaultServerPath)
	err := os.Setenv("PORT", fmt.Sprintf("%d", defaultPort))
	if err != nil {
		t.Errorf("Unable to set env variable PORT")
		return
	}
	// starting main in his own go routine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		main()
	}()
	WaitForHttpServer(listenAddr, 1*time.Second, 10)

	resp, err := http.Get(listenAddr)
	if err != nil {
		t.Fatalf("Cannot make http get: %v\n", err)
	}
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Should return an http status ok")

	receivedJson, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error reading response body: %v\n", err)
	}
	var decodedResponse OsInfo
	err = json.Unmarshal(receivedJson, &decodedResponse)
	assert.Nil(t, err, "the output should be a valid json")
	if err != nil {
		t.Fatalf("Cannot decode response <%p> from server. Err: %v", receivedJson, err)
	}

	// check that receivedJson contains the specified tt.wantBody substring . https://pkg.go.dev/github.com/stretchr/testify/assert#Contains
	assert.Contains(t, string(receivedJson), fmt.Sprintf("\"appname\": \"%s\"", APP), "Response should contain the appname field.")
	assert.Contains(t, string(receivedJson), "\"request_id\":", "Response should contain the request_id field.")

}
