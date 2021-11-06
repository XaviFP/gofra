package gofra

import (
	"testing"
)

func TestReplySetAnswer(t *testing.T) {
	r := Reply{}
	r.SetAnswer("testStr")
	if r.Payload["answer"] != "testStr" {
		t.Error(`Answer was not set by SetAnswer`)
	}
}

func TestReplyGetAnswer(t *testing.T) {
	r := Reply{}
	answer := r.GetAnswer()
	_, exists := r.Payload["answer"]
	if  exists {
		t.Error(`Answer field in payload should not exist`)
	}
	if answer != "" {
		t.Error(`Answer should be empty string`)
	}
	r.Payload["answer"] = 25
	answer = r.GetAnswer()
	if answer != "" {
		t.Error(`Answer field in payload should be of type string`)
	}
	r.Payload["answer"] = "testStr"
	answer = r.GetAnswer()
	if answer != "testStr" {
		t.Error(`Answer should match answer string`)
	}
}

func TestReplySetNoHandlers(t *testing.T) {
	r := Reply{}
	r.SetNoHandlers(true)
	if r.Payload["noHandlers"] != true {
		t.Error(`noHandlers was not set by SetNoHandlers`)
	}
}

func TestReplyGetNoHandlers(t *testing.T) {
	r := Reply{}
	noHandlers := r.GetNoHandlers()
	_, exists := r.Payload["noHandlers"]
	if  exists {
		t.Error(`noHandlers field in payload should not exist`)
	}
	if noHandlers != false {
		t.Error(`noHandlers should be false`)
	}
	r.Payload["noHandlers"] = 25
	noHandlers = r.GetNoHandlers()
	if noHandlers != false {
		t.Error(`noHandlers field in payload should be of type bool`)
	}
	r.Payload["noHandlers"] = true
	noHandlers = r.GetNoHandlers()
	if noHandlers != true {
		t.Error(`noHandlers should match noHandlers value`)
	}
}