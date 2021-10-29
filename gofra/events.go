package gofra

import (
	"errors"
	"sort"
)

type Events map[string][]EventHandler

func (e Events) Subscribe(eventName, pluginName string, handler Handler, options Options){
	if e[eventName] == nil {
		e[eventName] = []EventHandler{}
	}
	e[eventName] = append(
		e[eventName],
		EventHandler{Handler: handler, Priority: options.Priority},
	)
	// Find out smart/elegant way to name and find events in the code
	e.Publish(Event{Name:"addedEventListener", Payload: nil, Stanza: nil})
}

func (e Events) Publish(event Event) Reply{
	// Find out event chainning, unhandled events and whatnot
	handlers, exist := e[event.Name]
	var reply Reply
	if !exist {
		//No handlers for event
		r := Reply{Reply: make(map[string]interface{}), Ok: false, Empty: false}
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
	//TODO Deep copy event, following code is a placeholder
	acc := event
	for _, handler := range chainedHandlers {
		_, _ = handler.Handler(event, &acc)
	}
	return reply
}

func NewEvents(config Config) Events {
	return make(Events)
}

func (e Events) SetPriority(eventName, pluginName string, options Options) error{
	priorityChanged := false
	pluginFound := false
	handlers := e[eventName]
	if handlers == nil {return errors.New("event {} not found")}
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
		// "no '<eventName>' handler found for plugin '<pluginName>'"
		return errors.New("no {} handler found for plugin {}")
	}
	// Following code might not necessarily be an error per se
	if !priorityChanged {
		// "'<pluginName>' plugin's '<eventName>' handler had same priority previously"
		//return errors.New("{} plugin's {} handler had same priority previously")
		return nil
	}
	sortByPriority(handlers)
	return nil
}

func sortByPriority(handlers []EventHandler){
	// Sort handlers in descending priority
	sort.Slice(handlers, func(i, j int) bool {
		return handlers[i].Priority > handlers[j].Priority
	})
}