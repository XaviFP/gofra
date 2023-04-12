package gofra

import (
	"encoding/xml"
	"fmt"
	"io"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

type MessageBody struct {
	stanza.Message
	Body string `xml:"body"`
}

func (mb MessageBody) Reply(body string) MessageBody {
	reply := mb
	reply.Body = body

	if mb.Type == stanza.GroupChatMessage {
		reply.To, reply.From = mb.From.Bare(), jid.MustParse(
			fmt.Sprintf(
				"%s/%s",
				mb.From.Bare().String(),           // JID
				mucNicks[mb.From.Bare().String()], // Nickname
			),
		)

		return reply
	}

	reply.To, reply.From = mb.From, mb.To

	return reply
}

type stanzaHandler struct {
	logger  Logger
	publish func(e Event)
}

func (h stanzaHandler) HandleMessage(msg stanza.Message, t xmlstream.TokenReadEncoder) error {
	h.logger.Debug(fmt.Sprintf("Message received: %v", msg))

	d := xml.NewTokenDecoder(t)
	mb := MessageBody{}
	err := d.Decode(&mb)

	if err != nil && err != io.EOF {
		h.logger.Error(fmt.Sprintf("Error decoding message: %q", err))

		return nil
	}

	if mb.Body == "" || mb.Type != stanza.ChatMessage {
		h.logger.Debug("Message received has no body")
	}

	h.logger.Debug(fmt.Sprintf("Message received: %v, with body: %q", mb, mb.Body))

	e := Event{
		Name:    "messageReceived",
		Payload: make(map[string]interface{}),
		MB:      mb,
	}

	e.SetStanza(mb)

	defer func() {
		go h.publish(e)
	}()

	return nil
}

// Prevent same presence to be handled more than once.
// Using an empty xml.Name in the handler registration creates a wildcard
// making the handler run for every inner element in the stanza
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
	h.logger.Debug(fmt.Sprintf("Presence received: %v", p))

	e := Event{
		Name:    "presenceReceived",
		Payload: make(map[string]interface{}),
	}

	e.SetStanza(p)

	defer func() {
		go h.publish(e)
	}()

	return nil
}

func (h stanzaHandler) HandleIQ(iq stanza.IQ, t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
	h.logger.Debug(fmt.Sprintf("IQ received: %v", iq))

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
