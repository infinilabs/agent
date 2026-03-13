/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package setup

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecommendJVMMemoryGB(t *testing.T) {
	tests := []struct {
		name      string
		totalGB   float64
		available float64
		expected  float64
	}{
		{
			name:      "half total when within all limits",
			totalGB:   16,
			available: 12,
			expected:  8,
		},
		{
			name:      "available memory caps suggestion",
			totalGB:   16,
			available: 6,
			expected:  6,
		},
		{
			name:      "compressed oops cap at 31g",
			totalGB:   128,
			available: 96,
			expected:  31,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, recommendJVMMemoryGB(tt.totalGB, tt.available))
		})
	}
}

func TestRoundDownToTenths(t *testing.T) {
	assert.Equal(t, 31.0, roundDownToTenths(31.09))
	assert.Equal(t, 30.9, roundDownToTenths(30.99))
	assert.Equal(t, 8.0, roundDownToTenths(8.04))
}
