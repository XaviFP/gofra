package gofra

import (
	"testing"
)

const test_plugins_path = "../test_plugins/bin/"

func TestNewPlugins(t *testing.T) {
	plugins := NewPlugins(Config{})
	if len(plugins) != 0 {
		t.Error(`NewEvents returns a non empty Events object`)
	}
}

func TestLoadAllPlugins(t *testing.T) {
	config := Config{PluginPaths: []string{test_plugins_path},}
	var g API
	plugins := NewPlugins(config)
	plugins.loadAll(config, g)
	
	if len(plugins) != 2 {
		t.Error(`plugins were not loaded correctly`)
	}
}

func TestGetFileNamesInPaths(t *testing.T) {
	fileNames, err := getFileNamesInPaths([]string{test_plugins_path})
	if err != nil {
		// Skip if there's a File System failure
		return
	}
	referenceNames := []string{"naughty.so", "normie.so", "not_really.so"}
	var found bool
	for _, rname := range referenceNames {
		for _, fname := range fileNames {
			if fname == test_plugins_path + rname {
				found = true
				break
			}
		}
		if !found {
			t.Errorf(`file %s should be in list %v`, rname, fileNames)
		}
		found = false
	}
}

func TestIsPlugin(t *testing.T) {
	_, ok := isPlugin(test_plugins_path + "not_really.so")
	if ok {
		t.Error(`not_really shouldn't be a valid plugin`)
	}
	_, ok = isPlugin(test_plugins_path + "normie.so")
	if !ok {
		t.Error(`normie should be a valid plugin`)
	}
	_, ok = isPlugin(test_plugins_path + "naughty.so")
	if !ok {
		t.Error(`naughty should be a valid plugin`)
	}

}

func TestLoadPlugin(t *testing.T) {
	config := Config{PluginPaths: []string{test_plugins_path},}
	var g API
	plugins := NewPlugins(config)
	ok := plugins.loadPlugin(test_plugins_path + "not_really.so", config, g)
	if ok {
		t.Error(`not_really shouldn't be a valid plugin`)
	}
	ok = plugins.loadPlugin(test_plugins_path + "normie.so", config, g)
	if !ok {
		t.Error(`normie should be a valid plugin`)
	}
	if plugins["normie"].Name() != "normie" {
		t.Error(`plugin normie was not loaded correctly`)
	}

}
