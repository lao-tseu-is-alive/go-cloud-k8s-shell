package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/xid"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	VERSION                = "0.1.26"
	APP                    = "go-cloud-k8s-shell"
	AppCamelCase           = "goCloudK8sShell"
	AppGithubUrl           = "https://github.com/lao-tseu-is-alive/go-cloud-k8s-shell"
	defaultProtocol        = "http"
	defaultPort            = 9898
	defaultServerIp        = ""
	defaultServerPath      = "/"
	defaultSecondsToSleep  = 3
	secondsShutDownTimeout = 5 * time.Second  // maximum number of second to wait before closing server
	defaultReadTimeout     = 10 * time.Second // max time to read request from the client
	defaultWriteTimeout    = 10 * time.Second // max time to write response to the client
	defaultIdleTimeout     = 2 * time.Minute  // max time for connections using TCP Keep-Alive
	caCertPath             = "certificates/isrg-root-x1-cross-signed.pem"
	htmlHeaderStart        = `<!DOCTYPE html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/skeleton/2.0.4/skeleton.min.css"/>`
	charsetUTF8            = "charset=UTF-8"
	MIMEAppJSON            = "application/json"
	MIMEHtml               = "text/html"
	MIMEAppJSONCharsetUTF8 = MIMEAppJSON + "; " + charsetUTF8
	HeaderContentType      = "Content-Type"
	httpErrMethodNotAllow  = "ERROR: Http method not allowed"
	initCallMsg            = "INITIAL CALL TO %s()\n"
	defaultUnknown         = "_UNKNOWN_"
	// defaultUnknown         = "¯\\_( ͡° ͜ʖ ͡°)_/¯"
	defaultNotFound                 = "404 page not found"
	defaultNotFoundDescription      = "🤔 ℍ𝕞𝕞... 𝕤𝕠𝕣𝕣𝕪 :【𝟜𝟘𝟜 : ℙ𝕒𝕘𝕖 ℕ𝕠𝕥 𝔽𝕠𝕦𝕟𝕕】🕳️ 🔥"
	fmtErrK8sServiceHostEnvNotFound = "ERROR: KUBERNETES_SERVICE_HOST ENV variable does not exist (not inside K8s ?)."
	formatTraceRequest              = "TRACE: [%s] %s  path:'%s', RemoteAddrIP: [%s]\n"
	formatErrRequest                = "ERROR: Http method not allowed [%s] %s  path:'%s', RemoteAddrIP: [%s]\n"
)

var rootPathGetCounter = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: fmt.Sprintf("%s_root_get_request_count", AppCamelCase),
		Help: fmt.Sprintf("Number of GET request handled by %s default root handler", AppCamelCase),
	},
)

var rootPathNotFoundCounter = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: fmt.Sprintf("%s_root_not_found_request_count", AppCamelCase),
		Help: fmt.Sprintf("Number of page not found handled by %s default root handler", AppCamelCase),
	},
)

type Middleware interface {
	// WrapHandler wraps the given HTTP handler for instrumentation.
	WrapHandler(handlerName string, handler http.Handler) http.HandlerFunc
}

type middleware struct {
	buckets  []float64
	registry prometheus.Registerer
}

// WrapHandler wraps the given HTTP handler for instrumentation:
// It registers four metric collectors (if not already done) and reports HTTP
// metrics to the (newly or already) registered collectors.
// Each has a constant label named "handler" with the provided handlerName as
// value.
func (m *middleware) WrapHandler(handlerName string, handler http.Handler) http.HandlerFunc {
	reg := prometheus.WrapRegistererWith(prometheus.Labels{"handler": handlerName}, m.registry)

	requestsTotal := promauto.With(reg).NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Tracks the number of HTTP requests.",
		}, []string{"method", "code"},
	)
	requestDuration := promauto.With(reg).NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Tracks the latencies for HTTP requests.",
			Buckets: m.buckets,
		},
		[]string{"method", "code"},
	)
	requestSize := promauto.With(reg).NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "http_request_size_bytes",
			Help: "Tracks the size of HTTP requests.",
		},
		[]string{"method", "code"},
	)
	responseSize := promauto.With(reg).NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "http_response_size_bytes",
			Help: "Tracks the size of HTTP responses.",
		},
		[]string{"method", "code"},
	)

	// Wraps the provided http.Handler to observe the request result with the provided metrics.
	base := promhttp.InstrumentHandlerCounter(
		requestsTotal,
		promhttp.InstrumentHandlerDuration(
			requestDuration,
			promhttp.InstrumentHandlerRequestSize(
				requestSize,
				promhttp.InstrumentHandlerResponseSize(
					responseSize,
					handler,
				),
			),
		),
	)

	return base.ServeHTTP
}

// NewMiddleware returns a Middleware interface.
func NewMiddleware(registry prometheus.Registerer, buckets []float64) Middleware {
	if buckets == nil {
		buckets = prometheus.ExponentialBuckets(0.1, 1.5, 5)
	}

	return &middleware{
		buckets:  buckets,
		registry: registry,
	}
}

type RuntimeInfo struct {
	Hostname            string              `json:"hostname"`              // host name reported by the kernel.
	Pid                 int                 `json:"pid"`                   // process id of the caller.
	PPid                int                 `json:"ppid"`                  // process id of the caller's parent.
	Uid                 int                 `json:"uid"`                   // numeric user id of the caller.
	Appname             string              `json:"appname"`               // name of this application
	Version             string              `json:"version"`               // version of this application
	ParamName           string              `json:"param_name"`            // value of the name parameter (_NO_PARAMETER_NAME_ if name was not set)
	RemoteAddr          string              `json:"remote_addr"`           // remote client ip address
	RequestId           string              `json:"request_id"`            // globally unique request id
	GOOS                string              `json:"goos"`                  // operating system
	GOARCH              string              `json:"goarch"`                // architecture
	Runtime             string              `json:"runtime"`               // go runtime at compilation time
	NumGoroutine        string              `json:"num_goroutine"`         // number of go routines
	OsReleaseName       string              `json:"os_release_name"`       // Linux release Name or _UNKNOWN_
	OsReleaseVersion    string              `json:"os_release_version"`    // Linux release Version or _UNKNOWN_
	OsReleaseVersionId  string              `json:"os_release_version_id"` // Linux release VersionId or _UNKNOWN_
	NumCPU              string              `json:"num_cpu"`               // number of cpu
	Uptime              string              `json:"uptime"`                // tells how long this service was started based on an internal variable
	UptimeOs            string              `json:"uptime_os"`             // tells how long system was started based on /proc/uptime
	K8sApiUrl           string              `json:"k8s_api_url"`           // url for k8s api based KUBERNETES_SERVICE_HOST
	K8sVersion          string              `json:"k8s_version"`           // version of k8s cluster
	K8sLatestVersion    string              `json:"k8s_latest_version"`    // latest version announced in https://kubernetes.io/
	K8sCurrentNamespace string              `json:"k8s_current_namespace"` // k8s namespace of this container
	EnvVars             []string            `json:"env_vars"`              // environment variables
	Headers             map[string][]string `json:"headers"`               // received headers
}

type ErrorConfig struct {
	err error
	msg string
}

// Error returns a string with an error and a specifics message
func (e *ErrorConfig) Error() string {
	return fmt.Sprintf("%s : %v", e.msg, e.err)
}

type OsInfo struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	VersionId string `json:"versionId"`
}

type K8sInfo struct {
	CurrentNamespace string `json:"current_namespace"`
	Version          string `json:"version"`
	Token            string `json:"token"`
	CaCert           string `json:"ca_cert"`
}

func CloseBody(Body io.ReadCloser, msg string, logger *log.Logger) {
	err := Body.Close()
	if err != nil {
		logger.Printf("Error %v in %s doing Body.Close().\n", err, msg)
	}
}

func GetOsUptime() (string, error) {
	uptimeResult := defaultUnknown
	content, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return uptimeResult, err
	}
	uptimeResult = string(content)
	return uptimeResult, nil
}

func GetOsInfo() (*OsInfo, ErrorConfig) {
	const (
		OsReleasePath          = "/etc/os-release"
		regexFindOsNameVersion = `(?m)^NAME="(?P<name>[^"]+)"\s?|^VERSION="(?P<version>[^"]+)"|^VERSION_ID="?(?P<versid>[^"]+)"?\s`
	)
	info := OsInfo{
		Name:      defaultUnknown,
		Version:   defaultUnknown,
		VersionId: defaultUnknown,
	}
	content, err := os.ReadFile(OsReleasePath)
	if err != nil {
		return &info, ErrorConfig{
			err: err,
			msg: "GetOsInfo: error reading " + OsReleasePath,
		}
	}
	r := regexp.MustCompile(regexFindOsNameVersion)
	matches := r.FindAllStringSubmatch(string(content), -1)

	for _, match := range matches {
		for i, name := range r.SubexpNames() {
			if i > 0 && match[i] != "" {
				switch name {
				case "name":
					info.Name = match[i]
				case "version":
					info.Version = match[i]
				case "versid":
					info.VersionId = match[i]
				}
			}
		}
	}
	return &info, ErrorConfig{
		err: nil,
		msg: "",
	}
}

// GetPortFromEnv returns a valid TCP/IP listening ':PORT' string based on the values of environment variable :
//
//		PORT : int value between 1 and 65535 (the parameter defaultPort will be used if env is not defined)
//	 in case the ENV variable PORT exists and contains an invalid integer the functions returns an empty string and an error
func GetPortFromEnv(defaultPort int) (string, error) {
	srvPort := defaultPort

	var err error
	val, exist := os.LookupEnv("PORT")
	if exist {
		srvPort, err = strconv.Atoi(val)
		if err != nil {
			return "", &ErrorConfig{
				err: err,
				msg: "ERROR: CONFIG ENV PORT should contain a valid integer.",
			}
		}
		if srvPort < 1 || srvPort > 65535 {
			return "", &ErrorConfig{
				err: err,
				msg: "ERROR: CONFIG ENV PORT should contain an integer between 1 and 65535",
			}
		}
	}
	return fmt.Sprintf(":%d", srvPort), nil
}

// GetKubernetesApiUrlFromEnv returns the k8s api url based on the content of standard env var :
//
//	KUBERNETES_SERVICE_HOST
//	KUBERNETES_SERVICE_PORT
//	in case the above ENV variables doesn't  exist the function returns an empty string and an error
func GetKubernetesApiUrlFromEnv() (string, error) {
	srvPort := 443
	k8sApiUrl := "https://"

	var err error
	val, exist := os.LookupEnv("KUBERNETES_SERVICE_HOST")
	if !exist {
		return "", &ErrorConfig{
			err: err,
			msg: fmtErrK8sServiceHostEnvNotFound,
		}
	}
	k8sApiUrl = fmt.Sprintf("%s%s", k8sApiUrl, val)
	val, exist = os.LookupEnv("KUBERNETES_SERVICE_PORT")
	if exist {
		srvPort, err = strconv.Atoi(val)
		if err != nil {
			return "", &ErrorConfig{
				err: err,
				msg: "ERROR: CONFIG ENV PORT should contain a valid integer.",
			}
		}
		if srvPort < 1 || srvPort > 65535 {
			return "", &ErrorConfig{
				err: err,
				msg: "ERROR: CONFIG ENV PORT should contain an integer between 1 and 65535",
			}
		}
	}
	return fmt.Sprintf("%s:%d", k8sApiUrl, srvPort), nil
}

func GetKubernetesConnInfo(logger *log.Logger) (*K8sInfo, ErrorConfig) {
	const K8sServiceAccountPath = "/var/run/secrets/kubernetes.io/serviceaccount"
	K8sNamespacePath := fmt.Sprintf("%s/namespace", K8sServiceAccountPath)
	K8sTokenPath := fmt.Sprintf("%s/token", K8sServiceAccountPath)
	K8sCaCertPath := fmt.Sprintf("%s/ca.crt", K8sServiceAccountPath)

	info := K8sInfo{
		CurrentNamespace: "",
		Version:          "",
		Token:            "",
		CaCert:           "",
	}

	K8sNamespace, err := os.ReadFile(K8sNamespacePath)
	if err != nil {
		return &info, ErrorConfig{
			err: err,
			msg: "GetKubernetesConnInfo: error reading namespace in " + K8sNamespacePath,
		}
	}
	info.CurrentNamespace = string(K8sNamespace)

	K8sToken, err := os.ReadFile(K8sTokenPath)
	if err != nil {
		return &info, ErrorConfig{
			err: err,
			msg: "GetKubernetesConnInfo: error reading token in " + K8sTokenPath,
		}
	}
	info.Token = string(K8sToken)

	K8sCaCert, err := os.ReadFile(K8sCaCertPath)
	if err != nil {
		return &info, ErrorConfig{
			err: err,
			msg: "GetKubernetesConnInfo: error reading Ca Cert in " + K8sCaCertPath,
		}
	}
	info.CaCert = string(K8sCaCert)

	k8sUrl, err := GetKubernetesApiUrlFromEnv()
	if err != nil {
		return &info, ErrorConfig{
			err: err,
			msg: "GetKubernetesConnInfo: error reading GetKubernetesApiUrlFromEnv ",
		}
	}
	urlVersion := fmt.Sprintf("%s/openapi/v2", k8sUrl)
	res, err := GetJsonFromUrl(urlVersion, info.Token, K8sCaCert, true, logger)
	if err != nil {

		logger.Printf("GetKubernetesConnInfo: error in GetJsonFromUrl(url:%s) err:%v", urlVersion, err)
		//return &info, ErrorConfig{
		//	err: err,
		//	msg: fmt.Sprintf("GetKubernetesConnInfo: error doing GetJsonFromUrl(url:%s)", urlVersion),
		//}
	} else {
		logger.Printf("GetKubernetesConnInfo: successfully returned from GetJsonFromUrl(url:%s)", urlVersion)
		var myVersionRegex = regexp.MustCompile("{\"title\":\"(?P<title>.+)\",\"version\":\"(?P<version>.+)\"}")
		match := myVersionRegex.FindStringSubmatch(strings.TrimSpace(res[:150]))
		k8sVersionFields := make(map[string]string)
		for i, name := range myVersionRegex.SubexpNames() {
			if i != 0 && name != "" {
				k8sVersionFields[name] = match[i]
			}
		}
		info.Version = fmt.Sprintf("%s, %s", k8sVersionFields["title"], k8sVersionFields["version"])
	}

	return &info, ErrorConfig{
		err: nil,
		msg: "",
	}
}

func GetJsonFromUrl(url string, bearerToken string, caCert []byte, allowInsecure bool, logger *log.Logger) (string, error) {
	// Create a Bearer string by appending string access token
	var bearer = "Bearer " + bearerToken

	// Create a new request using http
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Printf("Error on http.NewRequest [ERROR: %v]\n", err)
		return "", err
	}

	// add authorization header to the req
	req.Header.Add("Authorization", bearer)
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:            caCertPool,
			InsecureSkipVerify: allowInsecure,
		},
	}
	// Send req using http Client
	client := &http.Client{
		Transport: tr,
		Timeout:   defaultReadTimeout,
	}
	resp, err := client.Do(req)
	if err != nil {
		logger.Println("Error on sending request.\n[ERROR] -", err)
		return "", err
	}
	defer CloseBody(resp.Body, "GetJsonFromUrl", logger)
	if resp.StatusCode != http.StatusOK {
		logger.Printf("Error on response StatusCode is not OK Received StatusCode:%d\n", resp.StatusCode)
		return "", errors.New(fmt.Sprintf("Error on response StatusCode:%d\n", resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Println("Error while reading the response bytes:", err)
		return "", err
	}
	return string(body), nil
}
func getKubernetesInfo(l *log.Logger) (string, string, string) {
	k8sVersion := ""
	k8sCurrentNameSpace := ""
	k8sUrl := ""

	k8sUrl, err := GetKubernetesApiUrlFromEnv()
	if err != nil {
		l.Printf("💥💥 ERROR: 'GetKubernetesApiUrlFromEnv() returned an error : %+#v'", err)
	} else {
		info, errConnInfo := GetKubernetesConnInfo(l)
		if errConnInfo.err != nil {
			l.Printf("💥💥 ERROR: 'GetKubernetesConnInfo() returned an error : %s : %+#v'", errConnInfo.msg, errConnInfo.err)
		}
		k8sVersion = info.Version
		k8sCurrentNameSpace = info.CurrentNamespace
	}

	return k8sUrl, k8sVersion, k8sCurrentNameSpace
}

func GetKubernetesLatestVersion(logger *log.Logger) (string, error) {
	k8sUrl := "https://kubernetes.io/"
	// Make an HTTP GET request to the Kubernetes releases page
	// Create a new request using http
	req, err := http.NewRequest("GET", k8sUrl, nil)
	if err != nil {
		logger.Printf("Error on http.NewRequest [ERROR: %v]\n", err)
		return "", err
	}
	caCert, err := os.ReadFile(caCertPath)
	if err != nil {
		logger.Printf("Error on ReadFile(caCertPath) [ERROR: %v]\n", err)
		return "", err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: caCertPool,
		},
	}

	//tr := &http.Transport{ TLSClientConfig: &tls.Config{InsecureSkipVerify: true} }

	// add authorization header to the req
	// req.Header.Add("Authorization", bearer)
	// Send req using http Client
	client := &http.Client{
		Timeout:   defaultReadTimeout,
		Transport: tr,
	}

	resp, err := client.Do(req)

	if err != nil {
		logger.Println("Error on response.\n[ERROR] -", err)
		return fmt.Sprintf("GetKubernetesLatestVersion was unable to get content from %s, Error: %v", k8sUrl, err), err
	}
	defer CloseBody(resp.Body, "GetKubernetesLatestVersion", logger)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Println("Error while reading the response bytes:", err)
		return fmt.Sprintf("GetKubernetesLatestVersion got a problem reading the response from %s, Error: %v", k8sUrl, err), err
	}
	// Use a regular expression to extract the latest release number from the page
	re := regexp.MustCompile(`(?m)href=.+?>v(\d+\.\d+)`)
	matches := re.FindAllStringSubmatch(string(body), -1)
	if matches == nil {
		return fmt.Sprintf("GetKubernetesLatestVersion was unable to find latest release number from %s", k8sUrl), nil
	}
	// Print only the release numbers
	maxVersion := 0.0
	for _, match := range matches {
		// fmt.Println(match[1])
		if val, err := strconv.ParseFloat(match[1], 32); err == nil {
			if val > maxVersion {
				maxVersion = val
			}
		}
	}
	// latestRelease := matches[0]
	// fmt.Printf("\nThe latest major release of Kubernetes is %T : %v+", latestRelease, latestRelease)
	return fmt.Sprintf("%2.2f", maxVersion), nil
}

func handleOsInfoError(errConf ErrorConfig, l *log.Logger) {
	var pathError *fs.PathError
	switch {
	case errors.As(errConf.err, &pathError):
		l.Printf("NOTICE: 'GetOsInfo() did not find os-release : %v'", errConf.err)
	default:
		l.Printf("💥💥 ERROR: 'GetOsInfo() returned an error : %+#v'", errConf.err)
	}
}

func collectRuntimeInfo(l *log.Logger) RuntimeInfo {
	hostName, err := os.Hostname()
	if err != nil {
		l.Printf("💥💥 ERROR: 'os.Hostname() returned an error : %v'", err)
		hostName = "#unknown#"
	}

	osReleaseInfo, errConf := GetOsInfo()
	if errConf.err != nil {
		handleOsInfoError(errConf, l)
	}

	uptimeOS, err := GetOsUptime()
	if err != nil {
		l.Printf("💥💥 ERROR: 'GetOsUptime() returned an error : %+#v'", err)
	}

	k8sApiUrl, k8sVersion, k8sCurrentNameSpace := getKubernetesInfo(l)

	latestK8sVersion, err := GetKubernetesLatestVersion(l)
	if err != nil {
		l.Printf("💥💥 ERROR: 'GetKubernetesLatestVersion() returned an error : %+#v'", err)
	}

	return RuntimeInfo{
		Hostname:            hostName,
		Pid:                 os.Getpid(),
		PPid:                os.Getppid(),
		Uid:                 os.Getuid(),
		Appname:             APP,
		Version:             VERSION,
		ParamName:           "_NO_PARAMETER_NAME_",
		RemoteAddr:          "",
		RequestId:           "",
		GOOS:                runtime.GOOS,
		GOARCH:              runtime.GOARCH,
		Runtime:             runtime.Version(),
		NumGoroutine:        strconv.FormatInt(int64(runtime.NumGoroutine()), 10),
		OsReleaseName:       osReleaseInfo.Name,
		OsReleaseVersion:    osReleaseInfo.Version,
		OsReleaseVersionId:  osReleaseInfo.VersionId,
		NumCPU:              strconv.FormatInt(int64(runtime.NumCPU()), 10),
		Uptime:              "",
		UptimeOs:            uptimeOS,
		K8sApiUrl:           k8sApiUrl,
		K8sVersion:          k8sVersion,
		K8sLatestVersion:    latestK8sVersion,
		K8sCurrentNamespace: k8sCurrentNameSpace,
		EnvVars:             os.Environ(),
		Headers:             map[string][]string{},
	}
}

func getHtmlHeader(title string, description string) string {
	return fmt.Sprintf("%s<meta name=\"description\" content=\"%s\"><title>%s</title></head>", htmlHeaderStart, description, title)
}

func getHtmlPage(title string, description string) string {
	return getHtmlHeader(title, description) +
		fmt.Sprintf("\n<body><div class=\"container\"><h4>%s</h4></div></body></html>", title)
}

// WaitForHttpServer attempts to establish a TCP connection to listenAddress
// in a given amount of time. It returns upon a successful connection;
// otherwise exits with an error.
func WaitForHttpServer(listenAddress string, waitDuration time.Duration, numRetries int) {
	log.Printf("INFO: 'WaitForHttpServer Will wait for server to be up at %s for %v seconds, with %d retries'\n", listenAddress, waitDuration.Seconds(), numRetries)
	httpClient := http.Client{
		Timeout: 5 * time.Second,
	}
	for i := 0; i < numRetries; i++ {
		//conn, err := net.DialTimeout("tcp", listenAddress, dialTimeout)
		resp, err := httpClient.Get(listenAddress)
		if err != nil {
			fmt.Printf("\n[%d] Cannot make http get %s: %v\n", i, listenAddress, err)
			time.Sleep(waitDuration)
			continue
		}
		// All seems is good
		fmt.Printf("OK: Server responded after %d retries, with status code %d ", i, resp.StatusCode)
		return
	}
	log.Fatalf("Server %s not ready up after %d attempts", listenAddress, numRetries)
}

// waitForShutdownToExit will wait for interrupt signal SIGINT or SIGTERM and gracefully shutdown the server after secondsToWait seconds.
func waitForShutdownToExit(srv *http.Server, secondsToWait time.Duration) {
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received.
	// wait for SIGINT (interrupt) 	: ctrl + C keypress, or in a shell : kill -SIGINT processId
	sig := <-interruptChan
	srv.ErrorLog.Printf("INFO: 'SIGINT %d interrupt signal received, about to shut down server after max %v seconds...'\n", sig, secondsToWait.Seconds())

	// create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), secondsToWait)
	defer cancel()
	// gracefully shuts down the server without interrupting any active connections
	// as long as the actives connections last less than shutDownTimeout
	// https://pkg.go.dev/net/http#Server.Shutdown
	if err := srv.Shutdown(ctx); err != nil {
		srv.ErrorLog.Printf("💥💥 ERROR: 'Problem doing Shutdown %v'\n", err)
	}
	<-ctx.Done()
	srv.ErrorLog.Println("INFO: 'Server gracefully stopped, will exit'")
	os.Exit(0)
}

// GoHttpServer is a struct type to store information related to all handlers of web server
type GoHttpServer struct {
	listenAddress string
	// later we will store here the connection to database
	//DB  *db.Conn
	logger     *log.Logger
	router     *http.ServeMux
	registry   *prometheus.Registry
	startTime  time.Time
	httpServer http.Server
}

// NewGoHttpServer is a constructor that initializes the server mux (routes) and all fields of the  GoHttpServer type
func NewGoHttpServer(listenAddress string, logger *log.Logger) *GoHttpServer {
	myServerMux := http.NewServeMux()
	// Create non-global registry.
	registry := prometheus.NewRegistry()

	// Add go runtime metrics and process collectors.
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	registry.MustRegister(rootPathGetCounter)
	registry.MustRegister(rootPathNotFoundCounter)

	myServer := GoHttpServer{
		listenAddress: listenAddress,
		logger:        logger,
		router:        myServerMux,
		registry:      registry,
		startTime:     time.Now(),
		httpServer: http.Server{
			Addr:         listenAddress,       // configure the bind address
			Handler:      myServerMux,         // set the http mux
			ErrorLog:     logger,              // set the logger for the server
			ReadTimeout:  defaultReadTimeout,  // max time to read request from the client
			WriteTimeout: defaultWriteTimeout, // max time to write response to the client
			IdleTimeout:  defaultIdleTimeout,  // max time for connections using TCP Keep-Alive
		},
	}
	myServer.routes()

	return &myServer
}

// (*GoHttpServer) routes initializes all the handlers paths of this web server, it is called inside the NewGoHttpServer constructor
func (s *GoHttpServer) routes() {
	// using new server Mux in Go 1.22 https://pkg.go.dev/net/http#ServeMux
	s.router.Handle("GET /{$}", NewMiddleware(
		s.registry, nil).
		WrapHandler("GET /$", s.getMyDefaultHandler()),
	)
	s.router.Handle("GET /time", s.getTimeHandler())
	s.router.Handle("GET /wait", s.getWaitHandler(defaultSecondsToSleep))
	s.router.Handle("GET /readiness", s.getReadinessHandler())
	s.router.Handle("GET /health", s.getHealthHandler())
	//expose the default prometheus metrics for Go applications
	s.router.Handle("GET /metrics", NewMiddleware(
		s.registry, nil).
		WrapHandler("GET /metrics", promhttp.HandlerFor(
			s.registry,
			promhttp.HandlerOpts{}),
		))

	s.router.Handle("GET /...", s.getHandlerNotFound())
}

// AddRoute   adds a handler for this web server
func (s *GoHttpServer) AddRoute(pathPattern string, handler http.Handler) {
	s.router.Handle(pathPattern, handler)
}

// StartServer initializes all the handlers paths of this web server, it is called inside the NewGoHttpServer constructor
func (s *GoHttpServer) StartServer() {

	// Starting the web server in his own goroutine
	go func() {
		s.logger.Printf("INFO: Starting http server listening at %s://%s/", defaultProtocol, s.listenAddress)
		err := s.httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Fatalf("💥💥 ERROR: 'Could not listen on %q: %s'\n", s.listenAddress, err)
		}
	}()
	s.logger.Printf("Server listening on : %s PID:[%d]", s.httpServer.Addr, os.Getpid())

	// Graceful Shutdown on SIGINT (interrupt)
	waitForShutdownToExit(&s.httpServer, secondsShutDownTimeout)

}

func (s *GoHttpServer) jsonResponse(w http.ResponseWriter, result interface{}) error {
	body, err := json.Marshal(result)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.logger.Printf("ERROR: 'JSON marshal failed. Error: %v'", err)
		return err
	}
	var prettyOutput bytes.Buffer
	err = json.Indent(&prettyOutput, body, "", "  ")
	if err != nil {
		s.logger.Printf("ERROR: 'JSON Indent failed. Error: %v'", err)
		return err
	}
	w.Header().Set(HeaderContentType, MIMEAppJSONCharsetUTF8)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(prettyOutput.Bytes())
	if err != nil {
		s.logger.Printf("ERROR: 'w.Write failed. Error: %v'", err)
		return err
	}
	return nil
}

//############# BEGIN HANDLERS

func (s *GoHttpServer) getReadinessHandler() http.HandlerFunc {
	handlerName := "getReadinessHandler"
	s.logger.Printf(initCallMsg, handlerName)
	return func(w http.ResponseWriter, r *http.Request) {
		s.logger.Printf(formatTraceRequest, handlerName, r.Method, r.URL.Path, r.RemoteAddr)
		w.WriteHeader(http.StatusOK)
	}
}
func (s *GoHttpServer) getHealthHandler() http.HandlerFunc {
	handlerName := "getHealthHandler"
	s.logger.Printf(initCallMsg, handlerName)
	return func(w http.ResponseWriter, r *http.Request) {
		s.logger.Printf(formatTraceRequest, handlerName, r.Method, r.URL.Path, r.RemoteAddr)
		w.WriteHeader(http.StatusOK)
	}
}
func (s *GoHttpServer) getMyDefaultHandler() http.HandlerFunc {
	handlerName := "getMyDefaultHandler"

	s.logger.Printf(initCallMsg, handlerName)

	data := collectRuntimeInfo(s.logger)

	return func(w http.ResponseWriter, r *http.Request) {
		remoteIp := r.RemoteAddr // ip address of the original request or the last proxy
		requestedUrlPath := r.URL.Path
		guid := xid.New()
		s.logger.Printf("INFO: 'Request ID: %s'\n", guid.String())
		s.logger.Printf(formatTraceRequest, handlerName, r.Method, requestedUrlPath, remoteIp)
		query := r.URL.Query()
		nameValue := query.Get("name")
		if nameValue != "" {
			data.ParamName = nameValue
		}
		data.Hostname, _ = os.Hostname()
		data.RemoteAddr = remoteIp
		data.Headers = r.Header
		data.Uptime = fmt.Sprintf("%s", time.Since(s.startTime))
		uptimeOS, err := GetOsUptime()
		if err != nil {
			s.logger.Printf("💥💥 ERROR: 'GetOsUptime() returned an error : %+#v'", err)
		}
		data.UptimeOs = uptimeOS
		data.RequestId = guid.String()
		rootPathGetCounter.Inc()
		err = s.jsonResponse(w, data)
		if err != nil {
			s.logger.Printf("ERROR:  %v doing jsonResponse [%s] path:'%s', from IP: [%s]\n", err, handlerName, requestedUrlPath, remoteIp)
			return
		}
		s.logger.Printf("SUCCESS: [%s] path:'%s', from IP: [%s]\n", handlerName, requestedUrlPath, remoteIp)

	}
}
func (s *GoHttpServer) getHandlerNotFound() http.HandlerFunc {
	handlerName := "getHandlerNotFound"
	s.logger.Printf(initCallMsg, handlerName)
	return func(w http.ResponseWriter, r *http.Request) {
		s.logger.Printf(formatErrRequest, handlerName, r.Method, r.URL.Path, r.RemoteAddr)
		w.Header().Set(HeaderContentType, MIMEAppJSONCharsetUTF8)
		w.WriteHeader(http.StatusNotFound)
		rootPathNotFoundCounter.Inc()
		type NotFound struct {
			Status  int    `json:"status"`
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		data := &NotFound{
			Status:  http.StatusNotFound,
			Error:   defaultNotFound,
			Message: defaultNotFoundDescription,
		}
		err := json.NewEncoder(w).Encode(data)
		if err != nil {
			s.logger.Printf("💥💥 ERROR: [%s] Not Found was unable to Fprintf. path:'%s', from IP: [%s]\n", handlerName, r.URL.Path, r.RemoteAddr)
			http.Error(w, "Internal server error. myDefaultHandler was unable to Fprintf", http.StatusInternalServerError)
		}
	}
}
func (s *GoHttpServer) getHandlerStaticPage(title string, descr string) http.HandlerFunc {
	handlerName := fmt.Sprintf("getHandlerStaticPage[%s]", title)
	s.logger.Printf(initCallMsg, handlerName)
	return func(w http.ResponseWriter, r *http.Request) {
		s.logger.Printf(formatErrRequest, handlerName, r.Method, r.URL.Path, r.RemoteAddr)
		w.Header().Set(HeaderContentType, MIMEHtml)
		w.WriteHeader(http.StatusOK)
		n, err := fmt.Fprintf(w, getHtmlPage(title, descr))
		if err != nil {
			s.logger.Printf("💥💥 ERROR: [%s]  was unable to Fprintf. path:'%s', from IP: [%s], send_bytes:%d\n", handlerName, r.URL.Path, r.RemoteAddr, n)
			http.Error(w, "Internal server error. getHandlerStaticPage was unable to Fprintf", http.StatusInternalServerError)
		}
	}
}
func (s *GoHttpServer) getTimeHandler() http.HandlerFunc {
	handlerName := "getTimeHandler"
	s.logger.Printf(initCallMsg, handlerName)
	return func(w http.ResponseWriter, r *http.Request) {
		s.logger.Printf(formatTraceRequest, handlerName, r.Method, r.URL.Path, r.RemoteAddr)
		now := time.Now()
		w.Header().Set(HeaderContentType, MIMEAppJSONCharsetUTF8)
		w.WriteHeader(http.StatusOK)
		_, err := fmt.Fprintf(w, "{\"time\":\"%s\"}", now.Format(time.RFC3339))
		if err != nil {
			s.logger.Printf("Error doing fmt.Fprintf err: %s", err)
			return
		}
	}
}
func (s *GoHttpServer) getWaitHandler(secondsToSleep int) http.HandlerFunc {
	handlerName := "getWaitHandler"
	s.logger.Printf(initCallMsg, handlerName)
	durationOfSleep := time.Duration(secondsToSleep) * time.Second
	return func(w http.ResponseWriter, r *http.Request) {
		s.logger.Printf(formatTraceRequest, handlerName, r.Method, r.URL.Path, r.RemoteAddr)
		if r.Method == http.MethodGet {
			w.Header().Set(HeaderContentType, MIMEAppJSONCharsetUTF8)
			time.Sleep(durationOfSleep) // simulate a delay to be ready
			w.WriteHeader(http.StatusOK)
			_, err := fmt.Fprintf(w, "{\"waited\":\"%v seconds\"}", secondsToSleep)
			if err != nil {
				s.logger.Printf("Error doing fmt.Fprintf err: %s", err)
				return
			}
		} else {
			s.logger.Printf(formatErrRequest, handlerName, r.Method, r.URL.Path, r.RemoteAddr)
			http.Error(w, httpErrMethodNotAllow, http.StatusMethodNotAllowed)
		}
	}
}

// ############# END HANDLERS
func main() {
	listenAddr, err := GetPortFromEnv(defaultPort)
	if err != nil {
		log.Fatalf("💥💥 ERROR: 'calling GetPortFromEnv got error: %v'\n", err)
	}
	listenAddr = defaultServerIp + listenAddr
	l := log.New(os.Stdout, fmt.Sprintf("HTTP_SERVER_%s ", APP), log.Ldate|log.Ltime|log.Lshortfile)
	l.Printf("INFO: '🚀🚀 App %s version:%s  from %s'", APP, VERSION, AppGithubUrl)
	l.Printf("INFO: 'Starting %s version:%s HTTP server on port %s'", APP, VERSION, listenAddr)
	server := NewGoHttpServer(listenAddr, l)
	// curl -vv  -X POST -H 'Content-Type: application/json'  http://localhost:8080/time   ==> 405 Method Not Allowed,
	// curl -vv  -X GET  -H 'Content-Type: application/json'  http://localhost:8080/time	==>200 OK , {"time":"2024-07-15T15:30:21+02:00"}
	server.AddRoute("GET /hello", server.getHandlerStaticPage("Hello", "Hello World!"))
	server.StartServer()
}
