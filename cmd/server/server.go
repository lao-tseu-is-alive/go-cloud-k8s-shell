package main

import (
	"embed"
	"fmt"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/config"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/gohttp"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/golog"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/info"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-shell/pkg/shell"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-shell/pkg/version"
	"github.com/rs/xid"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	defaultPort       = 9999
	defaultServerIp   = "0.0.0.0"
	defaultServerPath = "/"
	defaultWebRootDir = "front/dist/"
)

var (
	allowedHosts = []string{"localhost"}
	command      = "/bin/bash"
	args         []string
)

// content holds our static web server content.
//
//go:embed all:front/dist
var content embed.FS

func GetMyDefaultHandler(s *gohttp.Server, webRootDir string, content embed.FS) http.HandlerFunc {
	handlerName := "GetMyDefaultHandler"
	logger := s.GetLog()
	logger.Debug("Initial call to %s with webRootDir:%s", handlerName, webRootDir)

	// Create a subfolder filesystem to serve only the content of front/dist
	subFS, err := fs.Sub(content, "front/dist")
	if err != nil {
		logger.Fatal("Error creating sub-filesystem: %v", err)
	}

	// Create a file server handler for the embed filesystem
	handler := http.FileServer(http.FS(subFS))

	return func(w http.ResponseWriter, r *http.Request) {
		gohttp.TraceRequest(handlerName, r, logger)
		gohttp.RootPathGetCounter.Inc()
		handler.ServeHTTP(w, r)
	}
}

func GetInfoHandler(s *gohttp.Server) http.HandlerFunc {
	handlerName := "GetInfoHandler"
	logger := s.GetLog()
	logger.Debug("Initial call to %s", handlerName)

	data := info.CollectRuntimeInfo(version.APP, version.VERSION, logger)

	return func(w http.ResponseWriter, r *http.Request) {
		gohttp.TraceRequest(handlerName, r, logger)
		query := r.URL.Query()
		nameValue := query.Get("name")
		if nameValue != "" {
			data.ParamName = nameValue
		}
		data.Hostname, _ = os.Hostname()
		data.RemoteAddr = r.RemoteAddr
		data.Headers = r.Header
		data.Uptime = fmt.Sprintf("%s", time.Since(s.GetStartTime()))
		uptimeOS, err := info.GetOsUptime()
		if err != nil {
			logger.Error("GetOsUptime() returned an error : %+#v", err)
		}
		data.UptimeOs = uptimeOS
		guid := xid.New()
		data.RequestId = guid.String()
		err = s.JsonResponse(w, data)
		if err != nil {
			logger.Error("ERROR:  %v doing JsonResponse in %s, from IP: [%s]\n", err, handlerName, r.RemoteAddr)
			return
		}
		logger.Info("SUCCESS: [%s] from IP: [%s]\n", handlerName, r.RemoteAddr)
	}
}

func main() {
	listenAddr, err := config.GetPortFromEnv(defaultPort)
	if err != nil {
		log.Fatalf("💥💥 ERROR: 'calling GetPortFromEnv got error: %v'\n", err)
	}
	listenAddr = defaultServerIp + listenAddr
	l, err := golog.NewLogger("zap", golog.DebugLevel, fmt.Sprintf("%s ", version.APP))
	if err != nil {
		log.Fatalf("💥💥 error log.NewLogger error: %v'\n", err)
	}
	l.Info("🚀🚀 Starting %s v:%s from %s", version.APP, version.VERSION, version.REPOSITORY)
	l.Info("HTTP server listening %s'", listenAddr)
	server := gohttp.NewGoHttpServer(listenAddr, l)
	// curl -vv  -X GET  -H 'Content-Type: application/json'  http://localhost:9999/time	==>200 OK , {"time":"2024-07-15T15:30:21+02:00"}
	// using new server Mux in Go 1.22 https://pkg.go.dev/net/http#ServeMux
	mux := server.GetRouter()
	mux.Handle("GET /info", GetInfoHandler(server))
	mux.Handle("GET /goshell", shell.GetShellHandler(shell.HandlerOpts{
		AllowedHostnames:     allowedHosts,
		Arguments:            args,
		Command:              command,
		ConnectionErrorLimit: 10,
		Logger:               l,
		KeepalivePingTimeout: time.Second * 200,
		MaxBufferSizeBytes:   512,
	}))
	mux.Handle("GET /*", gohttp.NewMiddleware(
		server.GetRegistry(), nil).
		WrapHandler("GET /*", GetMyDefaultHandler(server, defaultWebRootDir, content)),
	)
	server.StartServer()
}
