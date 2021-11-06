package gofra

import (
	"fmt"
	"sort"
)

type Events map[string][]EventHandler

func (e Events) Subscribe(eventName, pluginName string, handler Handler, op Options){
	if e[eventName] == nil {
		e[eventName] = []EventHandler{}
	}
	e[eventName] = append(
		e[eventName],
		EventHandler{
			Handler: handler,
			Priority: op.Priority,
			PluginName: pluginName,
			Chain: op.Chain,
		},
	)
	e.sortByPriority(eventName)
	event := Event{
		Name:"addedEventListener",
		Payload: map[string]interface{}{
			"event": eventName,
			"plugin": pluginName,
			"chained": op.Chain,
			"priority": op.Priority,
		},
	}
	e.Publish(event)
}

func (e Events) Publish(event Event) Reply{
	handlers, exist := e[event.Name]
	var reply Reply
	if !exist {
		//No handlers for event
		r := Reply{Payload: make(map[string]interface{}), Ok: false, Empty: false}
		r.SetNoHandlers(true)
		return r
	}
	answered := false
	chainedHandlers := []EventHandler{}
	for _, handler := range handlers {
		if handler.Chain {
			chainedHandlers = append(chainedHandlers, handler)
		} else {
			r, _ := handler.Handler(event, nil)
			if !answered && !r.Empty && r.Ok {
				reply = r
				answered = true
			}
		}
	}

	if len(chainedHandlers) < 1 {
		return reply
	}

	for _, handler := range chainedHandlers {
		_, _ = handler.Handler(event, &event)
	}
	return reply
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