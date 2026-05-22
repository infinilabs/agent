/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	"bytes"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/cihub/seelog"
	"github.com/gorilla/websocket"
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/util"
	configcommon "infini.sh/framework/modules/configs/common"
)

const (
	reverseChannelHeaderInstanceID = "X-INFINI-INSTANCE-ID"
	reverseHelloCommand            = "reverse_hello"
	reverseRequestCommand          = "reverse_request"
	reverseResponseCommand         = "reverse_response"
	reverseChannelTag              = "agent_reverse_channel"
	reverseReconnectDelay          = 5 * time.Second
	reverseMaxIncomingMessageBytes = 8 * 1024 * 1024
	reverseResponseChunkBytes      = 32 * 1024
)

type agentReverseHelloMessage struct {
	SessionID  string `json:"session_id"`
	InstanceID string `json:"instance_id"`
}

type agentReverseRequestMessage struct {
	RequestID   string `json:"request_id"`
	InstanceID  string `json:"instance_id"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	Body        string `json:"body,omitempty"`
	AccessToken string `json:"access_token,omitempty"`
}

type agentReverseResponseMessage struct {
	RequestID  string `json:"request_id"`
	InstanceID string `json:"instance_id"`
	Chunk      string `json:"chunk,omitempty"`
	Status     int    `json:"status,omitempty"`
	Done       bool   `json:"done,omitempty"`
}

var agentReverseChannelRunning atomic.Bool
var agentReverseChannelWriteLock sync.Mutex
var agentReverseAPIs = newReverseAPIRouter(AgentAPI{})

func registerAgentReverseChannel() {
	global.RegisterBackgroundCallback(&global.BackgroundTask{
		Tag:          reverseChannelTag,
		Interval:     reverseReconnectDelay,
		InitialDelay: reverseReconnectDelay,
		Func:         ensureAgentReverseChannel,
	})
}

func ensureAgentReverseChannel() {
	if global.ShuttingDown() || !global.Env().SystemConfig.Configs.Managed || len(getAgentReverseChannelServers()) == 0 {
		return
	}
	if !agentReverseChannelRunning.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer agentReverseChannelRunning.Store(false)
		runAgentReverseChannel()
	}()
}

func runAgentReverseChannel() {
	for !global.ShuttingDown() {
		err := connectAndServeAgentReverseChannel()
		if err != nil && !global.ShuttingDown() {
			log.Warnf("agent reverse channel disconnected: %v", err)
		}
		if global.ShuttingDown() {
			return
		}
		time.Sleep(reverseReconnectDelay)
	}
}

func connectAndServeAgentReverseChannel() error {
	var lastErr error
	for _, server := range getAgentReverseChannelServers() {
		conn, err := dialAgentReverseChannel(server)
		if err != nil {
			lastErr = err
			continue
		}
		defer conn.Close()
		conn.SetReadLimit(reverseMaxIncomingMessageBytes)

		for {
			messageType, payload, err := conn.ReadMessage()
			if err != nil {
				return err
			}
			if messageType != websocket.TextMessage {
				continue
			}
			handleAgentReverseChannelMessage(conn, payload)
		}
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("no console server available for agent reverse channel")
}

func getAgentReverseChannelServers() []string {
	agentCfg := configcommon.GetAgentConfig()
	if agentCfg != nil && agentCfg.Setup != nil {
		if endpoint := strings.TrimSpace(agentCfg.Setup.ReverseChannelEndpoint); endpoint != "" {
			return []string{endpoint}
		}
	}

	servers := make([]string, 0, len(global.Env().SystemConfig.Configs.Servers))
	for _, server := range global.Env().SystemConfig.Configs.Servers {
		server = strings.TrimSpace(server)
		if server == "" {
			continue
		}
		servers = append(servers, server)
	}
	return servers
}

func dialAgentReverseChannel(server string) (*websocket.Conn, error) {
	wsURL, err := buildAgentReverseChannelURL(server)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Set(reverseChannelHeaderInstanceID, global.Env().SystemConfig.NodeConfig.ID)
	managerToken, err := configcommon.LoadTokenFromKeystore(configcommon.ManagerTokenKeystoreKey)
	if err != nil {
		return nil, err
	}
	managerAuth := global.Env().SystemConfig.Configs.ManagerConfig.BasicAuth
	if managerToken != "" {
		headers.Set("Authorization", "Bearer "+managerToken)
	} else if managerAuth.Username != "" {
		token := base64.StdEncoding.EncodeToString([]byte(managerAuth.Username + ":" + managerAuth.Password.Get()))
		headers.Set("Authorization", "Basic "+token)
	}

	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	if clientCfg := global.Env().GetHTTPClientConfig("configs", server); clientCfg != nil {
		tlsCfg, err := api.GetClientTLSConfig(&clientCfg.TLSConfig)
		if err != nil {
			return nil, err
		}
		dialer.TLSClientConfig = tlsCfg
	}
	conn, _, err := dialer.Dial(wsURL, headers)
	return conn, err
}

func buildAgentReverseChannelURL(server string) (string, error) {
	parsed, err := url.Parse(server)
	if err != nil {
		return "", err
	}
	switch parsed.Scheme {
	case "https":
		parsed.Scheme = "wss"
	default:
		parsed.Scheme = "ws"
	}
	parsed.Path = path.Join(parsed.Path, "/ws")
	return parsed.String(), nil
}

func handleAgentReverseChannelMessage(conn *websocket.Conn, payload []byte) {
	text := string(payload)
	parts := strings.SplitN(text, " ", 2)
	if len(parts) != 2 {
		return
	}

	switch parts[0] {
	case "CONFIG":
		if strings.HasPrefix(parts[1], "websocket-session-id:") {
			sessionID := strings.TrimSpace(strings.TrimPrefix(parts[1], "websocket-session-id:"))
			if sessionID != "" {
				helloPayload := string(util.MustToJSONBytes(agentReverseHelloMessage{
					SessionID:  sessionID,
					InstanceID: global.Env().SystemConfig.NodeConfig.ID,
				}))
				_ = writeAgentReverseText(conn, reverseHelloCommand+" "+helloPayload)
			}
		}
	case "PRIVATE":
		prefix := reverseRequestCommand + " "
		if strings.HasPrefix(parts[1], prefix) {
			go handleAgentReverseRequest(conn, strings.TrimPrefix(parts[1], prefix))
		}
	}
}

func handleAgentReverseRequest(conn *websocket.Conn, payload string) {
	reqMsg := agentReverseRequestMessage{}
	if err := util.FromJSONBytes([]byte(payload), &reqMsg); err != nil {
		body := buildAgentReverseErrorBody(http.StatusBadRequest, err.Error())
		_ = writeAgentReverseResponse(conn, reqMsg.RequestID, global.Env().SystemConfig.NodeConfig.ID, http.StatusBadRequest, body)
		return
	}
	if !validateAgentAccessToken(reqMsg.AccessToken) {
		body := buildAgentReverseErrorBody(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
		_ = writeAgentReverseResponse(conn, reqMsg.RequestID, reqMsg.InstanceID, http.StatusUnauthorized, body)
		return
	}

	body := []byte(nil)
	if reqMsg.Body != "" {
		decoded, err := base64.StdEncoding.DecodeString(reqMsg.Body)
		if err != nil {
			errBody := buildAgentReverseErrorBody(http.StatusBadRequest, err.Error())
			_ = writeAgentReverseResponse(conn, reqMsg.RequestID, reqMsg.InstanceID, http.StatusBadRequest, errBody)
			return
		}
		body = decoded
	}

	status, responseBody := executeAgentReverseRequest(reqMsg.Method, reqMsg.Path, body)
	_ = writeAgentReverseResponse(conn, reqMsg.RequestID, reqMsg.InstanceID, status, responseBody)
}

func executeAgentReverseRequest(method, requestPath string, body []byte) (status int, responseBody []byte) {
	defer func() {
		if r := recover(); r != nil {
			status = http.StatusInternalServerError
			responseBody = buildAgentReverseErrorBody(status, fmt.Sprint(r))
		}
	}()

	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}
	req := httptest.NewRequest(method, "http://agent"+requestPath, bodyReader)
	req.Header.Set("Content-Type", util.ContentTypeJson)
	recorder := httptest.NewRecorder()
	if _, err := url.ParseRequestURI(requestPath); err != nil {
		return http.StatusBadRequest, buildAgentReverseErrorBody(http.StatusBadRequest, err.Error())
	}

	if !agentReverseAPIs.ServeHTTP(recorder, req) {
		recorder.WriteHeader(http.StatusNotFound)
		recorder.Write(buildAgentReverseErrorBody(http.StatusNotFound, "reverse channel path not found"))
	}

	result := recorder.Result()
	defer result.Body.Close()
	responseBody, _ = io.ReadAll(result.Body)
	status = result.StatusCode
	if status == 0 {
		status = http.StatusOK
	}
	if len(responseBody) == 0 && status >= http.StatusBadRequest {
		responseBody = buildAgentReverseErrorBody(status, http.StatusText(status))
	}
	return status, responseBody
}

func writeAgentReverseResponse(conn *websocket.Conn, requestID, instanceID string, status int, body []byte) error {
	for start := 0; start < len(body); start += reverseResponseChunkBytes {
		end := start + reverseResponseChunkBytes
		if end > len(body) {
			end = len(body)
		}
		msg := agentReverseResponseMessage{
			RequestID:  requestID,
			InstanceID: instanceID,
			Chunk:      base64.StdEncoding.EncodeToString(body[start:end]),
		}
		if err := writeAgentReverseText(conn, reverseResponseCommand+" "+string(util.MustToJSONBytes(msg))); err != nil {
			return err
		}
	}

	doneMsg := agentReverseResponseMessage{
		RequestID:  requestID,
		InstanceID: instanceID,
		Status:     status,
		Done:       true,
	}
	return writeAgentReverseText(conn, reverseResponseCommand+" "+string(util.MustToJSONBytes(doneMsg)))
}

func writeAgentReverseText(conn *websocket.Conn, payload string) error {
	agentReverseChannelWriteLock.Lock()
	defer agentReverseChannelWriteLock.Unlock()
	return conn.WriteMessage(websocket.TextMessage, []byte(payload))
}

func buildAgentReverseErrorBody(status int, reason string) []byte {
	return util.MustToJSONBytes(util.MapStr{
		"error": util.MapStr{
			"reason": reason,
		},
		"status": status,
	})
}

func validateAgentAccessToken(tokenValue string) bool {
	expected, err := configcommon.LoadTokenFromKeystore(configcommon.AgentAccessTokenKeystoreKey)
	if err != nil || expected == "" || tokenValue == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(tokenValue)) == 1
}
