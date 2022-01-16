package gofra

import (
	"testing"

	"github.com/stretchr/testify/mock"
)


func TeststanzaHandler_HandleMessage(t *testing.T){
	// publish := func(e Event) {
	// 	expected := Event{

	// 	}
	// 	assert.Equal(t, expected, e)
	// }
	// s := stanzaHandler{
	// 	logger: NewLogger(true),// mock logger
	// 	publish: publish,
	// }

	//s.HandleMessage(stanza.Message{}, xmlstream)
}

type tokenReadEncoderMock struct {
	mock.Mock
}

