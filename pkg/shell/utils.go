package shell

import (
	"github.com/gorilla/websocket"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/golog"
	"net/http"
	"slices"
	"strings"
)

func getConnectionUpgrade(
	allowedHostnames []string,
	maxBufferSizeBytes int,
	logger golog.MyLogger,
) websocket.Upgrader {
	return websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			if slices.Contains(allowedHostnames, "*") {
				return true
			}
			requesterHostname := r.Host
			if strings.Index(requesterHostname, ":") != -1 {
				requesterHostname = strings.Split(requesterHostname, ":")[0]
			}
			if slices.Contains(allowedHostnames, "localhost") {
				if requesterHostname == "127.0.0.1" || requesterHostname == "::1" {
					return true
				}
			}
			for _, allowedHostname := range allowedHostnames {
				if requesterHostname == allowedHostname {
					return true
				}
			}
			logger.Warn("failed to find '%s' in the list of allowed hostnames ('%s')", requesterHostname)
			return false
		},
		HandshakeTimeout: 0,
		ReadBufferSize:   maxBufferSizeBytes,
		WriteBufferSize:  maxBufferSizeBytes,
	}
}
