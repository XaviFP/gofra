/*
list is a gofra plugin that allows users to manage lists
*/

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	gofra "github.com/XaviFP/gofra/internal"
)

// Form field type constants
const (
	fieldTypeListSingle = "list-single"
	fieldTypeListMulti  = "list-multi"
	fieldTypeTextSingle = "text-single"
	fieldTypeTextMulti  = "text-multi"
)

// Session key constants
const (
	keyAction        = "action"
	keyListName      = "list_name"
	keyItems         = "items"
	keyManageAction  = "manage_action"
	keySelectedItems = "selected_items_multi"
)

// Error constants
var errNotEnoughArguments = errors.New("not enough arguments")

var Plugin plugin

var g *gofra.Gofra

var lists State

type plugin struct{}

func (p plugin) Name() string {
	return "List"
}

func (p plugin) Description() string {
	return "Create and manage lists in MUCs"
}

func (p plugin) Help() string {
	reply := g.Publish(gofra.Event{Name: "command/getCommandChar", MB: gofra.MessageBody{}, Payload: nil})
	commandChar := reply.GetAnswer()
	return fmt.Sprintf(`
	Usage:
	%[1]slist new <list_name>; 
	%[1]slist add <list_name> <item_name>;
	%[1]slist del <list_name> <item_id>;
	%[1]slist show <list_name>;
	%[1]slist show all;
	`, commandChar)
}

func (p plugin) Init(c gofra.Config, api *gofra.Gofra) {
	g = api
	g.Subscribe(
		"command/list",
		p.Name(),
		handleList,
		0,
	)

	// Register ad-hoc command after all plugins are loaded
	g.Subscribe(
		"initialized",
		p.Name(),
		registerAdhocCommand,
		0,
	)

	lists = make(State)

	loadState()
}

func registerAdhocCommand(e gofra.Event) *gofra.Reply {
	g.Publish(gofra.Event{
		Name: "adhoc/register",
		Payload: map[string]interface{}{
			"command": &gofra.AdHocCommand{
				Node:    "list-manager",
				Name:    "List Manager",
				Handler: handleListAdhoc,
			},
		},
	})
	return nil
}

type command struct {
	action   string
	listName string
	item     string
	itemID   int
}

func parseCommand(args []string) (command, error) {
	if len(args) < 2 {
		return command{}, errNotEnoughArguments
	}

	cmd := command{
		action:   args[0],
		listName: args[1],
	}

	switch cmd.action {
	case "add":
		if len(args) < 3 {
			return command{}, errNotEnoughArguments
		}
		cmd.item = strings.Join(args[2:], " ")
	case "del":
		if len(args) < 3 {
			return command{}, errNotEnoughArguments
		}
		id, err := strconv.Atoi(args[2])
		if err != nil {
			return command{}, errors.New("invalid item id")
		}
		cmd.itemID = id
	}

	return cmd, nil
}

// !list new list_name
// !list add list_name item
// !list del list_name item_id
// !list show list_name
// !list show all
func handleList(e gofra.Event) *gofra.Reply {
	msg := e.MB
	args := strings.Fields(msg.Body)[1:]

	if len(args) == 0 || args[0] == "" {
		sendReply(e, "Possible subcommands are: new, add, del, show")
		return nil
	}

	cmd, err := parseCommand(args)
	if err != nil {
		sendReply(e, err.Error())
	}

	room := msg.From.Bare().String()

	switch cmd.action {
	case "new":
		lists.newList(room, cmd.listName)
		persistState()
		sendReply(e, "List created")

	case "add":
		lists.addItem(room, cmd.listName, cmd.item)
		persistState()
		sendReply(e, "Item added")

	case "del":
		lists.delItem(room, cmd.listName, cmd.itemID)
		persistState()
		sendReply(e, "Item deleted")

	case "show":
		if cmd.listName == "all" {
			sendReply(e, lists.showAll(room))
		} else {
			sendReply(e, lists.show(room, cmd.listName))
		}
	}

	return nil
}

func sendReply(e gofra.Event, reply string) {
	if err := g.SendStanza(e.MB.Reply(reply)); err != nil {
		g.Logger.Error(err.Error())
	}
}

func persistState() {
	serialized, err := json.MarshalIndent(lists, "", " ")
	if err != nil {
		g.Logger.Error(err.Error())
		return
	}

	file, err := os.Create("/data/lists.json")
	if err != nil {
		g.Logger.Error(err.Error())
		return
	}
	defer file.Close()

	_, err = file.Write(serialized)
	if err != nil {
		g.Logger.Error(err.Error())
	}
}

func loadState() {
	serialized, err := os.ReadFile("/data/lists.json")
	if err != nil {
		g.Logger.Error(err.Error())
		return
	}

	var state State
	err = json.Unmarshal(serialized, &state)
	if err != nil {
		g.Logger.Error(err.Error())
		return
	}

	lists = state
}

// handleListAdhoc is the ad-hoc command handler for list management.
func handleListAdhoc(session *gofra.CommandSession, action gofra.CommandAction, formData map[string]string) (*gofra.CommandResponse, error) {
	g.Logger.Debug(fmt.Sprintf("list-manager: action=%s formData=%v", action, formData))

	// Handle cancel
	if action == gofra.ActionCancel {
		return &gofra.CommandResponse{
			Status:     gofra.StatusCanceled,
			IsComplete: true,
		}, nil
	}

	// Handle prev - clear last piece of data and show appropriate form
	if action == gofra.ActionPrev {
		return handlePrev(session)
	}

	// Save form data
	if act, ok := formData[keyAction]; ok && act != "" {
		session.Set(keyAction, act)
	}
	if listName, ok := formData[keyListName]; ok && listName != "" {
		session.Set(keyListName, listName)
	}
	if items, ok := formData[keyItems]; ok && items != "" {
		session.Set(keyItems, items)
	}
	if manageAction, ok := formData[keyManageAction]; ok && manageAction != "" {
		session.Set(keyManageAction, manageAction)
	}
	// Note: selected_items_multi is stored directly by adhoc plugin for list-multi fields

	// Determine what to show based on what data we have
	return determineNextStep(session)
}

// handlePrev goes back one step by clearing the most recent data.
func handlePrev(session *gofra.CommandSession) (*gofra.CommandResponse, error) {
	act, hasAction := session.GetStr(keyAction)
	listName, hasListName := session.GetStr(keyListName)
	_, hasManageAction := session.GetStr(keyManageAction)
	_, hasSelectedItems := session.GetStrSlice(keySelectedItems)

	// Manage action: selected_items -> manage_action -> list_name -> action
	if hasSelectedItems || hasManageAction {
		session.Set(keySelectedItems, nil)
		session.Set(keyManageAction, nil)
		if hasListName {
			return buildManageItemsForm(session, listName)
		}
	}
	if hasListName {
		session.Set(keyListName, nil)
		session.Set(keyItems, nil)
		if hasAction {
			return buildStage1Form(session, act)
		}
	}
	if hasAction {
		session.Set(keyAction, nil)
	}
	// Show action selection
	return buildStage0Form()
}

// determineNextStep figures out what form/action to show based on session state.
func determineNextStep(session *gofra.CommandSession) (*gofra.CommandResponse, error) {
	act, hasAction := session.GetStr(keyAction)

	// No action yet - show action selection
	if !hasAction {
		return buildStage0Form()
	}

	listName, hasListName := session.GetStr(keyListName)

	// We have an action - check if we can execute or need more input
	switch act {
	case "new", "show":
		if hasListName {
			return executeListAction(session)
		}
		return buildStage1Form(session, act)

	case "add":
		_, hasItems := session.GetStr(keyItems)
		if hasListName && hasItems {
			return executeListAction(session)
		}
		return buildStage1Form(session, act)

	case "manage":
		_, hasManageAction := session.GetStr(keyManageAction)
		_, hasSelectedItems := session.GetStrSlice(keySelectedItems)
		g.Logger.Debug(fmt.Sprintf("list-manager manage: hasName=%v hasManageAction=%v hasSelectedItems=%v",
			hasListName, hasManageAction, hasSelectedItems))
		if hasListName && hasManageAction && hasSelectedItems {
			return executeListAction(session)
		}
		if hasListName {
			return buildManageItemsForm(session, listName)
		}
		return buildStage1Form(session, act)
	}

	return buildStage0Form()
}

// buildManageItemsForm shows all items with checkboxes for multi-select operations.
func buildManageItemsForm(session *gofra.CommandSession, listName string) (*gofra.CommandResponse, error) {
	room := session.Requester
	var itemOptions []gofra.XDataOption
	if roomLists, ok := lists[room]; ok {
		if list, ok := roomLists[listName]; ok {
			for i, item := range list.Items {
				itemOptions = append(itemOptions, gofra.XDataOption{
					Label: item,
					Value: strconv.Itoa(i),
				})
			}
		}
	}

	if len(itemOptions) == 0 {
		return &gofra.CommandResponse{
			Status:     gofra.StatusCompleted,
			IsComplete: true,
			Notes:      []gofra.Note{gofra.NewInfoNote(fmt.Sprintf("List '%s' is empty", listName))},
		}, nil
	}

	form := gofra.NewFormBuilder("form", fmt.Sprintf("Manage: %s", listName)).
		Instructions("Select items and choose an action").
		AddFieldWithMultipleValues("selected_items", fieldTypeListMulti, "Items", nil, itemOptions).
		AddFieldWithOptions(keyManageAction, fieldTypeListSingle, "Action", "done", []gofra.XDataOption{
			{Label: "✓ Mark as done", Value: "done"},
			{Label: "✗ Delete selected", Value: "delete"},
		}).
		Build()

	return &gofra.CommandResponse{
		Status:  gofra.StatusExecuting,
		Actions: gofra.NewActionsPrevComplete(),
		Form:    form,
	}, nil
}

// buildStage0Form builds the action selection form.
func buildStage0Form() (*gofra.CommandResponse, error) {
	form := gofra.NewFormBuilder("form", "List Manager").
		Instructions("Select the action you want to perform").
		AddFieldWithOptions(keyAction, fieldTypeListSingle, "Action", "", []gofra.XDataOption{
			{Label: "Create new list", Value: "new"},
			{Label: "Add item to list", Value: "add"},
			{Label: "Manage list (mark done/delete)", Value: "manage"},
			{Label: "Show list contents", Value: "show"},
		}).
		Build()

	return &gofra.CommandResponse{
		Status:  gofra.StatusExecuting,
		Actions: gofra.NewActionsNextOnly(),
		Form:    form,
	}, nil
}

// buildStage1Form builds the form for stage 1 based on the action.
func buildStage1Form(session *gofra.CommandSession, action string) (*gofra.CommandResponse, error) {
	room := session.Requester

	// Get available lists for this room
	var listOptions []gofra.XDataOption
	if roomLists, ok := lists[room]; ok {
		for name := range roomLists {
			listOptions = append(listOptions, gofra.XDataOption{Label: name, Value: name})
		}
	}

	switch action {
	case "new":
		form := gofra.NewFormBuilder("form", "Create New List").
			Instructions("Enter a name for the new list").
			AddField(keyListName, fieldTypeTextSingle, "List Name", "").
			Build()

		return &gofra.CommandResponse{
			Status:  gofra.StatusExecuting,
			Actions: gofra.NewActionsPrevComplete(),
			Form:    form,
		}, nil

	case "show":
		showOptions := append([]gofra.XDataOption{{Label: "All Lists", Value: "all"}}, listOptions...)
		defaultShow := "all"

		form := gofra.NewFormBuilder("form", "Show List").
			Instructions("Select a list to display").
			AddFieldWithOptions(keyListName, fieldTypeListSingle, "List", defaultShow, showOptions).
			Build()

		return &gofra.CommandResponse{
			Status:  gofra.StatusExecuting,
			Actions: gofra.NewActionsPrevComplete(),
			Form:    form,
		}, nil

	case "add":
		defaultList := ""
		if len(listOptions) > 0 {
			defaultList = listOptions[0].Value
		}
		form := gofra.NewFormBuilder("form", "Add Items to List").
			Instructions("Select a list and enter items (one per line)").
			AddFieldWithOptions(keyListName, fieldTypeListSingle, "List", defaultList, listOptions).
			AddField(keyItems, fieldTypeTextMulti, "Items", "").
			Build()

		return &gofra.CommandResponse{
			Status:  gofra.StatusExecuting,
			Actions: gofra.NewActionsPrevComplete(),
			Form:    form,
		}, nil

	case "manage":
		if len(listOptions) == 0 {
			return completedResponse(gofra.NewInfoNote("No lists exist yet. Create one first!"))
		}
		defaultList := listOptions[0].Value
		form := gofra.NewFormBuilder("form", "Manage List").
			Instructions("Select the list to manage").
			AddFieldWithOptions(keyListName, fieldTypeListSingle, "List", defaultList, listOptions).
			Build()

		return &gofra.CommandResponse{
			Status:  gofra.StatusExecuting,
			Actions: gofra.NewActionsPrevNext(),
			Form:    form,
		}, nil
	}

	return completedResponse(gofra.NewErrorNote("Unknown action"))
}

// completedResponse creates a completed CommandResponse with the given note.
func completedResponse(note gofra.Note) (*gofra.CommandResponse, error) {
	return &gofra.CommandResponse{
		Status:     gofra.StatusCompleted,
		IsComplete: true,
		Notes:      []gofra.Note{note},
	}, nil
}

// executeNewList creates a new list.
func executeNewList(room, listName string) (*gofra.CommandResponse, error) {
	lists.newList(room, listName)
	persistState()
	return completedResponse(gofra.NewInfoNote(fmt.Sprintf("List '%s' created", listName)))
}

// executeShowList displays list contents.
func executeShowList(room, listName string) (*gofra.CommandResponse, error) {
	var content string
	if listName == "all" {
		content = lists.showAll(room)
		if content == "" {
			content = "No lists found"
		}
	} else {
		content = lists.show(room, listName)
		if content == "" {
			content = "List is empty"
		}
	}
	return completedResponse(gofra.NewInfoNote(content))
}

// executeAddItems adds items to a list.
func executeAddItems(session *gofra.CommandSession, room, listName string) (*gofra.CommandResponse, error) {
	itemsStr, ok := session.GetStr(keyItems)
	if !ok {
		return completedResponse(gofra.NewErrorNote("No items provided"))
	}

	var addedCount int
	for _, item := range strings.Split(itemsStr, "\n") {
		item = strings.TrimSpace(item)
		if item != "" {
			lists.addItem(room, listName, item)
			addedCount++
		}
	}
	persistState()

	if addedCount == 0 {
		return completedResponse(gofra.NewErrorNote("No items provided"))
	}
	return completedResponse(gofra.NewInfoNote(fmt.Sprintf("Added %d item(s) to '%s'", addedCount, listName)))
}

// executeManageList handles manage actions (mark done/delete).
func executeManageList(session *gofra.CommandSession, room, listName string) (*gofra.CommandResponse, error) {
	manageAction, ok := session.GetStr(keyManageAction)
	if !ok {
		return completedResponse(gofra.NewErrorNote("No manage action selected"))
	}
	selectedItems, ok := session.GetStrSlice(keySelectedItems)
	if !ok {
		return completedResponse(gofra.NewInfoNote("No items selected"))
	}

	indices := parseAndSortIndices(selectedItems)

	switch manageAction {
	case "done":
		for _, idx := range indices {
			lists.markDone(room, listName, idx)
		}
		persistState()
		return completedResponse(gofra.NewInfoNote(fmt.Sprintf("Marked %d item(s) as done in '%s'", len(indices), listName)))

	case "delete":
		for _, idx := range indices {
			lists.delItem(room, listName, idx)
		}
		persistState()
		return completedResponse(gofra.NewInfoNote(fmt.Sprintf("Deleted %d item(s) from '%s'", len(indices), listName)))
	}

	return completedResponse(gofra.NewErrorNote("Unknown manage action"))
}

// parseAndSortIndices parses string indices and returns them sorted in descending order.
func parseAndSortIndices(items []string) []int {
	var indices []int
	for _, idxStr := range items {
		idx, err := strconv.Atoi(idxStr)
		if err != nil {
			continue
		}
		indices = append(indices, idx)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(indices)))
	return indices
}

// executeListAction performs the selected action.
func executeListAction(session *gofra.CommandSession) (*gofra.CommandResponse, error) {
	action, ok := session.GetStr(keyAction)
	if !ok {
		return completedResponse(gofra.NewErrorNote("No action specified"))
	}

	room := session.Requester
	listName, hasListName := session.GetStr(keyListName)

	if !hasListName {
		return completedResponse(gofra.NewErrorNote("No list name provided"))
	}

	switch action {
	case "new":
		return executeNewList(room, listName)
	case "show":
		return executeShowList(room, listName)
	case "add":
		return executeAddItems(session, room, listName)
	case "manage":
		return executeManageList(session, room, listName)
	}

	return completedResponse(gofra.NewErrorNote("Unknown action"))
}
