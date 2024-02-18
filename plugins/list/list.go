/*
list is a gofra plugin that allows users to manage lists
*/

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	gofra "github.com/XaviFP/gofra/internal"
)

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

func (p plugin) Init(c gofra.Config, gofra *gofra.Gofra) {
	g = gofra
	g.Subscribe(
		"command/list",
		p.Name(),
		handleList,
		0,
	)

	lists = make(State)

	loadState()
}

type command struct {
	action   string
	listName string
	item     string
	itemID   int
}

func parseCommand(args []string) (command, error) {
	if len(args) < 2 {
		return command{}, errors.New("not enough arguments")
	}

	cmd := command{
		action:   args[0],
		listName: args[1],
	}

	switch cmd.action {
	case "add":
		if len(args) < 3 {
			return command{}, errors.New("not enough arguments")
		}
		cmd.item = strings.Join(args[2:], " ")
	case "del":
		if len(args) < 3 {
			return command{}, errors.New("not enough arguments")
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

	if len(args) < 1 || (len(args) > 0 && args[0] == "") {
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
