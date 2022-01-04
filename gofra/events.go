package gofra

import (
	"fmt"
	"log"
	"sort"
)

// TODO try to rename Events, EventHandler, Hanlder, ChainHandler, etc..
type Events map[string][]EventHandler

func NewEvents(config Config) Events {
	return make(Events)
}

func (e Events) Subscribe(eventName, pluginName string, handler Handler, chain ChainHandler, priority int) {
	if e[eventName] == nil {
		e[eventName] = []EventHandler{}
	}

	e[eventName] = append(
		e[eventName],
		EventHandler{
			Handler:    handler,
			Priority:   priority,
			PluginName: pluginName,
			Chain:      chain,
		},
	)

	e.sortByPriority(eventName)

	event := Event{
		Name: "addedEventListener",
		Payload: map[string]interface{}{
			"event":    eventName,
			"plugin":   pluginName,
			"chained":  chain != nil,
			"priority": priority,
		},
	}

	e.Publish(event)
}

func (e Events) Publish(event Event) Reply {
	var reply Reply

	handlers, exist := e[event.Name]
	if !exist || len(handlers) == 0 {
		fmt.Println("No handlers for event: " + event.Name) // TODO use logger (modify Events type with logger as attribute and pass it down from gofra through NewEvents)
		return Reply{}
	}

	answered := false
	chainedHandlers := []EventHandler{}

	for _, handler := range handlers {
		if handler.Chain != nil {
			chainedHandlers = append(chainedHandlers, handler)
		} else {
			r := runHandlerSafely(handler, event)
			if !answered && !r.Empty {
				reply = r
				answered = true
				log.Printf("event %s was answered with reply %v", event.Name, reply)
			}
		}
	}

	reply.EventHandled = true

	if len(chainedHandlers) == 0 {
		return reply
	}

	for _, handler := range chainedHandlers {
		runChainHandlerSafely(handler, &event)
	}

	return reply
}

func (e Events) SetPriority(eventName, pluginName string) error {
	var priorityChanged bool
	var pluginFound bool
	_, exist := e[eventName]
	if !exist {
		return fmt.Errorf("event %s not found", eventName)
	}
	for i, element := range e[eventName] {
		if element.PluginName == pluginName {
			pluginFound = true
			if element.Priority != options.Priority {
				e[eventName][i].Priority = options.Priority
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
	e.sortByPriority(eventName)
	return nil
}

// Sorts handlers in descending priority order
func (e Events) sortByPriority(eventName string) {
	sort.Slice(e[eventName], func(i, j int) bool {
		return e[eventName][i].Priority > e[eventName][j].Priority
	})
}

type Handler func(event Event) Reply
type ChainHandler func(accumulated *Event)

type EventHandler struct {
	Handler    Handler
	Priority   int
	PluginName string
	Chain      ChainHandler
}

type Event struct {
	Name    string
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
	Ok           bool
	Empty        bool
	EventHandled bool
}

// Example
// type BetweenPlugins struc{
// 	isEventHandledBySomone bool
// 	answer strign
// }

// Data access interface for text-based commands to answer to a suitable message.
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

// TODO rename without safely
func runHandlerSafely(h EventHandler, e Event) Reply {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("plugin '%s' handler for event '%s' failed: %s", h.PluginName, e.Name, err)
		}
	}()

	return h.Handler(e)
}

func runChainHandlerSafely(h EventHandler, e *Event) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("plugin '%s' chain handler for event '%s' failed: %s", h.PluginName, e.Name, err)
		}
	}()

	h.Chain(e)
}
