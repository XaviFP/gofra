package gofra

import (
	"fmt"
	"log"
	"sort"
)

type Handler func(event Event) *Reply
type ChainHandler func(accumulated *Event)

type EventHandler struct {
	Handler    Handler
	Priority   int
	PluginName string
	Chain      ChainHandler
}

type EventManager struct {
	handlers map[string][]EventHandler
	logger   Logger
}

func NewEventManager(logger Logger) EventManager {
	return EventManager{
		handlers: make(map[string][]EventHandler),
		logger:   logger,
	}
}

func (em EventManager) Subscribe(eventName, pluginName string, handler Handler, chain ChainHandler, priority int) {
	if em.handlers[eventName] == nil {
		em.handlers[eventName] = []EventHandler{}
	}

	em.handlers[eventName] = append(
		em.handlers[eventName],
		EventHandler{
			Handler:    handler,
			Priority:   priority,
			PluginName: pluginName,
			Chain:      chain,
		},
	)

	em.sort(eventName)

	em.Publish(Event{
		Name: "addedEventListener",
		Payload: map[string]interface{}{
			"event":    eventName,
			"plugin":   pluginName,
			"chained":  chain != nil,
			"priority": priority,
		},
	})
}

func (em EventManager) Publish(event Event) *Reply {
	var reply *Reply

	handlers, exist := em.handlers[event.Name]
	if !exist || len(handlers) == 0 {
		em.logger.Debug(fmt.Sprintf("No handlers for event: %s ", event.Name))

		return nil
	}

	chainedHandlers := []EventHandler{}

	for _, handler := range handlers {
		if handler.Chain != nil {
			chainedHandlers = append(chainedHandlers, handler)
		} else {
			r := runHandler(handler, event)
			if reply == nil && r != nil {
				reply = r
				em.logger.Debug(fmt.Sprintf("event %s was answered with reply %v", event.Name, reply))
			}
		}
	}

	for _, handler := range chainedHandlers {
		runChainHandler(handler, &event)
	}

	return reply
}

func (em EventManager) SetPriority(eventName, pluginName string, priority int) error {
	var priorityChanged, pluginFound bool

	_, exist := em.handlers[eventName]
	if !exist {
		return fmt.Errorf("event %s not found", eventName)
	}

	for i, element := range em.handlers[eventName] {
		if element.PluginName == pluginName {
			pluginFound = true

			if element.Priority != priority {
				em.handlers[eventName][i].Priority = priority
				priorityChanged = true
			}

			break
		}
	}
	if !pluginFound {
		return fmt.Errorf("no %s handler found for plugin %s", eventName, pluginName)
	}
	if !priorityChanged {
		// If a given handler for a plugin had the same priority before, then do nothing
		return nil
	}

	em.sort(eventName)

	return nil
}

// sort sorts handlers in descending priority order
func (em EventManager) sort(eventName string) {
	sort.Slice(em.handlers[eventName], func(i, j int) bool {
		return em.handlers[eventName][i].Priority > em.handlers[eventName][j].Priority
	})
}

type Event struct {
	Name    string
	MB      MessageBody
	Payload map[string]interface{}
}

func (e *Event) SetStanza(stanza interface{}) {
	if e.Payload == nil {
		e.Payload = make(map[string]interface{})
	}
	e.Payload["stanza"] = stanza
}

func (e *Event) GetStanza() interface{} {
	if e.Payload == nil {
		e.Payload = make(map[string]interface{})
	}
	stanza, exists := e.Payload["stanza"]
	if !exists {
		return nil
	}
	return stanza
}

type Reply struct {
	Payload      map[string]interface{}
}

// Data access interface for text-based commands to answer a suitable message.
func (r *Reply) SetAnswer(answer string) {
	if r.Payload == nil {
		r.Payload = make(map[string]interface{})
	}

	r.Payload["answer"] = answer
}

// Data access interface for command plugin to receive the answer from an specific command.
func (r *Reply) GetAnswer() string {
	if r.Payload == nil {
		r.Payload = make(map[string]interface{})
	}

	answer, exists := r.Payload["answer"]
	if !exists {
		return ""
	}

	strAnswer, ok := answer.(string)
	if !ok {
		return ""
	}

	return strAnswer
}

func runHandler(h EventHandler, e Event) *Reply {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("plugin '%s' handler for event '%s' failed: %s", h.PluginName, e.Name, err)
		}
	}()

	return h.Handler(e)
}

func runChainHandler(h EventHandler, e *Event) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("plugin '%s' chain handler for event '%s' failed: %s", h.PluginName, e.Name, err)
		}
	}()

	h.Chain(e)
}
