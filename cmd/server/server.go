package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/config"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/go_http"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/info"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-shell/pkg/version"
	"github.com/rs/xid"
)

const (
	defaultPort       = 9999
	defaultServerIp   = ""
	defaultServerPath = "/"
	defaultWebRootDir = "front/dist/"
)

// content holds our static web server content.
//
//go:embed all:front/dist
var content embed.FS

func GetMyDefaultHandler(s *go_http.GoHttpServer, webRootDir string, content embed.FS) http.HandlerFunc {
	handlerName := "GetMyDefaultHandler"
	logger := s.GetLog()
	logger.Printf("Initial call to %s with webRootDir:%s", handlerName, webRootDir)

	// Create a subfolder filesystem to serve only the content of front/dist
	subFS, err := fs.Sub(content, "front/dist")
	if err != nil {
		logger.Fatalf("Error creating sub-filesystem: %v", err)
	}

	// Create a file server handler for the embed filesystem
	//handler := http.StripPrefix(webRootDir, http.FileServer(http.FS(subFS)))
	handler := http.FileServer(http.FS(subFS))

	return func(w http.ResponseWriter, r *http.Request) {
		go_http.TraceRequest(handlerName, r, logger)
		go_http.RootPathGetCounter.Inc()

		// Log the requested path
		logger.Printf("Requested path: %s", r.URL.Path)

		// Trim the leading slash if it exists
		//urlPath := strings.TrimPrefix(r.URL.Path, "/")
		// urlPath := r.URL.Path

		// Check if the file exists in the embedded filesystem
		/*
			_, err := content.Open(urlPath)
			if err != nil {
				logger.Printf("File not found: %s, error: %v", urlPath, err)
				http.NotFound(w, r)
				return
			}
		*/

		handler.ServeHTTP(w, r)
	}
}

func GetInfoHandler(s *go_http.GoHttpServer) http.HandlerFunc {
	handlerName := "GetInfoHandler"
	logger := s.GetLog()
	logger.Printf("Initial call to %s", handlerName)

	data := info.CollectRuntimeInfo(version.APP, version.VERSION, logger)

	return func(w http.ResponseWriter, r *http.Request) {
		go_http.TraceRequest(handlerName, r, logger)
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
			logger.Printf("💥💥 ERROR: 'GetOsUptime() returned an error : %+#v'", err)
		}
		data.UptimeOs = uptimeOS
		guid := xid.New()
		data.RequestId = guid.String()
		err = s.JsonResponse(w, data)
		if err != nil {
			logger.Printf("ERROR:  %v doing JsonResponse in %s, from IP: [%s]\n", err, handlerName, r.RemoteAddr)
			return
		}
		logger.Printf("SUCCESS: [%s] from IP: [%s]\n", handlerName, r.RemoteAddr)
	}
}

func main() {
	listenAddr, err := config.GetPortFromEnv(defaultPort)
	if err != nil {
		log.Fatalf("💥💥 ERROR: 'calling GetPortFromEnv got error: %v'\n", err)
	}
	listenAddr = defaultServerIp + listenAddr
	l := log.New(os.Stdout, fmt.Sprintf("HTTP_SERVER_%s ", version.APP), log.Ldate|log.Ltime|log.Lshortfile)
	l.Printf("INFO: '🚀🚀 App %s version:%s  from %s'", version.APP, version.VERSION, version.REPOSITORY)
	l.Printf("INFO: 'Starting %s version:%s HTTP server on port %s'", version.APP, version.VERSION, listenAddr)
	server := go_http.NewGoHttpServer(listenAddr, l)
	// curl -vv  -X GET  -H 'Content-Type: application/json'  http://localhost:9999/time	==>200 OK , {"time":"2024-07-15T15:30:21+02:00"}
	// using new server Mux in Go 1.22 https://pkg.go.dev/net/http#ServeMux
	mux := server.GetRouter()
	mux.Handle("GET /info", GetInfoHandler(server))
	mux.Handle("GET /*", go_http.NewMiddleware(
		server.GetRegistry(), nil).
		WrapHandler("GET /*", GetMyDefaultHandler(server, defaultWebRootDir, content)),
	)
	server.StartServer()
}
