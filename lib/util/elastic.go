/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package util

import (
	"context"
	"fmt"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/model"
	"infini.sh/framework/core/util"
	"time"
)

func GetLocalNodeInfo(endpoint string, auth *model.BasicAuth)(string, *elastic.NodesInfo, error) {
	url := fmt.Sprintf("%s/_nodes/_local", endpoint)
	req := util.Request{
		Method: util.Verb_GET,
		Url: url,
	}
	if auth != nil {
		req.SetBasicAuth(auth.Username, auth.Password)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	req.Context = ctx
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

func GetClusterVersion(endpoint string, auth *model.BasicAuth)(*elastic.ClusterInformation, error) {
	req := util.Request{
		Method: util.Verb_GET,
		Url: endpoint,
	}
	if auth != nil {
		req.SetBasicAuth(auth.Username, auth.Password)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	req.Context = ctx
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