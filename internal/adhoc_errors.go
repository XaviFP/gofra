package gofra

import (
	"encoding/xml"

	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
)

// XEP-0050 error conditions
const (
	ErrTypeMalformedAction = "malformed-action"
	ErrTypeBadAction       = "bad-action"
	ErrTypeBadLocale       = "bad-locale"
	ErrTypeBadPayload      = "bad-payload"
	ErrTypeBadSessionID    = "bad-sessionid"
	ErrTypeSessionExpired  = "session-expired"
)

// StanzaError represents an XMPP stanza error.
type StanzaError struct {
	XMLName      xml.Name `xml:"error"`
	Type         string   `xml:"type,attr"`
	Code         string   `xml:"code,attr,omitempty"`
	Condition    string   // Will be marshaled as child element
	CommandError string   // Command-specific error
	Text         string   `xml:"urn:ietf:params:xml:ns:xmpp-stanzas text,omitempty"`
}

// ErrorIQ creates an error IQ response.
func ErrorIQ(iq IQ, errType, condition, cmdError, text string) IQ {
	return IQ{
		IQ: stanza.IQ{
			ID:   iq.ID,
			Type: stanza.ErrorIQ,
			To:   iq.From,
			From: iq.To,
		},
	}
}

// IQErrorResponse represents a full IQ error response.
type IQErrorResponse struct {
	XMLName xml.Name `xml:"iq"`
	ID      string   `xml:"id,attr"`
	Type    string   `xml:"type,attr"`
	To      jid.JID  `xml:"to,attr"`
	From    jid.JID  `xml:"from,attr,omitempty"`
	Command *Command `xml:"command,omitempty"`
	Error   *IQError `xml:"error"`
}

// IQError represents the error element in an IQ response.
type IQError struct {
	Type       string `xml:"type,attr"`
	Code       string `xml:"code,attr,omitempty"`
	Conditions []ErrorCondition
}

// MarshalXML implements custom XML marshaling for IQError.
func (e *IQError) MarshalXML(enc *xml.Encoder, start xml.StartElement) error {
	start.Name = xml.Name{Local: "error"}
	start.Attr = []xml.Attr{
		{Name: xml.Name{Local: "type"}, Value: e.Type},
	}
	if e.Code != "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "code"}, Value: e.Code})
	}

	if err := enc.EncodeToken(start); err != nil {
		return err
	}

	for _, cond := range e.Conditions {
		if err := enc.EncodeElement(struct{}{}, xml.StartElement{
			Name: xml.Name{Space: cond.NS, Local: cond.Name},
		}); err != nil {
			return err
		}
	}

	return enc.EncodeToken(start.End())
}

// ErrorCondition represents an error condition element.
type ErrorCondition struct {
	NS   string
	Name string
}

const (
	nsXMPPStanzas = "urn:ietf:params:xml:ns:xmpp-stanzas"
	nsCommands    = "http://jabber.org/protocol/commands"
)

// NewBadRequestError creates a bad-request error response.
func NewBadRequestError(iq IQ, cmdError string) *IQErrorResponse {
	conditions := []ErrorCondition{
		{NS: nsXMPPStanzas, Name: "bad-request"},
	}
	if cmdError != "" {
		conditions = append(conditions, ErrorCondition{NS: nsCommands, Name: cmdError})
	}
	return &IQErrorResponse{
		ID:   iq.ID,
		Type: "error",
		To:   iq.From,
		From: iq.To,
		Error: &IQError{
			Type:       "modify",
			Code:       "400",
			Conditions: conditions,
		},
	}
}

// NewMalformedActionError creates a malformed-action error.
func NewMalformedActionError(iq IQ) *IQErrorResponse {
	return NewBadRequestError(iq, ErrTypeMalformedAction)
}

// NewBadActionError creates a bad-action error.
func NewBadActionError(iq IQ) *IQErrorResponse {
	return NewBadRequestError(iq, ErrTypeBadAction)
}

// NewBadPayloadError creates a bad-payload error.
func NewBadPayloadError(iq IQ) *IQErrorResponse {
	return NewBadRequestError(iq, ErrTypeBadPayload)
}

// NewBadSessionIDError creates a bad-sessionid error.
func NewBadSessionIDError(iq IQ) *IQErrorResponse {
	return NewBadRequestError(iq, ErrTypeBadSessionID)
}

// NewSessionExpiredError creates a session-expired error.
func NewSessionExpiredError(iq IQ) *IQErrorResponse {
	return &IQErrorResponse{
		ID:   iq.ID,
		Type: "error",
		To:   iq.From,
		From: iq.To,
		Error: &IQError{
			Type: "cancel",
			Code: "405",
			Conditions: []ErrorCondition{
				{NS: nsXMPPStanzas, Name: "not-allowed"},
				{NS: nsCommands, Name: ErrTypeSessionExpired},
			},
		},
	}
}

// NewForbiddenError creates a forbidden error response.
func NewForbiddenError(iq IQ) *IQErrorResponse {
	return &IQErrorResponse{
		ID:   iq.ID,
		Type: "error",
		To:   iq.From,
		From: iq.To,
		Error: &IQError{
			Type: "cancel",
			Code: "403",
			Conditions: []ErrorCondition{
				{NS: nsXMPPStanzas, Name: "forbidden"},
			},
		},
	}
}

// NewItemNotFoundError creates an item-not-found error response.
func NewItemNotFoundError(iq IQ) *IQErrorResponse {
	return &IQErrorResponse{
		ID:   iq.ID,
		Type: "error",
		To:   iq.From,
		From: iq.To,
		Error: &IQError{
			Type: "cancel",
			Code: "404",
			Conditions: []ErrorCondition{
				{NS: nsXMPPStanzas, Name: "item-not-found"},
			},
		},
	}
}

// NewFeatureNotImplementedError creates a feature-not-implemented error.
func NewFeatureNotImplementedError(iq IQ) *IQErrorResponse {
	return &IQErrorResponse{
		ID:   iq.ID,
		Type: "error",
		To:   iq.From,
		From: iq.To,
		Error: &IQError{
			Type: "cancel",
			Code: "501",
			Conditions: []ErrorCondition{
				{NS: nsXMPPStanzas, Name: "feature-not-implemented"},
			},
		},
	}
}
