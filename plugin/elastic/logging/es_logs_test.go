package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFixLogPath(t *testing.T) {
	v := fixLogPath(`C:/logs/246`)
	assert.Equal(t, v, `C:\logs\246`)
}
