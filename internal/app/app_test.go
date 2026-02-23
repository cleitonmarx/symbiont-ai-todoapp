package app

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewTodoApp_Initializers(t *testing.T) {
	t.Parallel()

	app := NewTodoApp()
	require.NotNil(t, app, "NewTodoApp should not return nil")
}
