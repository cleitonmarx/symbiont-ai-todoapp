package app

import (
	"testing"

	"github.com/cleitonmarx/symbiont"
	"github.com/stretchr/testify/require"
)

// TestNewMonolithicApp_Initializers verifies the monolithic constructor returns a valid app instance.
func TestNewMonolithicApp_Initializers(t *testing.T) {
	t.Parallel()

	app := NewMonolithic()
	require.NotNil(t, app, "NewMonolithic should not return nil")
}

// TestNewApps_NotNil verifies every deployable constructor returns a non-nil app.
func TestNewApps_NotNil(t *testing.T) {
	t.Parallel()

	apps := []*symbiont.App{
		NewMonolithic(),
		NewHTTPAPI(),
		NewGraphQLAPI(),
		NewMessageRelay(),
		NewBoardSummaryGenerator(),
		NewChatSummaryGenerator(),
		NewConversationTitleGenerator(),
	}

	for _, app := range apps {
		require.NotNil(t, app)
	}
}
