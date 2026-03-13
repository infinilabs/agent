/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package setup

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPluginConfigUnmarshalJSONString(t *testing.T) {
	var cfg PluginConfig
	err := json.Unmarshal([]byte(`"analysis-ik"`), &cfg)
	require.NoError(t, err)

	assert.Equal(t, PluginConfig{Name: "analysis-ik"}, cfg)
}

func TestPluginConfigUnmarshalJSONObject(t *testing.T) {
	var cfg PluginConfig
	err := json.Unmarshal([]byte(`{"url":"file:///tmp/plugin.zip"}`), &cfg)
	require.NoError(t, err)

	assert.Equal(t, PluginConfig{URL: "file:///tmp/plugin.zip"}, cfg)
}

func TestPluginConfigUnmarshalTrimsWhitespace(t *testing.T) {
	var cfg PluginConfig
	err := json.Unmarshal([]byte(" \n\t{\"url\":\"file:///tmp/plugin.zip\"}\r\n "), &cfg)
	require.NoError(t, err)

	assert.Equal(t, PluginConfig{URL: "file:///tmp/plugin.zip"}, cfg)
}

func TestPluginConfigUnmarshalResetsPreviousState(t *testing.T) {
	cfg := PluginConfig{Name: "stale-name"}
	err := json.Unmarshal([]byte(`{"url":"file:///tmp/plugin.zip"}`), &cfg)
	require.NoError(t, err)

	assert.Equal(t, PluginConfig{URL: "file:///tmp/plugin.zip"}, cfg)
}

func TestPluginConfigUnmarshalRejectsEmptyName(t *testing.T) {
	var cfg PluginConfig
	err := json.Unmarshal([]byte(`""`), &cfg)
	require.Error(t, err)

	assert.Contains(t, err.Error(), "plugin name must be non-empty")
}

func TestPluginConfigUnmarshalRejectsUnknownField(t *testing.T) {
	var cfg PluginConfig
	err := json.Unmarshal([]byte(`{"url":"file:///tmp/plugin.zip","name":"extra"}`), &cfg)
	require.Error(t, err)

	assert.Contains(t, err.Error(), "parse plugin object")
}

func TestPluginConfigUnmarshalRejectsEmptyURL(t *testing.T) {
	var cfg PluginConfig
	err := json.Unmarshal([]byte(`{"url":""}`), &cfg)
	require.Error(t, err)

	assert.Contains(t, err.Error(), "non-empty \"url\" field")
}

func TestPluginConfigUnmarshalRejectsInvalidJSONKind(t *testing.T) {
	var cfg PluginConfig
	err := json.Unmarshal([]byte(`123`), &cfg)
	require.Error(t, err)

	assert.Contains(t, err.Error(), "plugin must be a string or an object with \"url\"")
}

func TestPluginConfigMarshalJSONName(t *testing.T) {
	data, err := json.Marshal(PluginConfig{Name: "analysis-ik"})
	require.NoError(t, err)

	assert.JSONEq(t, `"analysis-ik"`, string(data))
}

func TestPluginConfigMarshalJSONURL(t *testing.T) {
	data, err := json.Marshal(PluginConfig{URL: "file:///tmp/plugin.zip"})
	require.NoError(t, err)

	assert.JSONEq(t, `{"url":"file:///tmp/plugin.zip"}`, string(data))
}

func TestPluginConfigJSONRoundTrip(t *testing.T) {
	tests := []PluginConfig{
		{Name: "analysis-ik"},
		{URL: "file:///tmp/plugin.zip"},
	}

	for _, tt := range tests {
		data, err := json.Marshal(tt)
		require.NoError(t, err)

		var got PluginConfig
		err = json.Unmarshal(data, &got)
		require.NoError(t, err)

		assert.Equal(t, tt, got)
	}
}

func TestPluginConfigSliceUnmarshalJSON(t *testing.T) {
	var cfgs []PluginConfig
	err := json.Unmarshal([]byte(`[
		"analysis-ik",
		{"url":"file:///tmp/plugin-a.zip"},
		"analysis-pinyin",
		{"url":"https://example.com/plugin-b.zip"}
	]`), &cfgs)
	require.NoError(t, err)

	assert.Equal(t, []PluginConfig{
		{Name: "analysis-ik"},
		{URL: "file:///tmp/plugin-a.zip"},
		{Name: "analysis-pinyin"},
		{URL: "https://example.com/plugin-b.zip"},
	}, cfgs)
}

func TestPluginConfigSliceMarshalJSON(t *testing.T) {
	data, err := json.Marshal([]PluginConfig{
		{Name: "analysis-ik"},
		{URL: "file:///tmp/plugin-a.zip"},
		{Name: "analysis-pinyin"},
		{URL: "https://example.com/plugin-b.zip"},
	})
	require.NoError(t, err)

	assert.JSONEq(t, `[
		"analysis-ik",
		{"url":"file:///tmp/plugin-a.zip"},
		"analysis-pinyin",
		{"url":"https://example.com/plugin-b.zip"}
	]`, string(data))
}

func TestPluginConfigMarshalJSONRejectsInvalidState(t *testing.T) {
	tests := []struct {
		name string
		cfg  PluginConfig
	}{
		{
			name: "empty",
			cfg:  PluginConfig{},
		},
		{
			name: "both name and url",
			cfg:  PluginConfig{Name: "analysis-ik", URL: "file:///tmp/plugin.zip"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := json.Marshal(tt.cfg)
			require.Error(t, err)
		})
	}
}
