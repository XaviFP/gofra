// Package main provides the adhoc plugin for Gofra.
// It implements XEP-0050 Ad-Hoc Commands support.
package main

import (
	"fmt"
	"strings"

	gofra "github.com/XaviFP/gofra/internal"
	"mellium.im/xmpp/stanza"
)

// Plugin is the exported plugin instance.
var Plugin plugin

var g *gofra.Gofra
var registry *gofra.CommandRegistry

type plugin struct{}

func (p plugin) Name() string {
	return "adhoc"
}

func (p plugin) Description() string {
	return "Provides XEP-0050 Ad-Hoc Commands support"
}

func (p plugin) Help() string {
	return "adhoc is a meta-plugin that enables ad-hoc command support for other plugins"
}

func (p plugin) Init(config gofra.Config, api *gofra.Gofra) {
	g = api
	registry = gofra.NewCommandRegistry()

	// Subscribe to IQ events
	g.Subscribe("iqReceived", p.Name(), handleIQ, 1)

	// Subscribe to command registration events from other plugins
	g.Subscribe("adhoc/register", p.Name(), handleRegister, 0)
	g.Subscribe("adhoc/unregister", p.Name(), handleUnregister, 0)
}

// handleRegister handles command registration from other plugins.
func handleRegister(e gofra.Event) *gofra.Reply {
	cmd, ok := e.Payload["command"].(*gofra.AdHocCommand)
	if !ok {
		g.Logger.Error("adhoc: invalid command registration payload")
		return nil
	}

	registry.Register(cmd)
	g.Logger.Info(fmt.Sprintf("adhoc: registered command '%s'", cmd.Node))

	return nil
}

// handleUnregister handles command unregistration.
func handleUnregister(e gofra.Event) *gofra.Reply {
	node, ok := e.Payload["node"].(string)
	if !ok {
		g.Logger.Error("adhoc: invalid unregister payload")
		return nil
	}

	registry.Unregister(node)
	g.Logger.Info(fmt.Sprintf("adhoc: unregistered command '%s'", node))
	return nil
}

// handleIQ routes incoming IQ stanzas to appropriate handlers.
func handleIQ(e gofra.Event) *gofra.Reply {
	g.Logger.Debug("adhoc: handleIQ called")
	iq, err := e.GetIQ()
	if err != nil {
		g.Logger.Error(fmt.Sprintf("adhoc: error getting IQ: %v", err))
		return nil
	}

	g.Logger.Debug(fmt.Sprintf("adhoc: IQ type=%s, Query=%v, Command=%v", iq.Type, iq.Query, iq.Command))

	var handled bool
	switch iq.Type {
	case stanza.GetIQ:
		handled = handleIQGet(e, iq)
	case stanza.SetIQ:
		handled = handleIQSet(e, iq)
	}

	if handled {
		e.MarkHandled()
	}

	return nil
}

// handleIQGet handles IQ get requests (disco queries).
// Returns true if the IQ was handled.
func handleIQGet(e gofra.Event, iq gofra.IQ) bool {
	if iq.Query == nil {
		return false
	}
	switch iq.Query.XMLNS {
	case "jabber:iq:version":
		return handleVersion(e, iq)

	case "http://jabber.org/protocol/disco#info":
		return handleDiscoInfo(e, iq)

	case "http://jabber.org/protocol/disco#items":
		return handleDiscoItems(e, iq)
	}

	return false
}

// handleVersion responds to version queries.
func handleVersion(e gofra.Event, iq gofra.IQ) bool {
	reply := iq.Reply()
	reply.Query = &gofra.Query{
		XMLNS:   "jabber:iq:version",
		Name:    "Gofra",
		Version: "1.0.0",
	}

	if err := g.SendIQResponse(e, reply); err != nil {
		g.Logger.Error(fmt.Sprintf("adhoc: error sending version reply: %v", err))
	}
	return true
}

// handleDiscoInfo responds to disco#info queries.
func handleDiscoInfo(e gofra.Event, iq gofra.IQ) bool {
	reply := iq.Reply()
	reply.Query = &gofra.Query{
		XMLNS: "http://jabber.org/protocol/disco#info",
		Node:  iq.Query.Node,
	}

	if iq.Query.Node == "" {
		// Service discovery: advertise commands support
		reply.Query.Features = []gofra.Feature{
			{Var: gofra.CommandsNS},
		}
	} else {
		// Info about a specific command
		cmd, ok := registry.GetCommand(iq.Query.Node)
		if !ok {
			if err := g.SendIQResponse(e, gofra.NewItemNotFoundError(iq)); err != nil {
				g.Logger.Error(fmt.Sprintf("adhoc: error sending error: %v", err))
			}
			return true
		}

		reply.Query.Identity = &gofra.Identity{
			Name:     cmd.Name,
			Category: "automation",
			Type:     "command-node",
		}
		reply.Query.Features = []gofra.Feature{
			{Var: gofra.CommandsNS},
			{Var: "jabber:x:data"},
		}
	}

	if err := g.SendIQResponse(e, reply); err != nil {
		g.Logger.Error(fmt.Sprintf("adhoc: error sending disco#info reply: %v", err))
	}
	return true
}

// handleDiscoItems responds to disco#items queries.
func handleDiscoItems(e gofra.Event, iq gofra.IQ) bool {
	if iq.Query.Node != gofra.CommandsNS {
		return false
	}

	reply := iq.Reply()
	reply.Query = &gofra.Query{
		XMLNS: "http://jabber.org/protocol/disco#items",
		Node:  gofra.CommandsNS,
	}

	requesterJID := iq.From.Bare().String()
	for _, cmd := range registry.ListCommandsForJID(requesterJID) {
		reply.Query.Items = append(reply.Query.Items, gofra.Item{
			JID:  iq.To,
			Node: cmd.Node,
			Name: cmd.Name,
		})
	}

	if err := g.SendIQResponse(e, reply); err != nil {
		g.Logger.Error(fmt.Sprintf("adhoc: error sending disco#items reply: %v", err))
	}
	return true
}

// handleIQSet handles IQ set requests (command execution).
func handleIQSet(e gofra.Event, iq gofra.IQ) bool {
	if iq.Command == nil {
		return false
	}

	action := gofra.CommandAction(iq.Command.Action)
	if action == "" {
		action = gofra.ActionExecute
	}

	// Validate action
	if !isValidAction(action) {
		if err := g.SendIQResponse(e, gofra.NewMalformedActionError(iq)); err != nil {
			g.Logger.Error(fmt.Sprintf("adhoc: error sending error: %v", err))
		}
		return true
	}

	// Handle cancel action
	if action == gofra.ActionCancel {
		return handleCancel(e, iq)
	}

	// Get the command
	cmd, ok := registry.GetCommand(iq.Command.Node)
	if !ok {
		if err := g.SendIQResponse(e, gofra.NewItemNotFoundError(iq)); err != nil {
			g.Logger.Error(fmt.Sprintf("adhoc: error sending error: %v", err))
		}
		return true
	}

	// Get or create session
	var session *gofra.CommandSession
	if iq.Command.SessionID != "" {
		session, ok = registry.GetSession(iq.Command.SessionID)
		if !ok {
			if err := g.SendIQResponse(e, gofra.NewSessionExpiredError(iq)); err != nil {
				g.Logger.Error(fmt.Sprintf("adhoc: error sending error: %v", err))
			}
			return true
		}

		// Validate session belongs to this requester and command
		if session.Requester != iq.From.Bare().String() || session.Node != iq.Command.Node {
			if err := g.SendIQResponse(e, gofra.NewBadSessionIDError(iq)); err != nil {
				g.Logger.Error(fmt.Sprintf("adhoc: error sending error: %v", err))
			}
			return true
		}

		registry.RefreshSession(session.ID)
	} else {
		// New session for execute action
		if action != gofra.ActionExecute {
			if err := g.SendIQResponse(e, gofra.NewBadSessionIDError(iq)); err != nil {
				g.Logger.Error(fmt.Sprintf("adhoc: error sending error: %v", err))
			}
			return true
		}
		session = registry.CreateSession(iq.Command.Node, iq.From.Bare().String())
	}

	// Parse form data if present
	// For single-value fields, store as string
	// For multi-value fields (list-multi), store as []string in session
	formData := make(map[string]string)
	if iq.Command.XData != nil {
		g.Logger.Debug(fmt.Sprintf("adhoc: parsing XData with %d fields", len(iq.Command.XData.Fields)))
		for _, field := range iq.Command.XData.Fields {
			g.Logger.Debug(fmt.Sprintf("adhoc: field var=%s type=%s values=%v", field.Var, field.Type, field.Values))
			if len(field.Values) > 1 {
				// Multi-value field - store in session directly as slice
				session.Set(field.Var+"_multi", field.Values)
				// Also store joined version for backwards compat
				formData[field.Var] = strings.Join(field.Values, "\n")
			} else {
				formData[field.Var] = field.Value()
			}
		}
	} else {
		g.Logger.Debug("adhoc: no XData in command")
	}

	// Execute the command handler
	resp, err := cmd.Handler(session, action, formData)
	if err != nil {
		g.Logger.Error(fmt.Sprintf("adhoc: command handler error: %v", err))
		if err := g.SendIQResponse(e, gofra.NewBadRequestError(iq, gofra.ErrTypeBadPayload)); err != nil {
			g.Logger.Error(fmt.Sprintf("adhoc: error sending error: %v", err))
		}
		registry.DeleteSession(session.ID)
		return true
	}

	// Build response
	reply := iq.Reply()
	reply.Command = &gofra.Command{
		Node:      iq.Command.Node,
		SessionID: session.ID,
	}

	if resp.IsComplete || resp.Status == gofra.StatusCompleted {
		reply.Command.Status = string(gofra.StatusCompleted)
		registry.DeleteSession(session.ID)
	} else if resp.Status == gofra.StatusCanceled {
		reply.Command.Status = string(gofra.StatusCanceled)
		registry.DeleteSession(session.ID)
	} else {
		reply.Command.Status = string(gofra.StatusExecuting)
	}

	reply.Command.Actions = resp.Actions
	reply.Command.Notes = resp.Notes
	reply.Command.XData = resp.Form

	if err := g.SendIQResponse(e, reply); err != nil {
		g.Logger.Error(fmt.Sprintf("adhoc: error sending command reply: %v", err))
	}

	return true
}

// handleCancel handles command cancellation.
func handleCancel(e gofra.Event, iq gofra.IQ) bool {
	if iq.Command.SessionID == "" {
		if err := g.SendIQResponse(e, gofra.NewBadSessionIDError(iq)); err != nil {
			g.Logger.Error(fmt.Sprintf("adhoc: error sending error: %v", err))
		}
		return true
	}

	session, ok := registry.GetSession(iq.Command.SessionID)
	if !ok {
		if err := g.SendIQResponse(e, gofra.NewSessionExpiredError(iq)); err != nil {
			g.Logger.Error(fmt.Sprintf("adhoc: error sending error: %v", err))
		}
		return true
	}

	// Validate session ownership
	if session.Requester != iq.From.Bare().String() {
		if err := g.SendIQResponse(e, gofra.NewForbiddenError(iq)); err != nil {
			g.Logger.Error(fmt.Sprintf("adhoc: error sending error: %v", err))
		}
		return true
	}

	registry.DeleteSession(session.ID)

	reply := iq.Reply()
	reply.Command = &gofra.Command{
		Node:      iq.Command.Node,
		SessionID: session.ID,
		Status:    string(gofra.StatusCanceled),
	}

	if err := g.SendIQResponse(e, reply); err != nil {
		g.Logger.Error(fmt.Sprintf("adhoc: error sending cancel reply: %v", err))
	}

	return true
}

// isValidAction checks if an action is valid.
func isValidAction(action gofra.CommandAction) bool {
	switch action {
	case gofra.ActionExecute, gofra.ActionCancel, gofra.ActionPrev, gofra.ActionNext, gofra.ActionComplete:
		return true
	}
	return false
}
