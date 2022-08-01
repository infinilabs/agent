/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package model

type Host struct {
	IPs      []string  `json:"ip,omitempty"`
	Clusters []Cluster `json:"clusters,omitempty"`
	TLS      bool      `json:"tls"`
}

type Cluster struct {
	Name     string `json:"cluster_name,omitempty"`
	ID       string `json:"cluster_id,omitempty"`
	UserName string `json:"user_name,omitempty"`
	Password string `json:"password,omitempty"`
}
