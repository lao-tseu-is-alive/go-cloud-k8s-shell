package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"time"

	"github.com/jub0bs/cors"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/config"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/gohttp"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/golog"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-shell/pkg/shell"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-shell/pkg/version"
)

const (
	defaultPort       = 9999
	defaultServerIp   = "0.0.0.0"
	defaultLogName    = "stderr"
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
	l, err := golog.NewLogger(
		"simple",
		config.GetLogWriterFromEnvOrPanic(defaultLogName),
		config.GetLogLevelFromEnvOrPanic(golog.InfoLevel),
		version.APP,
	)
	if err != nil {
		log.Fatalf("💥💥 error log.NewLogger error: %v'\n", err)
	}
	l.Info("🚀🚀 Starting %s v:%s, build:%s from %s", version.APP, version.VERSION, version.BuildStamp, version.REPOSITORY)
	// Get the ENV JWT_AUTH_URL value
	jwtAuthUrl := config.GetJwtAuthUrlFromEnvOrPanic()
	jwtContextKey := config.GetJwtContextKeyFromEnvOrPanic()
	myVersionReader := gohttp.NewSimpleVersionReader(version.APP, version.VERSION, version.REPOSITORY, version.REVISION, version.BuildStamp, jwtAuthUrl)
	// Create a new JWT checker
	myJwt := gohttp.NewJwtChecker(
		config.GetJwtSecretFromEnvOrPanic(),
		config.GetJwtIssuerFromEnvOrPanic(),
		version.APP,
		jwtContextKey,
		config.GetJwtDurationFromEnvOrPanic(60),
		l)
	// Create a new Authenticator with a simple admin user
	myAuthenticator := gohttp.NewSimpleAdminAuthenticator(
		config.GetAdminUserFromEnvOrPanic(defaultAdminUser),
		config.GetAdminPasswordFromEnvOrPanic(),
		config.GetAdminEmailFromEnvOrPanic(defaultAdminEmail),
		config.GetAdminIdFromEnvOrPanic(defaultAdminId),
		myJwt)

	server := gohttp.CreateNewServerFromEnvOrFail(
		defaultPort,
		defaultServerIp, version.APP, l,
		gohttp.WithAuthentication(myAuthenticator),
		gohttp.WithJwtChecker(myJwt),
		gohttp.WithVersionReader(myVersionReader),
	)

	allowedHosts := config.GetAllowedHostsFromEnvOrPanic()
	server.AddRoute("GET /info", gohttp.GetInfoHandler(server))

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

	mux.Handle("GET /goAppInfo", corsMw.Wrap(gohttp.GetAppInfoHandler(server)))
	mux.Handle("GET /health", server.GetHealthHandler(
		func(msg string) bool {
			return true
		},
		fmt.Sprintf("%s v%s", version.APP, version.VERSION)))

	mux.Handle("GET /readiness", server.GetHealthHandler(
		func(msg string) bool {
			return true
		},
		fmt.Sprintf("%s v%s", version.APP, version.VERSION)))

	mux.Handle(fmt.Sprintf("POST %s", jwtAuthUrl), corsMw.Wrap(gohttp.GetLoginPostHandler(server)))
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
