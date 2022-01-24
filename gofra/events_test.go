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
		func(e Event) *Reply {
			ran = true
			return nil
		},
		nil,
		0,
	)

	assert.True(t, ran)
}

func TestEvents_PublishSubscribeChain(t *testing.T) {
	em := NewEventManager(Logger{})
	var testString string

	em.Subscribe(
		"addedEventListener",
		"testPlugin",
		nil,
		func(e *Event) {
			e.Payload = map[string]interface{}{"test": testString}
		},
		1,
	)
	em.Subscribe(
		"addedEventListener",
		"testPlugin",
		nil,
		func(e *Event) {
			test, ok := e.Payload["test"].(string)
			assert.True(t, ok)
			assert.EqualValues(t, testString, test)
		},
		0,
	)

	em.Subscribe(
		"testEvent1",
		"testPlugin",
		panicHandler,
		nil,
		0,
	)

	em.Subscribe(
		"testEvent2",
		"testPlugin",
		nil,
		chainPanicHandler,
		0,
	)
	r := em.Publish(Event{Name: "testEvent1"})
	assert.True(t, r.EventHandled)

	r = em.Publish(Event{Name: "testEvent2"})
	assert.True(t, r.EventHandled)
}

func exampleHandler(e Event) *Reply {
	return nil
}

func panicHandler(e Event) *Reply {
	panic("Panic on purpose")
}

func chainPanicHandler(e *Event) {
	panic("Panic on purpose")
}

func TestEvents_NoHandlersPublish(t *testing.T) {
	em := NewEventManager(Logger{})

	r := em.Publish(Event{Name: "testEvent"})
	assert.False(t, r.EventHandled)
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

	err := em.SetPriority("addedEventListener", "testPlugin1", 3)
	assert.Nil(t, err)
	assert.Equal(t, "testPlugin1", em.handlers["addedEventListener"][0].PluginName)

	err = em.SetPriority("madeUpEvent", "testPlugin1", 3)
	assert.NotNil(t, err)

	em.Subscribe(
		"testEvent",
		"testPlugin1",
		exampleHandler,
		nil,
		0,
	)
	err = em.SetPriority("testEvent", "testPlugin5", 3)
	assert.NotNil(t, err)

	err = em.SetPriority("testEvent", "testPlugin1", 0)
	assert.Nil(t, err)
}

func TestEvents_SetStanza(t *testing.T) {
	e := Event{}
	e.SetStanza(MessageBody{Body: "Hello body"})

	testMessageBody, ok := e.Payload["stanza"].(MessageBody)
	assert.True(t, ok)
	assert.EqualValues(t, testMessageBody.Body, "Hello body")
}

func TestEvents_GetStanza(t *testing.T) {
	e := Event{}
	assert.Nil(t, e.GetStanza())

	e.SetStanza(MessageBody{Body: "Hello body"})

	testMessageBody, ok := e.GetStanza().(MessageBody)
	assert.True(t, ok)
	assert.EqualValues(t, testMessageBody.Body, "Hello body")
}

func TestReply_SetAnswer(t *testing.T) {
	r := Reply{}

	r.SetAnswer("testStr")
	assert.Equal(t, "testStr", r.Payload["answer"])
}

func TestReply_GetAnswer(t *testing.T) {
	r := Reply{}

	answer := r.GetAnswer()
	_, exists := r.Payload["answer"]
	assert.False(t, exists)
	assert.Empty(t, answer)

	r.Payload["answer"] = 25
	answer = r.GetAnswer()
	assert.Empty(t, answer)

	r.Payload["answer"] = "testStr"
	answer = r.GetAnswer()
	assert.Equal(t, "testStr", answer)
}
