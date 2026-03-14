// Package main provides an example ad-hoc command plugin for Gofra.
// It demonstrates how to create multi-stage ad-hoc commands using the adhoc plugin.
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

type plugin struct{}

func (p plugin) Name() string {
	return "greeting"
}

func (p plugin) Description() string {
	return "Provides a multi-stage greeting ad-hoc command example"
}

func (p plugin) Help() string {
	return "Use the ad-hoc commands interface to send customized greetings"
}

func (p plugin) Init(config gofra.Config, api *gofra.Gofra) {
	g = api

	// Register the greeting command via the adhoc plugin
	g.Publish(gofra.Event{
		Name: "adhoc/register",
		Payload: map[string]interface{}{
			"command": &gofra.AdHocCommand{
				Node:    "greeting",
				Name:    "Send a Greeting",
				Handler: handleGreeting,
			},
		},
	})

	// Register a simple single-stage command
	g.Publish(gofra.Event{
		Name: "adhoc/register",
		Payload: map[string]interface{}{
			"command": &gofra.AdHocCommand{
				Node:    "ping",
				Name:    "Ping",
				Handler: handlePing,
			},
		},
	})
}

// handlePing is a simple single-stage command.
func handlePing(session *gofra.CommandSession, action gofra.CommandAction, formData map[string]string) (*gofra.CommandResponse, error) {
	return &gofra.CommandResponse{
		Status:     gofra.StatusCompleted,
		IsComplete: true,
		Notes:      []gofra.Note{gofra.NewInfoNote("Pong!")},
	}, nil
}

// handleGreeting is a multi-stage command handler.
func handleGreeting(session *gofra.CommandSession, action gofra.CommandAction, formData map[string]string) (*gofra.CommandResponse, error) {
	// Handle cancel
	if action == gofra.ActionCancel {
		return &gofra.CommandResponse{
			Status:     gofra.StatusCanceled,
			IsComplete: true,
		}, nil
	}

	// Handle prev - go back by clearing most recent data
	if action == gofra.ActionPrev {
		return handleGreetingPrev(session)
	}

	// Save form data
	if greetType, ok := formData["greeting_type"]; ok && greetType != "" {
		session.Set("greeting_type", greetType)
	}
	if recipient, ok := formData["recipient"]; ok && recipient != "" {
		session.Set("recipient", recipient)
	}
	if customMsg, ok := formData["custom_message"]; ok && customMsg != "" {
		session.Set("custom_message", customMsg)
	}

	// Handle complete
	if action == gofra.ActionComplete {
		return completeGreeting(session, formData)
	}

	// Determine what to show based on session state
	return determineGreetingStep(session)
}

// handleGreetingPrev handles going back one step.
func handleGreetingPrev(session *gofra.CommandSession) (*gofra.CommandResponse, error) {
	_, hasGreetType := session.Get("greeting_type")
	_, hasRecipient := session.Get("recipient")

	// Clear from most recent to least recent
	if hasRecipient {
		session.Set("recipient", nil)
		session.Set("custom_message", nil)
		// Show recipient form
		return showRecipientForm(session)
	}
	if hasGreetType {
		session.Set("greeting_type", nil)
	}
	// Show greeting type selection
	return showGreetingTypeForm()
}

// determineGreetingStep figures out what form to show based on session state.
func determineGreetingStep(session *gofra.CommandSession) (*gofra.CommandResponse, error) {
	greetTypeVal, hasGreetType := session.Get("greeting_type")
	recipientVal, hasRecipient := session.Get("recipient")

	// No greeting type - show type selection
	if !hasGreetType || greetTypeVal == nil {
		return showGreetingTypeForm()
	}

	// No recipient - show recipient form
	if !hasRecipient || recipientVal == nil {
		return showRecipientForm(session)
	}

	// Have both - show confirmation
	return showConfirmForm(session)
}

// showGreetingTypeForm shows the greeting type selection.
func showGreetingTypeForm() (*gofra.CommandResponse, error) {
	form := gofra.NewFormBuilder("form", "Select Greeting Type").
		Instructions("Choose the type of greeting you want to send").
		AddFieldWithOptions("greeting_type", "list-single", "Greeting Type", "",
			[]gofra.XDataOption{
				{Label: "Hello", Value: "hello"},
				{Label: "Good Morning", Value: "morning"},
				{Label: "Good Evening", Value: "evening"},
				{Label: "Custom", Value: "custom"},
			}).
		Build()

	return &gofra.CommandResponse{
		Status:  gofra.StatusExecuting,
		Actions: gofra.NewActionsNextOnly(),
		Form:    form,
	}, nil
}

// showRecipientForm shows the recipient entry form.
func showRecipientForm(session *gofra.CommandSession) (*gofra.CommandResponse, error) {
	greetTypeStr, ok := session.GetStr("greeting_type")
	if !ok {
		greetTypeStr = ""
	}

	builder := gofra.NewFormBuilder("form", "Enter Details").
		Instructions("Enter the JID to send the greeting to").
		AddField("recipient", "jid-single", "Recipient JID", "")

	// Show custom message field only for custom greeting type
	if greetTypeStr == "custom" {
		builder.AddField("custom_message", "text-multi", "Custom Message", "")
	}

	return &gofra.CommandResponse{
		Status:  gofra.StatusExecuting,
		Actions: gofra.NewActionsPrevNext(),
		Form:    builder.Build(),
	}, nil
}

// showConfirmForm shows the confirmation form.
func showConfirmForm(session *gofra.CommandSession) (*gofra.CommandResponse, error) {
	greetTypeStr, ok := session.GetStr("greeting_type")
	if !ok {
		return &gofra.CommandResponse{
			Status:     gofra.StatusCompleted,
			IsComplete: true,
			Notes:      []gofra.Note{gofra.NewErrorNote("Missing greeting type")},
		}, nil
	}
	recipientJID, ok := session.GetStr("recipient")
	if !ok {
		return &gofra.CommandResponse{
			Status:     gofra.StatusCompleted,
			IsComplete: true,
			Notes:      []gofra.Note{gofra.NewErrorNote("Missing recipient")},
		}, nil
	}
	customMsgStr, _ := session.GetStr("custom_message") // optional field

	message := buildGreetingMessage(greetTypeStr, customMsgStr)

	form := gofra.NewFormBuilder("form", "Confirm Greeting").
		Instructions("Review your greeting and click Complete to send").
		AddField("to", "fixed", "Recipient", recipientJID).
		AddField("preview", "fixed", "Message", message).
		Build()

	return &gofra.CommandResponse{
		Status:  gofra.StatusExecuting,
		Actions: gofra.NewActionsPrevComplete(),
		Form:    form,
	}, nil
}

// completeGreeting sends the greeting and completes the command.
func completeGreeting(session *gofra.CommandSession, formData map[string]string) (*gofra.CommandResponse, error) {
	greetTypeStr, ok := session.GetStr("greeting_type")
	if !ok {
		greetTypeStr = "hello" // default
	}
	recipientJID, hasRecipient := session.GetStr("recipient")
	customMsgStr, _ := session.GetStr("custom_message") // optional field

	if !hasRecipient {
		return &gofra.CommandResponse{
			Status:     gofra.StatusCompleted,
			IsComplete: true,
			Notes: []gofra.Note{
				gofra.NewErrorNote("Recipient JID is required"),
			},
		}, nil
	}

	message := buildGreetingMessage(greetTypeStr, customMsgStr)

	// Actually send the message
	if err := g.SendMessage(recipientJID, message, stanza.ChatMessage); err != nil {
		g.Logger.Error(fmt.Sprintf("Failed to send greeting: %v", err))
		return &gofra.CommandResponse{
			Status:     gofra.StatusCompleted,
			IsComplete: true,
			Notes: []gofra.Note{
				gofra.NewErrorNote(fmt.Sprintf("Failed to send: %v", err)),
			},
		}, nil
	}

	g.Logger.Info(fmt.Sprintf("Greeting sent to %s: %s", recipientJID, message))

	return &gofra.CommandResponse{
		Status:     gofra.StatusCompleted,
		IsComplete: true,
		Notes: []gofra.Note{
			gofra.NewInfoNote(fmt.Sprintf("Greeting sent to %s!", recipientJID)),
		},
	}, nil
}

// buildGreetingMessage constructs the greeting message.
func buildGreetingMessage(greetType, customMsg string) string {
	switch greetType {
	case "hello":
		return "Hello!"
	case "morning":
		return "Good morning!"
	case "evening":
		return "Good evening!"
	case "custom":
		if customMsg != "" {
			return strings.TrimSpace(customMsg)
		}
		return "Greetings!"
	default:
		return "Hello!"
	}
}
