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
	httprouter "infini.sh/framework/core/api/router"
	framework_reverse "infini.sh/framework/core/api/websocket/reverse"
	"infini.sh/framework/core/config"
	"infini.sh/framework/core/global"
	"infini.sh/framework/core/model"
	"infini.sh/framework/core/util"
	configcommon "infini.sh/framework/modules/configs/common"
)

const (
	agentReverseChannelTag              = "agent_reverse_channel"
	agentReverseReconnectDelay          = 5 * time.Second
	agentReverseMaxIncomingMessageBytes = 8 * 1024 * 1024
)

type agentReverseProxyTarget struct {
	endpoint      string
	basePath      string
	tlsConfig     config.TLSConfig
	basicAuthUser string
	basicAuthPass string
}

var agentReverseChannelRunning atomic.Bool
var agentReverseChannelWriteLock sync.Mutex
var agentReverseAPIPathMatcher = newAgentReverseAPIPathMatcher()
var agentReverseAPIProxyTargetResolver = resolveAgentReverseAPIProxyTarget
var agentReverseHTTPClientFactory = newAgentReverseHTTPClient
var agentReverseStateLock sync.Mutex
var agentReverseLastState string

func newAgentReverseAPIPathMatcher() *httprouter.Router {
	router := httprouter.New(nil)
	handle := func(http.ResponseWriter, *http.Request, httprouter.Params) {}
	registerProtectedAPIRoutes(router, handle)
	return router
}

func shouldServeRegisteredAPIReverse(method, rawPath string) bool {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(rawPath))
	if err != nil || parsed.Path == "" {
		return false
	}
	handle, _, _ := agentReverseAPIPathMatcher.Lookup(strings.ToUpper(method), parsed.Path)
	return handle != nil
}

func registerAgentReverseChannel() {
	global.RegisterBackgroundCallback(&global.BackgroundTask{
		Tag:          agentReverseChannelTag,
		Interval:     agentReverseReconnectDelay,
		InitialDelay: agentReverseReconnectDelay,
		Func:         ensureAgentReverseChannel,
	})
}

func ensureAgentReverseChannel() {
	if global.ShuttingDown() {
		logAgentReverseStatef("shutting-down", "skip agent reverse channel startup: application is shutting down")
		return
	}
	if !global.Env().SystemConfig.Configs.Managed {
		logAgentReverseStatef("configs-unmanaged", "skip agent reverse channel startup: configs.managed is disabled")
		return
	}
	servers := getAgentReverseChannelServers()
	if len(servers) == 0 {
		logAgentReverseStatef("no-endpoints", "skip agent reverse channel startup: no reverse channel endpoints configured")
		return
	}
	if !agentReverseChannelRunning.CompareAndSwap(false, true) {
		return
	}
	logAgentReverseStatef("starting", "starting agent reverse channel with endpoints: %s", strings.Join(servers, ", "))
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
		time.Sleep(agentReverseReconnectDelay)
	}
}

func connectAndServeAgentReverseChannel() error {
	var lastErr error
	for _, server := range getAgentReverseChannelServers() {
		log.Debugf("attempting agent reverse channel connection to [%s]", server)
		conn, err := dialAgentReverseChannel(server)
		if err != nil {
			log.Debugf("failed to dial agent reverse channel endpoint [%s]: %v", server, err)
			lastErr = err
			continue
		}
		logAgentReverseStatef("connected:"+server, "agent reverse channel connected to [%s]", server)
		defer conn.Close()
		conn.SetReadLimit(agentReverseMaxIncomingMessageBytes)

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
		servers := make([]string, 0, len(agentCfg.Setup.ReverseChannelEndpoints))
		for _, endpoint := range agentCfg.Setup.ReverseChannelEndpoints {
			endpoint = strings.TrimSpace(endpoint)
			if endpoint == "" {
				continue
			}
			servers = append(servers, endpoint)
		}
		if len(servers) > 0 {
			return servers
		}
	}
	return nil
}

func dialAgentReverseChannel(server string) (*websocket.Conn, error) {
	wsURL, err := buildAgentReverseChannelURL(server)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Set(framework_reverse.HeaderPeerID, global.Env().SystemConfig.NodeConfig.ID)
	managerToken, err := configcommon.LoadTokenFromKeystore(configcommon.ManagerTokenKeystoreKey)
	if err != nil {
		return nil, err
	}
	managerAccessToken := strings.TrimSpace(global.Env().SystemConfig.Configs.ManagerConfig.AccessToken.Get())
	managerAuth := global.Env().SystemConfig.Configs.ManagerConfig.BasicAuth
	if managerAccessToken != "" {
		headers.Set(model.API_TOKEN, managerAccessToken)
	} else if managerToken != "" {
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
				log.Debugf("agent reverse channel received websocket session id [%s]", sessionID)
				err := writeAgentReverseText(conn, framework_reverse.FormatHelloCommand(framework_reverse.HelloMessage{
					SessionID: sessionID,
					PeerID:    global.Env().SystemConfig.NodeConfig.ID,
				}))
				if err != nil {
					log.Warnf("failed to send agent reverse hello for session [%s]: %v", sessionID, err)
					return
				}
				log.Infof("agent reverse channel handshake sent for session [%s]", sessionID)
			}
		}
	case "PRIVATE":
		prefix := framework_reverse.RequestCommand + " "
		if strings.HasPrefix(parts[1], prefix) {
			go handleAgentReverseRequest(conn, strings.TrimPrefix(parts[1], prefix))
		}
	}
}

func handleAgentReverseRequest(conn *websocket.Conn, payload string) {
	reqMsg, err := framework_reverse.ParseRequestPayload(payload)
	if err != nil {
		body := buildAgentReverseErrorBody(http.StatusBadRequest, err.Error())
		_ = writeAgentReverseResponse(conn, "", global.Env().SystemConfig.NodeConfig.ID, http.StatusBadRequest, body)
		return
	}
	if !validateAgentAccessToken(reqMsg.BearerToken()) {
		body := buildAgentReverseErrorBody(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
		_ = writeAgentReverseResponse(conn, reqMsg.RequestID, reqMsg.PeerID, http.StatusUnauthorized, body)
		return
	}

	body, err := reqMsg.BodyBytes()
	if err != nil {
		errBody := buildAgentReverseErrorBody(http.StatusBadRequest, err.Error())
		_ = writeAgentReverseResponse(conn, reqMsg.RequestID, reqMsg.PeerID, http.StatusBadRequest, errBody)
		return
	}

	status, responseBody := executeAgentReverseRequest(reqMsg.Method, reqMsg.Path, body, reqMsg)
	_ = writeAgentReverseResponse(conn, reqMsg.RequestID, reqMsg.PeerID, status, responseBody)
}

func executeAgentReverseRequest(method, requestPath string, body []byte, reqMsg framework_reverse.RequestMessage) (status int, responseBody []byte) {
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
	reqMsg.ApplyHeaders(req)
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", util.ContentTypeJson)
	}
	recorder := httptest.NewRecorder()
	handler := AgentAPI{}
	parsedPath, err := url.ParseRequestURI(requestPath)
	if err != nil {
		return http.StatusBadRequest, buildAgentReverseErrorBody(http.StatusBadRequest, err.Error())
	}

	switch parsedPath.Path {
	case "/agent/_info":
		handler.getAgentInfo(recorder, req, nil)
	case "/elasticsearch/node/_discovery":
		handler.getESNodes(recorder, req, nil)
	case "/elasticsearch/node/_info":
		handler.getESNodeInfo(recorder, req, nil)
	case "/elasticsearch/logs/_list":
		handler.getElasticLogFiles(recorder, req, nil)
	case "/elasticsearch/logs/_read":
		handler.readElasticLogFile(recorder, req, nil)
	default:
		if shouldServeRegisteredAPIReverse(method, requestPath) {
			return executeAgentRegisteredAPIReverse(method, requestPath, body, reqMsg)
		}
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

func executeAgentRegisteredAPIReverse(method, requestPath string, body []byte, reqMsg framework_reverse.RequestMessage) (status int, responseBody []byte) {
	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, "http://agent"+requestPath, bodyReader)
	if err != nil {
		return http.StatusBadRequest, buildAgentReverseErrorBody(http.StatusBadRequest, err.Error())
	}
	reqMsg.ApplyHeaders(req)
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", util.ContentTypeJson)
	}
	applyAgentReverseLocalAPIAuth(req)
	recorder := httptest.NewRecorder()
	api.ServeRegisteredAPIRequest(recorder, req)
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

func applyAgentReverseLocalAPIAuth(req *http.Request) {
	if req == nil {
		return
	}
	apiCfg := global.Env().SystemConfig.APIConfig
	if !apiCfg.Security.Enabled {
		return
	}
	username := strings.TrimSpace(apiCfg.Security.Username)
	if username == "" {
		return
	}
	req.SetBasicAuth(username, apiCfg.Security.Password)
}

func resolveAgentReverseAPIProxyTarget() (agentReverseProxyTarget, error) {
	apiCfg := global.Env().SystemConfig.APIConfig
	if apiCfg.Enabled {
		return agentReverseProxyTarget{
			endpoint:      fmt.Sprintf("%s://%s", apiCfg.GetSchema(), apiCfg.NetworkConfig.GetPublishAddr()),
			basePath:      apiCfg.BasePath,
			tlsConfig:     apiCfg.TLSConfig,
			basicAuthUser: apiCfg.Security.Username,
			basicAuthPass: apiCfg.Security.Password,
		}, nil
	}

	webCfg := global.Env().SystemConfig.WebAppConfig
	if webCfg.Enabled && webCfg.EmbeddingAPI {
		return agentReverseProxyTarget{
			endpoint:  fmt.Sprintf("%s://%s", webCfg.GetSchema(), webCfg.NetworkConfig.GetPublishAddr()),
			basePath:  webCfg.BasePath,
			tlsConfig: webCfg.TLSConfig,
		}, nil
	}
	return agentReverseProxyTarget{}, fmt.Errorf("reverse channel api endpoint unavailable")
}

func buildAgentReverseProxyURL(endpoint, basePath, requestPath string) (string, error) {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(requestPath))
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(endpoint) == "" {
		return "", fmt.Errorf("reverse channel endpoint unavailable")
	}

	fullPath := parsed.Path
	basePath = strings.TrimSpace(basePath)
	if basePath != "" && basePath != "/" {
		if !strings.HasPrefix(basePath, "/") {
			basePath = "/" + basePath
		}
		fullPath = strings.TrimRight(basePath, "/") + fullPath
	}

	proxyURL := strings.TrimRight(strings.TrimSpace(endpoint), "/") + fullPath
	if parsed.RawQuery != "" {
		proxyURL += "?" + parsed.RawQuery
	}
	return proxyURL, nil
}

func newAgentReverseHTTPClient(target agentReverseProxyTarget) (*http.Client, error) {
	transport := &http.Transport{}
	if target.tlsConfig.TLSEnabled {
		tlsCfg := target.tlsConfig
		tlsCfg.SkipDomainVerify = true
		clientTLSConfig, err := api.GetClientTLSConfig(&tlsCfg)
		if err != nil {
			return nil, err
		}
		transport.TLSClientConfig = clientTLSConfig
	}

	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}, nil
}

func writeAgentReverseResponse(conn *websocket.Conn, requestID, instanceID string, status int, body []byte) error {
	return framework_reverse.WriteChunkedResponse(func(payload string) error {
		return writeAgentReverseText(conn, payload)
	}, requestID, instanceID, status, body, framework_reverse.DefaultResponseChunkBytes)
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

func logAgentReverseStatef(state string, format string, args ...interface{}) {
	agentReverseStateLock.Lock()
	defer agentReverseStateLock.Unlock()
	if state == agentReverseLastState {
		return
	}
	agentReverseLastState = state
	log.Infof(format, args...)
}
