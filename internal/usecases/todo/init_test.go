package todo

import (
	"testing"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/stretchr/testify/assert"
)

func TestInitCreateTodo_Initialize(t *testing.T) {
	t.Parallel()

	ict := InitCreateTodo{}

	ctx, err := ict.Initialize(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	registeredCreateTodo, err := depend.Resolve[Create]()
	assert.NoError(t, err)
	assert.NotNil(t, registeredCreateTodo)
}

func TestInitDeleteTodo_Initialize(t *testing.T) {
	t.Parallel()

	idt := InitDeleteTodo{}

	ctx, err := idt.Initialize(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	registeredDeleteTodo, err := depend.Resolve[Delete]()
	assert.NoError(t, err)
	assert.NotNil(t, registeredDeleteTodo)
}

func TestInitListTodos_Initialize(t *testing.T) {
	t.Parallel()

	ilt := InitListTodos{}

	ctx, err := ilt.Initialize(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	registeredListTodos, err := depend.Resolve[List]()
	assert.NoError(t, err)
	assert.NotNil(t, registeredListTodos)
}

func TestInitCreator_Initialize(t *testing.T) {
	t.Parallel()

	ict := InitCreator{}

	ctx, err := ict.Initialize(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	registeredTodoCreator, err := depend.Resolve[Creator]()
	assert.NoError(t, err)
	assert.NotNil(t, registeredTodoCreator)

}

func TestInitDeleter_Initialize(t *testing.T) {
	t.Parallel()

	id := InitDeleter{
		TimeProvider: core.NewMockCurrentTimeProvider(t),
	}

	ctx, err := id.Initialize(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	// Verify that the Deleter is registered
	todoDeleter, err := depend.Resolve[Deleter]()
	assert.NoError(t, err)
	assert.NotNil(t, todoDeleter)
}

func TestInitUpdater_Initialize(t *testing.T) {
	t.Parallel()

	iut := InitUpdater{}

	ctx, err := iut.Initialize(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	registeredTodoUpdater, err := depend.Resolve[Updater]()
	assert.NoError(t, err)
	assert.NotNil(t, registeredTodoUpdater)
}

func TestInitUpdateTodo_Initialize(t *testing.T) {
	t.Parallel()

	iut := InitUpdateTodo{}

	ctx, err := iut.Initialize(t.Context())
	assert.NoError(t, err)
	assert.NotNil(t, ctx)

	registeredUpdateTodo, err := depend.Resolve[Update]()
	assert.NoError(t, err)
	assert.NotNil(t, registeredUpdateTodo)
}
