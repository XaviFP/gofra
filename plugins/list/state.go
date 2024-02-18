package main

import (
	"fmt"
	"strings"
)

type List struct {
	Items []string `json:"items"`
}

// {room: {list_name: list}
type State map[string]map[string]*List 

func (s State) addItem(room, listName, item string) {
	s[room][listName].addItem(item)
}

func (s State) delItem(room, listName string, id int) {
	s[room][listName].delItem(id)
}

func (s State) show(room, listName string) string {
	return s[room][listName].show()
}

func (s State) showAll(room string) string {
	var lists strings.Builder
	for listName := range s[room] {
		lists.WriteString(listName + "\n")
	}
	return lists.String()
}

func (s State) newList(room, listName string) {
	if _, ok := s[room]; !ok {
		s[room] = make(map[string]*List)
	}
	s[room][listName] = &List{}
}

func (l *List) addItem(item string) {
	l.Items = append(l.Items, item)
}

func (l *List) delItem(id int) {
	if id < 0 || id >= len(l.Items) {
		return
	}
	l.Items = append(l.Items[:id], l.Items[id+1:]...)
}

func (l *List) show() string {
	var items strings.Builder
	for i, item := range l.Items {
		items.WriteString(fmt.Sprintf("%d. %s\n", i, item))
	}
	return items.String()
}
