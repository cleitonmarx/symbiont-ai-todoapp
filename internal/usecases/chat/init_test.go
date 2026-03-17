package chat

import (
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/outbox"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/stretchr/testify/assert"
)

func TestInitDeleteConversation_Initialize(t *testing.T) {
	t.Parallel()

	idc := InitDeleteConversation{}

	_, err := idc.Initialize(t.Context())
	assert.NoError(t, err)

	uc, err := depend.Resolve[DeleteConversation]()
	assert.NoError(t, err)
	assert.NotNil(t, uc)

}

func TestInitConversationCompactor_Initialize(t *testing.T) {
	t.Parallel()

	i := InitConversationCompactor{}

	ctx, err := i.Initialize(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	uc, err := depend.Resolve[ConversationCompactor]()
	assert.NoError(t, err)
	assert.NotNil(t, uc)
}

func TestInitGenerateConversationTitle_Initialize(t *testing.T) {
	t.Parallel()

	i := InitGenerateConversationTitle{}

	ctx, err := i.Initialize(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	registeredUseCase, err := depend.Resolve[GenerateConversationTitle]()
	assert.NoError(t, err)
	assert.NotNil(t, registeredUseCase)
}

func TestInitListAvailableModels_Initialize(t *testing.T) {
	t.Parallel()

	assistantCatalog := assistant.NewMockModelCatalog(t)
	init := InitListAvailableModels{
		AssistantCatalog: assistantCatalog,
	}

	_, err := init.Initialize(t.Context())
	assert.NoError(t, err)

	uc, err := depend.Resolve[ListAvailableModels]()
	assert.NoError(t, err)
	assert.NotNil(t, uc)
}

func TestInitListChatMessages_Initialize(t *testing.T) {
	t.Parallel()

	idc := InitListChatMessages{}

	_, err := idc.Initialize(t.Context())
	assert.NoError(t, err)

	uc, err := depend.Resolve[ListChatMessages]()
	assert.NoError(t, err)
	assert.NotNil(t, uc)
}

func TestInitListConversations_Initialize(t *testing.T) {
	t.Parallel()

	ilc := InitListConversations{}

	ctx, err := ilc.Initialize(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	registeredListConversations, err := depend.Resolve[ListConversations]()
	assert.NoError(t, err)
	assert.NotNil(t, registeredListConversations)
}

func TestInitStreamChat_Initialize(t *testing.T) {
	t.Parallel()

	i := InitStreamChat{}

	ctx, err := i.Initialize(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	// Verify that the StreamChat use case is registered
	streamChatUseCase, err := depend.Resolve[StreamChat]()
	assert.NoError(t, err)
	assert.NotNil(t, streamChatUseCase)
}

func TestInitConversationCreator_Initialize(t *testing.T) {
	t.Parallel()

	i := InitConversationCreator{}
	_, err := i.Initialize(t.Context())
	assert.NoError(t, err)

	component, err := depend.Resolve[ConversationCreator]()
	assert.NoError(t, err)
	assert.NotNil(t, component)
}

func TestInitActionPipeline_Initialize(t *testing.T) {
	t.Parallel()

	i := InitActionPipeline{}
	_, err := i.Initialize(t.Context())
	assert.NoError(t, err)

	component, err := depend.Resolve[ActionPipeline]()
	assert.NoError(t, err)
	assert.NotNil(t, component)
}

func TestInitTurnRunner_Initialize(t *testing.T) {
	t.Parallel()

	i := InitTurnRunner{}
	_, err := i.Initialize(t.Context())
	assert.NoError(t, err)

	component, err := depend.Resolve[TurnRunner]()
	assert.NoError(t, err)
	assert.NotNil(t, component)
}

func TestInitTurnStateBuilder_Initialize(t *testing.T) {
	t.Parallel()

	i := InitTurnStateBuilder{}
	_, err := i.Initialize(t.Context())
	assert.NoError(t, err)

	component, err := depend.Resolve[TurnStateBuilder]()
	assert.NoError(t, err)
	assert.NotNil(t, component)
}

func TestInitSubmitActionApproval_Initialize(t *testing.T) {
	t.Parallel()

	publisher := outbox.NewMockEventPublisher(t)
	init := InitSubmitActionApproval{
		Publisher: publisher,
	}

	_, err := init.Initialize(t.Context())
	assert.NoError(t, err)

	uc, err := depend.Resolve[SubmitActionApproval]()
	assert.NoError(t, err)
	assert.NotNil(t, uc)
}

func TestInitUpdateConversation_Initialize(t *testing.T) {
	t.Parallel()

	iuc := InitUpdateConversation{}

	ctx, err := iuc.Initialize(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	registeredUpdateConversation, err := depend.Resolve[UpdateConversation]()
	assert.NoError(t, err)
	assert.NotNil(t, registeredUpdateConversation)
}
