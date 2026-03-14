package gofra

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"

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

// IQ represents an XMPP IQ stanza with optional query or command children.
type IQ struct {
	stanza.IQ
	Query   *Query   `xml:"query,omitempty"`
	Command *Command `xml:"command,omitempty"`
}

// Reply creates a result IQ response.
func (iq IQ) Reply() *IQReply {
	return &IQReply{
		XMLName: xml.Name{Space: "jabber:client", Local: "iq"},
		ID:      iq.ID,
		To:      iq.From,
		From:    iq.To,
		Type:    stanza.ResultIQ,
	}
}

// IQReply is the structure for sending IQ responses.
type IQReply struct {
	XMLName xml.Name      `xml:"iq"`
	ID      string        `xml:"id,attr"`
	To      jid.JID       `xml:"to,attr"`
	From    jid.JID       `xml:"from,attr"`
	Type    stanza.IQType `xml:"type,attr"`
	Query   *Query        `xml:"query,omitempty"`
	Command *Command      `xml:"command,omitempty"`
	Error   *IQError      `xml:"error,omitempty"`
}

// Query represents a disco#info or disco#items query.
type Query struct {
	XMLNS    string    `xml:"xmlns,attr"`
	Node     string    `xml:"node,attr,omitempty"`
	Name     string    `xml:"name,omitempty"`
	Version  string    `xml:"version,omitempty"`
	Identity *Identity `xml:"identity,omitempty"`
	Features []Feature `xml:"feature,omitempty"`
	Items    []Item    `xml:"item,omitempty"`
}

// Identity represents a disco#info identity.
type Identity struct {
	Name     string `xml:"name,attr"`
	Category string `xml:"category,attr"`
	Type     string `xml:"type,attr"`
}

// Feature represents a disco#info feature.
type Feature struct {
	Var string `xml:"var,attr"`
}

// Item represents a disco#items item.
type Item struct {
	JID  jid.JID `xml:"jid,attr"`
	Node string  `xml:"node,attr,omitempty"`
	Name string  `xml:"name,attr,omitempty"`
}

// Command represents an XEP-0050 ad-hoc command element.
type Command struct {
	XMLName   xml.Name `xml:"http://jabber.org/protocol/commands command"`
	Node      string   `xml:"node,attr"`
	SessionID string   `xml:"sessionid,attr,omitempty"`
	Action    string   `xml:"action,attr,omitempty"`
	Status    string   `xml:"status,attr,omitempty"`
	Actions   *Actions `xml:"actions,omitempty"`
	Notes     []Note   `xml:"note,omitempty"`
	XData     *XData   `xml:"x,omitempty"`
}

// Actions represents the available actions in a command.
type Actions struct {
	Execute  string `xml:"execute,attr,omitempty"`
	Prev     *struct{} `xml:"prev,omitempty"`
	Next     *struct{} `xml:"next,omitempty"`
	Complete *struct{} `xml:"complete,omitempty"`
}

// Note represents a note in a command response.
type Note struct {
	Type  string `xml:"type,attr,omitempty"`
	Value string `xml:",chardata"`
}

// XData represents an XEP-0004 Data Form.
type XData struct {
	XMLName      xml.Name     `xml:"jabber:x:data x"`
	Type         string       `xml:"type,attr"`
	Title        string       `xml:"title,omitempty"`
	Instructions string       `xml:"instructions,omitempty"`
	Fields       []XDataField `xml:"field,omitempty"`
}

// XDataField represents a field in a data form.
type XDataField struct {
	Var     string        `xml:"var,attr,omitempty"`
	Type    string        `xml:"type,attr,omitempty"`
	Label   string        `xml:"label,attr,omitempty"`
	Values  []string      `xml:"value,omitempty"`
	Options []XDataOption `xml:"option,omitempty"`
}

// Value returns the first value or empty string for backwards compatibility.
func (f XDataField) Value() string {
	if len(f.Values) > 0 {
		return f.Values[0]
	}
	return ""
}

// XDataOption represents an option in a list field.
type XDataOption struct {
	Label string `xml:"label,attr,omitempty"`
	Value string `xml:"value"`
}

func (h stanzaHandler) HandleIQ(iq stanza.IQ, t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
	h.logger.Debug(fmt.Sprintf("IQ received: %s %v", start.Name.Local, start.Attr))

	d := xml.NewTokenDecoder(t)
	gIQ := IQ{IQ: iq}

	switch start.Name.Local {
	case "query":
		q := Query{}
		// Extract attributes from start element
		for _, attr := range start.Attr {
			switch attr.Name.Local {
			case "xmlns":
				q.XMLNS = attr.Value
			case "node":
				q.Node = attr.Value
			}
		}
		if q.XMLNS == "" {
			q.XMLNS = start.Name.Space
		}
		// Try to decode nested elements, but don't fail on empty queries
		if err := d.DecodeElement(&q, start); err != nil && err != io.EOF {
			// Ignore "unexpected end element" for empty self-closing tags
			if !strings.Contains(err.Error(), "unexpected end element") {
				h.logger.Error(fmt.Sprintf("Error decoding Query: %v", err))
				return nil
			}
		}
		gIQ.Query = &q

	case "command":
		c := Command{}
		// Extract attributes from start element
		for _, attr := range start.Attr {
			switch attr.Name.Local {
			case "node":
				c.Node = attr.Value
			case "action":
				c.Action = attr.Value
			case "sessionid":
				c.SessionID = attr.Value
			}
		}
		// Try to decode nested elements, but don't fail on empty commands
		if err := d.DecodeElement(&c, start); err != nil && err != io.EOF {
			// Ignore "unexpected end element" for empty self-closing tags
			if !strings.Contains(err.Error(), "unexpected end element") {
				h.logger.Error(fmt.Sprintf("Error decoding Command: %v", err))
				return nil
			}
		}
		gIQ.Command = &c

	default:
		h.logger.Debug(fmt.Sprintf("Unknown IQ child element: %s", start.Name.Local))
	}

	e := Event{
		Name:    "iqReceived",
		Payload: make(map[string]interface{}),
	}
	e.SetStanza(gIQ)
	e.SetIQEncoder(t) // Pass the encoder so plugins can write responses directly

	h.logger.Debug(fmt.Sprintf("Publishing iqReceived event for IQ id=%s type=%s", iq.ID, iq.Type))

	// Publish synchronously so handlers can respond before mux sends service-unavailable
	h.publish(e)

	// If a handler marked the IQ as handled, it wrote to the encoder already
	// and we just return nil. If not handled, mux sends service-unavailable.

	return nil
}
