package shell

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/creack/pty"
	"github.com/gorilla/websocket"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/gohttp"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common/pkg/golog"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const DefaultConnectionErrorLimit = 10

type HandlerOpts struct {
	// AllowedHostnames is a list of strings which will be matched to the client
	// requesting for a connection upgrade to a websocket connection
	AllowedHostnames []string
	// Arguments is a list of strings to pass as arguments to the specified COmmand
	Arguments []string
	// Command is the path to the binary we should create a TTY for
	Command string
	// ConnectionErrorLimit defines the number of consecutive errors that can happen
	// before a connection is considered unusable
	ConnectionErrorLimit int
	Logger               golog.MyLogger
	// KeepalivePingTimeout defines the maximum duration between which a ping and pong
	// cycle should be tolerated, beyond this the connection should be deemed dead
	KeepalivePingTimeout time.Duration
	MaxBufferSizeBytes   int
	JwtCheck             gohttp.JwtChecker
}

func GetShellHandler(opts HandlerOpts) http.HandlerFunc {
	handlerName := "GetShellHandler"
	opts.Logger.Info("INITIAL CALL TO %s", handlerName)
	return func(w http.ResponseWriter, r *http.Request) {
		clog := opts.Logger
		gohttp.TraceRequest(handlerName, r, clog)
		connectionErrorLimit := opts.ConnectionErrorLimit
		if connectionErrorLimit < 0 {
			connectionErrorLimit = DefaultConnectionErrorLimit
		}
		maxBufferSizeBytes := opts.MaxBufferSizeBytes
		keepalivePingTimeout := opts.KeepalivePingTimeout
		if keepalivePingTimeout <= time.Second {
			keepalivePingTimeout = 20 * time.Second
		}
		clog.Info("established connection identity")

		allowedHostnames := opts.AllowedHostnames
		upgrade := getConnectionUpgrade(allowedHostnames, maxBufferSizeBytes, clog)
		connection, err := upgrade.Upgrade(w, r, nil)
		if err != nil {
			clog.Warn("failed to upgrade connection: %s", err)
			return
		}

		terminal := opts.Command
		args := opts.Arguments
		clog.Debug("starting new tty using command '%s' with arguments ['%s']...", terminal, strings.Join(args, "', '"))
		cmd := exec.Command(terminal, args...)
		cmd.Env = os.Environ()
		clog.Info("executing command '%s' with arguments ['%s']...", terminal, strings.Join(args, "', '"))
		tty, err := pty.Start(cmd)
		if err != nil {
			message := fmt.Sprintf("failed to start tty: %s", err)
			clog.Warn(message)
			err := connection.WriteMessage(websocket.TextMessage, []byte(message))
			if err != nil {
				clog.Warn("failed to send error message to xterm.js: %s", err)
				return
			}
			return
		}
		clog.Info("pty.Start done....")
		token := r.URL.Query().Get("token")
		clog.Info("received token: '%s'", token)
		claims, err := opts.JwtCheck.ParseToken(token)
		if err != nil {
			clog.Warn("failed to parse token: %s", err)
			err := connection.WriteMessage(websocket.TextMessage, []byte("failed to parse JWT token"))
			if err != nil {
				clog.Warn("failed to send error message to xterm.js: %s", err)
				return
			}
			return
		}
		clog.Info("OK parsed JWT token: %+v", claims)
		defer func() {
			clog.Info("gracefully stopping spawned tty...")
			if err := cmd.Process.Kill(); err != nil {
				clog.Warn("failed to kill process: %s", err)
			}
			if _, err := cmd.Process.Wait(); err != nil {
				clog.Warn("failed to wait for process to exit: %s", err)
			}
			if err := tty.Close(); err != nil {
				clog.Warn("failed to close spawned tty gracefully: %s", err)
			}
			if err := connection.Close(); err != nil {
				clog.Warn("failed to close webscoket connection: %s", err)
			}
		}()

		var connectionClosed bool
		var waiter sync.WaitGroup
		waiter.Add(1)
		clog.Info("establishing Pong handler for keep-alive loop...")
		// this is a keep-alive loop that ensures connection does not hang-up itself
		lastPongTime := time.Now()
		connection.SetPongHandler(func(msg string) error {
			lastPongTime = time.Now()
			return nil
		})
		go func() {
			for {
				if connectionClosed {
					clog.Info("connection is closed, exiting from tty >> xterm.js...")
					return
				}
				if err := connection.WriteMessage(websocket.PingMessage, []byte("keepalive")); err != nil {
					clog.Warn("failed to write ping message")
					return
				}
				time.Sleep(keepalivePingTimeout / 2)
				if time.Now().Sub(lastPongTime) > keepalivePingTimeout {
					clog.Warn("failed to get response from ping, triggering disconnect now...")
					waiter.Done()
					return
				}
				clog.Debug("received response from ping successfully")
				if !claims.IsValidAt(time.Now()) {
					clog.Warn("token has expired, triggering disconnect now...")
					waiter.Done()
					return
				} else {
					clog.Debug("token is still valid at %v, until %v", time.Now(), claims.ExpiresAt)
				}
			}
		}()

		// tty >> xterm.js
		go func() {
			errorCounter := 0
			for {
				if connectionClosed {
					clog.Info("connection is closed, exiting from tty >> xterm.js...")
					return
				}
				if errorCounter > connectionErrorLimit {
					waiter.Done()
					break
				}
				buffer := make([]byte, maxBufferSizeBytes)
				readLength, err := tty.Read(buffer)
				if err != nil {
					clog.Warn(" 🚩 failed to read from tty: %s", err)
					if err := connection.WriteMessage(websocket.TextMessage, []byte("bye!")); err != nil {
						clog.Warn("failed to send termination message from tty to xterm.js: %s", err)
					}
					waiter.Done()
					return
				}
				if err := connection.WriteMessage(websocket.BinaryMessage, buffer[:readLength]); err != nil {
					clog.Warn("failed to send %v bytes from tty to xterm.js", readLength)
					errorCounter++
					continue
				}
				clog.Debug("sent message of size %v bytes from tty to xterm.js", readLength)
				errorCounter = 0
			}
		}()

		// tty << xterm.js
		go func() {
			for {
				if connectionClosed {
					clog.Info("connection is closed, exiting from tty << xterm.js...")
					return
				}
				// data processing
				messageType, data, err := connection.ReadMessage()
				if err != nil {
					if !connectionClosed {
						clog.Warn("failed to get next reader: %s", err)
					}
					return
				}
				dataLength := len(data)
				dataBuffer := bytes.Trim(data, "\x00")
				dataType, ok := WebsocketMessageType[messageType]
				if !ok {
					dataType = "unknown"
				}
				clog.Info("received %s (type: %v) message of size %v byte(s) from xterm.js with key sequence: %v", dataType, messageType, dataLength, dataBuffer)

				// process
				if dataLength == -1 { // invalid
					clog.Warn("failed to get the correct number of bytes read, ignoring message")
					continue
				}

				// handle resizing
				if messageType == websocket.BinaryMessage {
					if dataBuffer[0] == 1 {
						ttySize := &TTYSize{}
						resizeMessage := bytes.Trim(dataBuffer[1:], " \n\r\t\x00\x01")
						if err := json.Unmarshal(resizeMessage, ttySize); err != nil {
							clog.Warn("failed to unmarshal received resize message '%s': %s", string(resizeMessage), err)
							continue
						}
						clog.Info("resizing tty to use %v rows and %v columns...", ttySize.Rows, ttySize.Cols)
						if err := pty.Setsize(tty, &pty.Winsize{
							Rows: ttySize.Rows,
							Cols: ttySize.Cols,
						}); err != nil {
							clog.Warn("failed to resize tty, error: %s", err)
						}
						continue
					}
				}

				// write to tty
				bytesWritten, err := tty.Write(dataBuffer)
				if err != nil {
					clog.Warn(fmt.Sprintf("failed to write %v bytes to tty: %s", len(dataBuffer), err))
					continue
				}
				clog.Debug("%v bytes written to tty...", bytesWritten)
			}
		}()

		waiter.Wait()
		clog.Info("closing connection...")
		connectionClosed = true
	}
}
