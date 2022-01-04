package gofra

import (
	"encoding/xml"
	"io"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/stanza"
)

type MessageBody struct {
	stanza.Message
	Body string `xml:"body"`
}

type stanzaHandler struct {
	logger  Logger
	publish func(e Event)
}

func (h stanzaHandler) HandleMessage(msg stanza.Message, t xmlstream.TokenReadEncoder) error {
	h.logger.Debug.Printf("Message received: %v", msg)

	d := xml.NewTokenDecoder(t)
	msgStruct := MessageBody{}
	err := d.Decode(&msgStruct)

	if err != nil && err != io.EOF {
		h.logger.Error.Printf("Error decoding message: %q", err)
		return nil
	}

	if msgStruct.Body == "" || msgStruct.Type != stanza.ChatMessage {
		h.logger.Debug.Printf("Message received has no body")
	}

	h.logger.Debug.Printf("Message received: %v, with body: %q", msgStruct, msgStruct.Body)
	e := Event{
		Name:    "messageReceived",
		Payload: make(map[string]interface{}),
	}
	e.SetStanza(msgStruct)

	defer func() {
		go h.publish(e)
	}()

	return nil
}

// Prevent same presence to be handled more than once.
// Using an empty xml.Name in the handler registration creates a wildcard that makes the handler run for every inner element in the stanza
var lastP stanza.Presence

func isLastPresence(p stanza.Presence) bool {
	if lastP.From.String() == p.From.String() && lastP.To.String() == p.To.String() && lastP.Type == p.Type {
		return true
	}

	return false
}

func (h stanzaHandler) HandlePresence(p stanza.Presence, t xmlstream.TokenReadEncoder) error {
	if isLastPresence(p) {
		return nil
	}

	lastP = p
	h.logger.Debug.Printf("Presence received: %v", p)

	e := Event{
		Name:    "presenceReceived",
		Payload: make(map[string]interface{}),
	}
	//TODO use presence extended fields struct as decode receiver like MessageBody{}
	e.SetStanza(p)

	defer func() {
		go h.publish(e)
	}()

	return nil
}

//TODO add proper IQ handling
func (h stanzaHandler) HandleIQ(iq stanza.IQ, t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
	h.logger.Debug.Printf("Presence received: %v", iq)

	e := Event{
		Name:    "iqReceived",
		Payload: make(map[string]interface{}),
	}
	e.SetStanza(iq)

	defer func() {
		go h.publish(e)
	}()

	return nil
}
