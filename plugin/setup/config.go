/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config definition of the setup package.
type setupConfig struct {
	EnvCheck setupEnvCheckConfig `config:"env_check"`
	Task     setupTaskConfig     `config:"task"`
}

type setupTaskConfig struct {
	// IDs of the committed tasks
	Committed []string `config:"committed"`
}

// ReadConfigStringSlice reads a string-sequence value at the given
// dotted key path directly from the YAML file on disk.
//
// Returns an empty slice if the key does not exist.
func ReadConfigStringSlice(path string, key string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var root yaml.Node
	if err := yaml.Unmarshal(content, &root); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		return nil, nil
	}
	mapping := root.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return nil, nil
	}

	node := findYAMLNode(mapping, strings.Split(key, "."))
	if node == nil || node.Kind != yaml.SequenceNode {
		return nil, nil
	}

	result := make([]string, 0, len(node.Content))
	for _, child := range node.Content {
		result = append(result, child.Value)
	}
	return result, nil
}

// findYAMLNode navigates a mapping node along the given path and returns the
// value node at the leaf, or nil if any segment is missing.
func findYAMLNode(mapping *yaml.Node, path []string) *yaml.Node {
	current := mapping
	for i, part := range path {
		if current.Kind != yaml.MappingNode {
			return nil
		}
		idx := -1
		for j := 0; j < len(current.Content); j += 2 {
			if current.Content[j].Value == part {
				idx = j
				break
			}
		}
		if idx < 0 {
			return nil
		}
		if i == len(path)-1 {
			return current.Content[idx+1]
		}
		current = current.Content[idx+1]
	}
	return nil
}

// UpdateConfigField updates a field in the yaml config file.
// It preserves comments and uses atomic write.
func UpdateConfigField(path string, key string, value interface{}) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	newContent, err := updateYAMLNode(content, key, value)
	if err != nil {
		return fmt.Errorf("update yaml node: %w", err)
	}

	return atomicWriteFile(path, newContent)
}

// updateYAMLNode updates a field in yaml content while preserving comments.
func updateYAMLNode(content []byte, key string, value interface{}) ([]byte, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(content, &root); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}

	// Ensure document node
	if root.Kind != yaml.DocumentNode {
		root = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{&root}}
	}

	mapping := ensureDocumentMapping(&root)

	// Encode new value
	valueNode := &yaml.Node{}
	if err := valueNode.Encode(value); err != nil {
		return nil, fmt.Errorf("encode value: %w", err)
	}

	parts := strings.Split(key, ".")
	setYAMLValueGreedy(mapping, parts, valueNode)

	return yaml.Marshal(&root)
}

// ensureDocumentMapping ensures the root document node has a mapping node as its first content
// and returns that mapping.
func ensureDocumentMapping(root *yaml.Node) *yaml.Node {
	if root.Kind != yaml.DocumentNode {
		*root = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{root}}
	}

	if len(root.Content) > 0 && root.Content[0].Kind == yaml.MappingNode {
		return root.Content[0]
	}

	mapping := &yaml.Node{Kind: yaml.MappingNode}
	root.Content = []*yaml.Node{mapping}
	return mapping
}

// setYAMLValueGreedy resolves a dotted key path using greedy longest-prefix
// matching at each mapping level. This handles YAML files that mix flat dotted
// keys (e.g. "security.ssl.http.cert_file") with nested structure.
//
// At each mapping level it tries the longest prefix of the remaining segments
// that matches a literal key, then recurses into that value with the leftover
// segments. If no existing prefix matches, it falls back to single-segment
// descent (creating intermediate mappings as needed).
//
// Examples — key "a.b.c.d":
//
//	flat:   "a.b.c.d: val"                → prefix "a.b.c.d" matches at root
//	nested: "a: b: c: d: val"             → "a" matches, recurse "b.c.d"
//	mixed:  "a.b: c: d: val"              → "a.b" matches at root, recurse "c.d"
func setYAMLValueGreedy(mapping *yaml.Node, parts []string, valueNode *yaml.Node) {
	// Try longest prefix first — this naturally prefers literal dotted keys.
	for prefixLen := len(parts); prefixLen >= 1; prefixLen-- {
		candidate := strings.Join(parts[:prefixLen], ".")
		idx := findMappingKeyIndex(mapping, candidate)
		if idx < 0 {
			continue
		}

		if prefixLen == len(parts) {
			// Exact match — replace value.
			mapping.Content[idx+1] = valueNode
			return
		}

		// Matched a prefix — descend into its value with remaining segments.
		child := mapping.Content[idx+1]
		if child.Kind != yaml.MappingNode {
			child = &yaml.Node{Kind: yaml.MappingNode}
			mapping.Content[idx+1] = child
		}
		setYAMLValueGreedy(child, parts[prefixLen:], valueNode)
		return
	}

	// No existing key matches any prefix — create with single-segment key.
	if len(parts) == 1 {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: parts[0]}
		mapping.Content = append(mapping.Content, keyNode, valueNode)
		return
	}

	keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: parts[0]}
	child := &yaml.Node{Kind: yaml.MappingNode}
	mapping.Content = append(mapping.Content, keyNode, child)
	setYAMLValueGreedy(child, parts[1:], valueNode)
}

// findMappingKeyIndex returns the index of the key node whose Value equals key,
// or -1 if not found. mapping must be a MappingNode.
func findMappingKeyIndex(mapping *yaml.Node, key string) int {
	for i := 0; i < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return i
		}
	}
	return -1
}

// atomicWriteFile writes data to path atomically using temp file + rename.
func atomicWriteFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	tmp, err := os.CreateTemp(dir, base+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	success := false
	defer func() {
		if !success {
			os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	// Preserve permissions
	if info, err := os.Stat(path); err == nil {
		os.Chmod(tmpPath, info.Mode())
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}

	success = true
	return nil
}
