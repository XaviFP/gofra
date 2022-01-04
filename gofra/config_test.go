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
	if exists {
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
