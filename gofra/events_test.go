package gofra

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvents_PublishSubscribe(t *testing.T) {
	em := NewEventManager(Logger{})
	var ran bool

	em.Subscribe(
		"addedEventListener",
		"testPlugin",
		func(e Event) Reply {
			ran = true
			return Reply{}
		},
		nil,
		0,
	)

	assert.True(t, ran)
}

func exampleHandler(e Event) Reply {
	return Reply{}
}
func TestEvents_Setpriority(t *testing.T) {
	em := NewEventManager(Logger{})

	em.Subscribe(
		"addedEventListener",
		"testPlugin1",
		exampleHandler,
		nil,
		1,
	)
	em.Subscribe(
		"addedEventListener",
		"testPlugin2",
		exampleHandler,
		nil,
		2,
	)

	assert.Equal(t, "testPlugin2", em.handlers["addedEventListener"][0].PluginName)

	em.SetPriority("addedEventListener", "testPlugin1", 3)
	assert.Equal(t, "testPlugin1", em.handlers["addedEventListener"][0].PluginName)
}
