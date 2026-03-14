package gofra

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const (
	// CommandsNS is the namespace for XEP-0050 Ad-Hoc Commands.
	CommandsNS = "http://jabber.org/protocol/commands"

	// Default session timeout.
	defaultSessionTimeout = 10 * time.Minute
)

// CommandStatus represents the status of a command execution.
type CommandStatus string

const (
	StatusExecuting CommandStatus = "executing"
	StatusCompleted CommandStatus = "completed"
	StatusCanceled  CommandStatus = "canceled"
)

// CommandAction represents an action in a command request.
type CommandAction string

const (
	ActionExecute  CommandAction = "execute"
	ActionCancel   CommandAction = "cancel"
	ActionPrev     CommandAction = "prev"
	ActionNext     CommandAction = "next"
	ActionComplete CommandAction = "complete"
)

// AdHocCommand defines an ad-hoc command that can be registered.
type AdHocCommand struct {
	Node    string
	Name    string
	Handler CommandHandler
}

// CommandHandler handles command execution stages.
type CommandHandler func(session *CommandSession, action CommandAction, formData map[string]string) (*CommandResponse, error)

// CommandResponse represents the response from a command handler.
type CommandResponse struct {
	Status       CommandStatus
	Actions      *Actions
	Notes        []Note
	Form         *XData
	IsComplete   bool
	ErrorType    string
	ErrorMessage string
}

// CommandSession represents an active command session.
type CommandSession struct {
	ID        string
	Node      string
	Requester string
	Data      map[string]interface{}
	CreatedAt time.Time
	ExpiresAt time.Time
	mu        sync.Mutex
}

// Get retrieves a value from session data.
func (s *CommandSession) Get(key string) (interface{}, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.Data[key]
	return v, ok
}

// GetStr retrieves a string value from session data.
// Returns the value and true if found and is a non-empty string.
// Returns empty string and false if not found, nil, or wrong type.
func (s *CommandSession) GetStr(key string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.Data[key]
	if !ok || v == nil {
		return "", false
	}
	str, ok := v.(string)
	if !ok || str == "" {
		return "", false
	}
	return str, true
}

// GetStrSlice retrieves a []string value from session data.
// Returns the value and true if found and is a non-empty slice.
// Returns nil and false if not found, nil, wrong type, or empty.
func (s *CommandSession) GetStrSlice(key string) ([]string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.Data[key]
	if !ok || v == nil {
		return nil, false
	}
	slice, ok := v.([]string)
	if !ok || len(slice) == 0 {
		return nil, false
	}
	return slice, true
}

// Set stores a value in session data.
func (s *CommandSession) Set(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Data == nil {
		s.Data = make(map[string]interface{})
	}
	s.Data[key] = value
}

// CommandRegistry manages ad-hoc command registration and sessions.
type CommandRegistry struct {
	commands       map[string]*AdHocCommand
	sessions       map[string]*CommandSession
	sessionTimeout time.Duration
	mu             sync.RWMutex
	stopCleanup    chan struct{}
}

// NewCommandRegistry creates a new command registry.
func NewCommandRegistry() *CommandRegistry {
	r := &CommandRegistry{
		commands:       make(map[string]*AdHocCommand),
		sessions:       make(map[string]*CommandSession),
		sessionTimeout: defaultSessionTimeout,
		stopCleanup:    make(chan struct{}),
	}

	go r.cleanupExpiredSessions()

	return r
}

// SetSessionTimeout sets the session timeout duration.
func (r *CommandRegistry) SetSessionTimeout(d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessionTimeout = d
}

// Register adds a new ad-hoc command.
func (r *CommandRegistry) Register(cmd *AdHocCommand) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.commands[cmd.Node] = cmd
}

// Unregister removes an ad-hoc command.
func (r *CommandRegistry) Unregister(node string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.commands, node)
}

// GetCommand retrieves a registered command by node.
func (r *CommandRegistry) GetCommand(node string) (*AdHocCommand, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cmd, ok := r.commands[node]
	return cmd, ok
}

// ListCommands returns all registered commands.
func (r *CommandRegistry) ListCommands() []*AdHocCommand {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cmds := make([]*AdHocCommand, 0, len(r.commands))
	for _, cmd := range r.commands {
		cmds = append(cmds, cmd)
	}
	return cmds
}

// ListCommandsForJID returns commands available for a specific JID.
func (r *CommandRegistry) ListCommandsForJID(requesterJID string) []*AdHocCommand {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cmds := make([]*AdHocCommand, 0, len(r.commands))
	for _, cmd := range r.commands {
		cmds = append(cmds, cmd)
	}
	return cmds
}

// CreateSession creates a new command session.
func (r *CommandRegistry) CreateSession(node, requesterJID string) *CommandSession {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := generateSessionID()
	now := time.Now()
	session := &CommandSession{
		ID:        id,
		Node:      node,
		Requester: requesterJID,
		Data:      make(map[string]interface{}),
		CreatedAt: now,
		ExpiresAt: now.Add(r.sessionTimeout),
	}
	r.sessions[id] = session
	return session
}

// GetSession retrieves an active session.
func (r *CommandRegistry) GetSession(sessionID string) (*CommandSession, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	session, ok := r.sessions[sessionID]
	if !ok {
		return nil, false
	}
	if time.Now().After(session.ExpiresAt) {
		return nil, false
	}
	return session, true
}

// RefreshSession extends the session expiration time.
func (r *CommandRegistry) RefreshSession(sessionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if session, ok := r.sessions[sessionID]; ok {
		session.ExpiresAt = time.Now().Add(r.sessionTimeout)
	}
}

// DeleteSession removes a session.
func (r *CommandRegistry) DeleteSession(sessionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, sessionID)
}

// cleanupExpiredSessions periodically removes expired sessions.
func (r *CommandRegistry) cleanupExpiredSessions() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.mu.Lock()
			now := time.Now()
			for id, session := range r.sessions {
				if now.After(session.ExpiresAt) {
					delete(r.sessions, id)
				}
			}
			r.mu.Unlock()
		case <-r.stopCleanup:
			return
		}
	}
}

// Stop stops the cleanup goroutine.
func (r *CommandRegistry) Stop() {
	close(r.stopCleanup)
}

// generateSessionID creates a unique session identifier.
func generateSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// NewActionsNextOnly creates Actions with only Next allowed.
func NewActionsNextOnly() *Actions {
	return &Actions{
		Execute: "next",
		Next:    &struct{}{},
	}
}

// NewActionsPrevNext creates Actions with Prev and Next allowed.
func NewActionsPrevNext() *Actions {
	return &Actions{
		Execute: "next",
		Prev:    &struct{}{},
		Next:    &struct{}{},
	}
}

// NewActionsPrevComplete creates Actions with Prev and Complete allowed.
func NewActionsPrevComplete() *Actions {
	return &Actions{
		Execute:  "complete",
		Prev:     &struct{}{},
		Complete: &struct{}{},
	}
}

// NewActionsComplete creates Actions with only Complete allowed.
func NewActionsComplete() *Actions {
	return &Actions{
		Execute:  "complete",
		Complete: &struct{}{},
	}
}

// NewFormBuilder creates a new form builder.
func NewFormBuilder(formType, title string) *FormBuilder {
	return &FormBuilder{
		form: &XData{
			Type:  formType,
			Title: title,
		},
	}
}

// FormBuilder helps construct XData forms.
type FormBuilder struct {
	form *XData
}

// Instructions sets the form instructions.
func (b *FormBuilder) Instructions(text string) *FormBuilder {
	b.form.Instructions = text
	return b
}

// AddField adds a field to the form.
func (b *FormBuilder) AddField(varName, fieldType, label, value string) *FormBuilder {
	var values []string
	if value != "" {
		values = []string{value}
	}
	b.form.Fields = append(b.form.Fields, XDataField{
		Var:    varName,
		Type:   fieldType,
		Label:  label,
		Values: values,
	})
	return b
}

// AddFieldWithOptions adds a field with options.
func (b *FormBuilder) AddFieldWithOptions(varName, fieldType, label, value string, options []XDataOption) *FormBuilder {
	var values []string
	if value != "" {
		values = []string{value}
	}
	b.form.Fields = append(b.form.Fields, XDataField{
		Var:     varName,
		Type:    fieldType,
		Label:   label,
		Values:  values,
		Options: options,
	})
	return b
}

// AddFieldWithMultipleValues adds a field with multiple selected values (for list-multi).
func (b *FormBuilder) AddFieldWithMultipleValues(varName, fieldType, label string, values []string, options []XDataOption) *FormBuilder {
	b.form.Fields = append(b.form.Fields, XDataField{
		Var:     varName,
		Type:    fieldType,
		Label:   label,
		Values:  values,
		Options: options,
	})
	return b
}

// Build returns the constructed form.
func (b *FormBuilder) Build() *XData {
	return b.form
}

// NewInfoNote creates an info note.
func NewInfoNote(text string) Note {
	return Note{Type: "info", Value: text}
}

// NewWarnNote creates a warning note.
func NewWarnNote(text string) Note {
	return Note{Type: "warn", Value: text}
}

// NewErrorNote creates an error note.
func NewErrorNote(text string) Note {
	return Note{Type: "error", Value: text}
}
