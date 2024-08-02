package main

import (
	"embed"
	"fmt"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/config"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/gohttp"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/golog"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-shell/pkg/shell"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-shell/pkg/version"
	"io/fs"
	"log"
	"net/http"
	"time"
)

const (
	defaultPort       = 9999
	defaultServerIp   = "0.0.0.0"
	defaultServerPath = "/"
	defaultWebRootDir = "front/dist/"
	defaultAdminId    = 99999
	defaultAdminUser  = "goadmin"
	defaultAdminEmail = "goadmin@lausanne.ch"
)

var (
	defaultAllowedHosts = []string{"127.0.0.1"}
	command             = "/bin/bash"
	args                []string
)

// content holds our static web server content.
//
//go:embed all:front/dist
var content embed.FS

func GetMyDefaultHandler(s *gohttp.Server, webRootDir string, content embed.FS) http.HandlerFunc {
	handlerName := "GetMyDefaultHandler"
	logger := s.GetLog()
	logger.Debug("Initial call to %s with webRootDir:%s", handlerName, webRootDir)
	RootPathGetCounter := s.RootPathGetCounter

	// Create a subfolder filesystem to serve only the content of front/dist
	subFS, err := fs.Sub(content, "front/dist")
	if err != nil {
		logger.Fatal("Error creating sub-filesystem: %v", err)
	}

	// Create a file server handler for the embed filesystem
	handler := http.FileServer(http.FS(subFS))

	return func(w http.ResponseWriter, r *http.Request) {
		gohttp.TraceRequest(handlerName, r, logger)
		RootPathGetCounter.Inc()
		handler.ServeHTTP(w, r)
	}
}

func main() {
	l, err := golog.NewLogger("zap", golog.DebugLevel, fmt.Sprintf("%s ", version.APP))
	if err != nil {
		log.Fatalf("💥💥 error log.NewLogger error: %v'\n", err)
	}
	l.Info("🚀🚀 Starting %s v:%s from %s", version.APP, version.VERSION, version.REPOSITORY)

	myVersionReader := gohttp.NewSimpleVersionReader(version.APP, version.VERSION, version.REVISION)
	server := gohttp.CreateNewServerFromEnvOrFail(
		defaultPort,
		defaultServerIp,
		defaultAdminUser,
		defaultAdminEmail,
		defaultAdminId,
		myVersionReader,
		l)

	allowedHosts := config.GetAllowedIpsFromEnvOrPanic(defaultAllowedHosts)
	mux := server.GetRouter()
	myJwt := server.JwtCheck
	mux.Handle("POST /login", gohttp.GetLoginPostHandler(server))

	mux.Handle("GET /goshell", myJwt.JwtMiddleware(shell.GetShellHandler(shell.HandlerOpts{
		AllowedHostnames:     allowedHosts,
		Arguments:            args,
		Command:              command,
		ConnectionErrorLimit: 10,
		Logger:               l,
		KeepalivePingTimeout: time.Second * 200,
		MaxBufferSizeBytes:   512,
	})))
	mux.Handle("GET /*", gohttp.NewPrometheusMiddleware(
		server.GetPrometheusRegistry(), nil).
		WrapHandler("GET /*", GetMyDefaultHandler(server, defaultWebRootDir, content)),
	)
	server.StartServer()
}
