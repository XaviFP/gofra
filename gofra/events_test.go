package gofra

import (
	"testing"
)

func TestNewEvents(t *testing.T) {
	events := NewEvents(Config{})
	if len(events) != 0 {
		t.Error(`NewEvents returns a non empty Events object`)
	}
}

var ran bool
func setRan(b bool) {
	ran = b
}
func TestPublishSubscribeEvent(t *testing.T) {
	events := NewEvents(Config{})
	
	events.Subscribe(
		"addedEventListener",
		"testPlugin",
		func(e Event, _ *Event) (Reply, Event){
			setRan(true)
			return Reply{}, e
		},
		Options{},
	)
	if !ran {
		t.Error(`Event subscribed didn't run`)
	}
}

func exampleHandler(e Event, _ *Event) (Reply, Event){
	return Reply{}, e
}
func TestSetpriority(t *testing.T) {
	events := NewEvents(Config{})
	
	events.Subscribe(
		"addedEventListener",
		"testPlugin1",
		exampleHandler,
		Options{Priority: 1},
	)
	events.Subscribe(
		"addedEventListener",
		"testPlugin2",
		exampleHandler,
		Options{Priority: 2},
	)
	if events["addedEventListener"][0].PluginName != "testPlugin2"{
		t.Error(`Event handlers are no sorted correctly`)
	}
	events.SetPriority("addedEventListener", "testPlugin1", Options{Priority: 3})

	if events["addedEventListener"][0].PluginName != "testPlugin1"{
		t.Error(`SetPriority does not sort correctly`)
	}
}

