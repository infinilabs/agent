/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package setup

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ErrEnrollTokenInvalid indicates the enroll_token itself is malformed or
// missing required fields (as opposed to a remote-cluster connectivity error).
var ErrEnrollTokenInvalid = errors.New("invalid enroll token")

// EnrollTokenData is the decoded form of the base64 enroll_token field sent in
// POST /setup/cluster/join.
type EnrollTokenData struct {
	Endpoint    string `json:"endpoint"`
	ClusterName string `json:"cluster_name"`
	ClusterUUID string `json:"cluster_uuid"`
	AccessToken string `json:"access_token"`
}

// EnrollNodeResponse is the response from POST /_cluster/_enroll/node on the
// existing cluster. It carries TLS material, seed addresses, and cluster health
// needed to configure the joining node.
type EnrollNodeResponse struct {
	ClusterName   string             `json:"cluster_name"`
	ClusterUUID   string             `json:"cluster_uuid"`
	Security      enrollSecurityInfo `json:"security"`
	SeedAddresses []string           `json:"seed_addresses"`
	Version       string             `json:"version"`
	Plugins       []string           `json:"plugins"`
	JDK           enrollJDKInfo      `json:"jdk"`
	Health        map[string]any     `json:"health"`

	// ResponseAccessToken is the second access_token returned in the response
	// body (1h expiry, read-nodes permission). It is different from the
	// 30-min enroll token stored in EnrollTokenData.
	ResponseAccessToken string `json:"access_token"`

	// The fields below are populated from EnrollTokenData, not the API response
	// body, and are excluded from any JSON re-serialisation.
	Endpoint string `json:"-"`
}

// enrollSecurityInfo holds the security section of the enroll/node response.
type enrollSecurityInfo struct {
	Enabled             bool   `json:"enabled"`
	SSLTransportEnabled bool   `json:"ssl.transport.enabled"`
	TransportCaCert     string `json:"transport_ca_cert"`
	TransportCert       string `json:"transport_cert"`
	TransportKey        string `json:"transport_key"`
	SSLHTTPEnabled      bool   `json:"ssl.http.enabled"`
	HTTPCaKey           string `json:"http_ca_key"`
	HTTPCert            string `json:"http_cert"`
	HTTPKey             string `json:"http_key"`
}

type enrollJDKInfo struct {
	Version string `json:"version"`
}

// fetchEnrollInfo decodes the base64 enroll_token, then calls the existing
// cluster's POST /_cluster/_enroll/node endpoint to obtain joining
// configuration, TLS material, and cluster health.
//
// On success both return values are non-nil. On failure the first return value
// may be non-nil if decoding succeeded before the error occurred (useful for
// error context).
func fetchEnrollInfo(token string) (*EnrollTokenData, *EnrollNodeResponse, error) {
	raw, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		// Try URL-safe encoding as a fallback.
		raw, err = base64.URLEncoding.DecodeString(token)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: not valid base64: %v", ErrEnrollTokenInvalid, err)
		}
	}

	var td EnrollTokenData
	if err := json.Unmarshal(raw, &td); err != nil {
		return nil, nil, fmt.Errorf("%w: invalid JSON payload: %v", ErrEnrollTokenInvalid, err)
	}
	if td.Endpoint == "" || td.AccessToken == "" {
		return nil, nil, fmt.Errorf("%w: missing required fields (endpoint, access_token)", ErrEnrollTokenInvalid)
	}

	// TLS verification is skipped here: we have no CA cert yet. The cert will
	// be obtained from the response body (http_ca_cert).
	client := &http.Client{
		Timeout: httpTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402
		},
	}

	url := strings.TrimRight(td.Endpoint, "/") + "/_cluster/_enroll/node"
	httpReq, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return &td, nil, fmt.Errorf("build enroll request: %w", err)
	}
	httpReq.Header.Set("X-API-TOKEN", td.AccessToken)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(httpReq)
	if err != nil {
		return &td, nil, fmt.Errorf("call enroll endpoint %s: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &td, nil, fmt.Errorf("read enroll response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return &td, nil, fmt.Errorf("enroll endpoint returned HTTP %d: %s", resp.StatusCode, body)
	}

	var enroll EnrollNodeResponse
	if err := json.Unmarshal(body, &enroll); err != nil {
		return &td, nil, fmt.Errorf("parse enroll response: %w", err)
	}
	enroll.Endpoint = td.Endpoint

	return &td, &enroll, nil
}
