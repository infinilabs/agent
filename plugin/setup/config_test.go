/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package setup

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpdateYAMLNodePreservesComments(t *testing.T) {
	original := `# Top comment
env:
  API_BINDING: "0.0.0.0:2900"  # inline comment

# Path settings
path.data: data
path.logs: log
`

	updated, err := updateYAMLNode([]byte(original), "committed_setup_tasks", []string{"uuid1", "uuid2"})
	assert.NoError(t, err)

	// Comments should be preserved
	assert.Contains(t, string(updated), "# Top comment")
	assert.Contains(t, string(updated), "# inline comment")
	assert.Contains(t, string(updated), "# Path settings")

	// New field should exist
	assert.Contains(t, string(updated), "committed_setup_tasks")
}

func TestUpdateYAMLNodeUpdatesExistingField(t *testing.T) {
	original := `# config file
path.data: data
path.logs: log
`

	updated, err := updateYAMLNode([]byte(original), "path.data", "new_data_dir")
	assert.NoError(t, err)

	// Comment preserved
	assert.Contains(t, string(updated), "# config file")
	// Value updated
	assert.Contains(t, string(updated), "new_data_dir")
	assert.NotContains(t, string(updated), "path.data: data")
	// Other field unchanged
	assert.Contains(t, string(updated), "path.logs: log")
}

func TestUpdateYAMLNodeAddsNewField(t *testing.T) {
	original := `# config file
path.data: data
path.logs: log
`

	updated, err := updateYAMLNode([]byte(original), "committed_setup_tasks", []string{"uuid1", "uuid2"})
	assert.NoError(t, err)

	// Comments preserved
	assert.Contains(t, string(updated), "# config file")
	// Existing fields unchanged
	assert.Contains(t, string(updated), "path.data: data")
	assert.Contains(t, string(updated), "path.logs: log")
	// New field added
	assert.Contains(t, string(updated), "committed_setup_tasks")
	assert.Contains(t, string(updated), "uuid1")
	assert.Contains(t, string(updated), "uuid2")
}

func TestUpdateYAMLNodeUpdatesNestedFieldWithDottedKey(t *testing.T) {
	original := `# nested config
paths:
  data: data
  logs: log
`

	updated, err := updateYAMLNode([]byte(original), "paths.data", "new_data_dir")
	assert.NoError(t, err)

	s := string(updated)

	// Comment preserved
	assert.Contains(t, s, "# nested config")
	// Nested value updated
	assert.Contains(t, s, "paths:")
	assert.Contains(t, s, "data: new_data_dir")
	assert.NotContains(t, s, "data: data")
	// Other nested field unchanged
	assert.Contains(t, s, "logs: log")
	// Should not introduce a flat "paths.data" key
	assert.NotContains(t, s, "paths.data:")
}

func TestUpdateYAMLNodeMixedDottedAndNested(t *testing.T) {
	// "security.ssl" is a flat dotted key at the top level,
	// with nested children underneath.
	original := `security.ssl:
  http:
    cert_file: old.crt
    key_file: old.key
`

	updated, err := updateYAMLNode([]byte(original), "security.ssl.http.cert_file", "new.crt")
	assert.NoError(t, err)

	s := string(updated)

	// The value should be updated in-place under the existing structure.
	assert.Contains(t, s, "cert_file: new.crt")
	assert.NotContains(t, s, "cert_file: old.crt")
	// Other sibling field unchanged.
	assert.Contains(t, s, "key_file: old.key")
	// Must not create a duplicate "security" or "security.ssl" key.
	assert.NotContains(t, s, "security:\n")
}

func TestUpdateYAMLNodeMixedDottedCreateNew(t *testing.T) {
	// Existing partial-dotted key; add a new child that doesn't exist yet.
	original := `security.ssl:
  http:
    cert_file: node.crt
`

	updated, err := updateYAMLNode([]byte(original), "security.ssl.http.ca_file", "ca.crt")
	assert.NoError(t, err)

	s := string(updated)

	// New field added under existing structure.
	assert.Contains(t, s, "ca_file: ca.crt")
	// Existing field unchanged.
	assert.Contains(t, s, "cert_file: node.crt")
}
