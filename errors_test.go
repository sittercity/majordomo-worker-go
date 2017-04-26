package majordomo_worker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_GracefulShutdown(t *testing.T) {
	e := GracefulShutdown("")
	assert.Error(t, e)
	assert.IsType(t, GracefulShutdown(""), e)
}

func Test_GracefulShutdown_ReturnsTrue(t *testing.T) {
	e := GracefulShutdown("")
	assert.True(t, e.GracefulShutdown())
}

func Test_GracefulShutdown_ReturnsString(t *testing.T) {
	e := GracefulShutdown("")
	assert.Equal(t, "Shutdown signal detected, exiting ...", e.Error())
}
