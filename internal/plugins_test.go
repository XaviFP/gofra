package gofra

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

const test_plugins_path = "../test_plugins/bin/"

// func TestLoadAllPlugins(t *testing.T) {
// 	config := Config{PluginPaths: []string{test_plugins_path}}
// 	var g *Gofra
// 	plugins := NewPlugins(config)
// 	plugins.loadAll(config, g)

// 	assert.Len(t, plugins, 2)
// }

func TestGetFileNamesInPaths(t *testing.T) {
	fileNames, err := getFileNamesInPaths([]string{test_plugins_path})
	assert.Nil(t, err)

	expected := []string{
		fmt.Sprintf("%snaughty.so", test_plugins_path),
		fmt.Sprintf("%snormie.so", test_plugins_path),
		fmt.Sprintf("%snot_really.so", test_plugins_path),
	}
	var actual []string

	for _, rname := range expected {
		for _, fname := range fileNames {
			if fname == rname {
				actual = append(actual, fname)
				break
			}
		}
	}

	assert.ElementsMatch(t, expected, actual)
}

// func TestIsPlugin(t *testing.T) {
// 	_, ok := isPlugin(test_plugins_path + "not_really.so")
// 	assert.False(t, ok)

// 	_, ok = isPlugin(test_plugins_path + "normie.so")
// 	assert.True(t, ok)

// 	_, ok = isPlugin(test_plugins_path + "naughty.so")
// 	assert.True(t, ok)
// }

// func TestLoadPlugin(t *testing.T) {
// 	config := Config{PluginPaths: []string{test_plugins_path}}
// 	var g *Gofra
// 	plugins := NewPlugins(config)

// 	ok := plugins.load(test_plugins_path + "not_really.so", config, g)
// 	assert.False(t, ok)

// 	ok = plugins.load(test_plugins_path + "normie.so", config, g)
// 	assert.True(t, ok)

// 	assert.Equal(t, "normie", plugins["normie"].Name())
// }

// func TestRunPanickingHandler(t *testing.T) {
// 	config := Config{PluginPaths: []string{test_plugins_path}}
// 	var g *Gofra
// 	g.plugins = NewPlugins(config)
// 	g.em = NewEventManager(g.Logger)

// 	ok := g.plugins.load(test_plugins_path + "naughty.so", config, g)
// 	assert.False(t, ok)

// 	publishPanickingEvent(g, t)
// }

// func publishPanickingEvent(g *Gofra, t *testing.T) {
// 	defer func() {
// 		assert.Nil(t, recover())
// 	}()

// 	g.Publish(Event{Name: "naughtyCrash"})
// }
