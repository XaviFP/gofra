package gofra

import (
	"fmt"
	"log"
	"sort"
)

type Events map[string][]EventHandler

func (e Events) Subscribe(eventName, pluginName string, handler Handler, chain ChainHandler, op Options){
	if e[eventName] == nil {
		e[eventName] = []EventHandler{}
	}
	e[eventName] = append(
		e[eventName],
		EventHandler{
			Handler: handler,
			Priority: op.Priority,
			PluginName: pluginName,
			Chain: chain,
		},
	)
	e.sortByPriority(eventName)
	event := Event{
		Name:"addedEventListener",
		Payload: map[string]interface{}{
			"event": eventName,
			"plugin": pluginName,
			"chained": chain != nil,
			"priority": op.Priority,
		},
	}
	e.Publish(event)
}

func (e Events) Publish(event Event) Reply{
	handlers, exist := e[event.Name]
	var reply Reply

	if !exist || len(handlers) == 0 {
		fmt.Println("No handlers for event: " + event.Name)
		reply = Reply{Payload: make(map[string]interface{}), Ok: false, Empty: false}
		reply.SetNoHandlers(true)
		return reply
	}

	answered := false
	chainedHandlers := []EventHandler{}

	for _, handler := range handlers {
		if handler.Chain != nil {
			chainedHandlers = append(chainedHandlers, handler)
		} else {
			r := runHandlerSafely(handler,event)
			if !answered && !r.Empty {
				reply = r
				answered = true
				log.Printf("event %s was answered with reply %v", event.Name, reply)
			}
		}
	}

	if len(chainedHandlers) == 0 {
		return reply
	}

	for _, handler := range chainedHandlers {
		runChainHandlerSafely(handler, &event)
	}
	return reply
}

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

func NewEvents(config Config) Events {
	return make(Events)
}

func (e Events) SetPriority(eventName, pluginName string, options Options) error{
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
func (e Events)sortByPriority(eventName string){
	sort.Slice(e[eventName], func(i, j int) bool {
		return e[eventName][i].Priority > e[eventName][j].Priority
	})
}