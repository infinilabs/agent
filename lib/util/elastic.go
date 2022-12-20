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