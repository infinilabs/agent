/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package util

import (
	"fmt"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/util"
)

func GetLocalNodeInfo(endpoint string, auth *elastic.BasicAuth)(string, *elastic.NodesInfo, error) {
	url := fmt.Sprintf("%s/_nodes/_local", endpoint)
	req := util.Request{
		Method: util.Verb_GET,
		Url: url,
	}
	if auth != nil {
		req.SetBasicAuth(auth.Username, auth.Password)
	}
	resp, err := util.ExecuteRequest(&req)

	if err != nil {
		return "", nil, err
	}
	if resp.StatusCode != 200 {
		return "", nil, fmt.Errorf(string(resp.Body))
	}

	node := elastic.NodesResponse{}
	err=util.FromJSONBytes(resp.Body,&node)
	if err != nil {
		return "", nil, err
	}
	for k, n := range node.Nodes {
		return k, &n, nil
	}
	return "", nil, fmt.Errorf("node not found")
}

func GetClusterVersion(endpoint string, auth *elastic.BasicAuth)(*elastic.ClusterInformation, error) {
	req := util.Request{
		Method: util.Verb_GET,
		Url: endpoint,
	}
	if auth != nil {
		req.SetBasicAuth(auth.Username, auth.Password)
	}
	resp, err := util.ExecuteRequest(&req)

	if err != nil {
		return  nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf(string(resp.Body))
	}

	version := elastic.ClusterInformation{}
	err = util.FromJSONBytes(resp.Body, &version)
	if err != nil {
		return nil, err
	}
	return &version, nil
}

func GetLocalNodesInfo(endpoint string, auth *elastic.BasicAuth)(map[string]elastic.NodesInfo, error) {
	url := fmt.Sprintf("%s/_nodes", endpoint)
	req := util.Request{
		Method: util.Verb_GET,
		Url: url,
	}
	if auth != nil {
		req.SetBasicAuth(auth.Username, auth.Password)
	}
	resp, err := util.ExecuteRequest(&req)

	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return  nil, fmt.Errorf(string(resp.Body))
	}

	nodesRes := &elastic.NodesResponse{}
	err=util.FromJSONBytes(resp.Body,nodesRes)
	if err != nil {
		return nil, err
	}
	ips := util.GetLocalIPs()
	nodesInfo := map[string]elastic.NodesInfo{}
	for k, n := range nodesRes.Nodes {
		if util.StringInArray(ips, n.Ip){
			nodesInfo[k] = n
		}
	}
	return nodesInfo, nil
}