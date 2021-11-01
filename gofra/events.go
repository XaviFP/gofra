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
	var acc Event
	Clone(&event, &acc)
	for _, handler := range chainedHandlers {
		_, _ = handler.Handler(event, &acc)
	}
	return reply
}

func NewEvents(config Config) Events {
	return make(Events)
}

func (e Events) SetPriority(eventName, pluginName string, options Options) error{
	var priorityChanged bool
	var pluginFound bool
	handlers := e[eventName]
	if handlers == nil {return fmt.Errorf("event %s not found", eventName)}
	for _, element := range handlers {
		if element.PluginName == pluginName {
			pluginFound = true
			if element.Priority != options.Priority {
				element.Priority = options.Priority
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
	sortByPriority(handlers)
	return nil
}

// Sorts handlers in descending priority order
func sortByPriority(handlers []EventHandler){
	sort.Slice(handlers, func(i, j int) bool {
		return handlers[i].Priority > handlers[j].Priority
	})
}