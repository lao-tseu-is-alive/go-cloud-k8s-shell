package main

import (
	"embed"
	"fmt"
	"github.com/jub0bs/cors"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/config"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/gohttp"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/golog"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-shell/pkg/shell"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-shell/pkg/version"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"time"
)

const (
	defaultPort       = 9999
	defaultServerIp   = "0.0.0.0"
	defaultServerPath = "/"
	defaultWebRootDir = "front/dist"
	defaultAdminId    = 99999
	defaultAdminUser  = "goadmin"
	defaultAdminEmail = "goadmin@lausanne.ch"
)

var (
	defaultAllowedHosts = []string{"localhost"}
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
	subFS, err := fs.Sub(content, webRootDir)
	if err != nil {
		logger.Fatal("Error creating sub-filesystem: %v", err)
	}
	mime.AddExtensionType(".js", "application/javascript")
	mime.AddExtensionType(".css", "text/css")
	mime.AddExtensionType(".svg", "image/svg+xml")
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

	myVersionReader := gohttp.NewSimpleVersionReader(version.APP, version.VERSION, version.REVISION, version.Build)
	// Create a new JWT checker
	myJwt := gohttp.NewJwtChecker(
		config.GetJwtSecretFromEnvOrPanic(),
		config.GetJwtIssuerFromEnvOrPanic(),
		version.APP,
		config.GetJwtDurationFromEnvOrPanic(60),
		l)
	// Create a new Authenticator with a simple admin user
	myAuthenticator := gohttp.NewSimpleAdminAuthenticator(
		config.GetAdminUserFromFromEnvOrPanic(defaultAdminUser),
		config.GetAdminPasswordFromFromEnvOrPanic(),
		config.GetAdminEmailFromFromEnvOrPanic(defaultAdminEmail),
		config.GetAdminIdFromFromEnvOrPanic(defaultAdminId),
		myJwt)
	server := gohttp.CreateNewServerFromEnvOrFail(
		defaultPort,
		defaultServerIp,
		myAuthenticator,
		myJwt,
		myVersionReader,
		l)

	allowedHosts := config.GetAllowedHostsFromEnvOrPanic()
	mux := server.GetRouter()

	// create CORS middleware
	corsMw, err := cors.NewMiddleware(cors.Config{
		Origins:        []string{"http://localhost:5173"}, // for vite js dev server
		Methods:        []string{http.MethodGet, http.MethodPost},
		RequestHeaders: []string{"Authorization"},
	})
	if err != nil {
		log.Fatalf("💥💥 error cors.NewMiddleware error: %v'\n", err)
	}
	corsMw.SetDebug(true) // turn debug mode on (optional)
	mux.Handle("POST /login", corsMw.Wrap(gohttp.GetLoginPostHandler(server)))
	mux.Handle("GET /goshell", shell.GetShellHandler(shell.HandlerOpts{
		AllowedHostnames:     allowedHosts,
		Arguments:            args,
		Command:              command,
		ConnectionErrorLimit: 10,
		Logger:               l,
		KeepalivePingTimeout: time.Second * 60,
		MaxBufferSizeBytes:   512,
		JwtCheck:             myJwt,
	}))
	mux.Handle("GET /api", http.StripPrefix("/api", corsMw.Wrap(mux))) // note: method-less pattern here
	mux.Handle("GET /", gohttp.NewPrometheusMiddleware(
		server.GetPrometheusRegistry(), nil).
		WrapHandler("GET /", GetMyDefaultHandler(server, defaultWebRootDir, content)),
	)
	server.StartServer()
}
