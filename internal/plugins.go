package gofra

import (
	"log"
	"os"
	"plugin"
	"strings"
)

// Interface to be satisfied by any Gofra plugin
type Plugin interface {
	Name() string
	Description() string
	Help() string
	Init(Config, *Gofra)
}

// Interface to be satisfied by plugins that need an execution loop
// like, for example, an HTTP server. Run method is executed as a goroutine.
type Runnable interface {
	Run()
}

type Plugins map[string]Plugin

func (p Plugins) Init(config Config, gofra *Gofra) error {
	return p.loadAll(config, gofra)
}

func NewPlugins(config Config) Plugins {
	return make(Plugins)
}

func getFileNamesInPaths(paths []string) ([]string, error) {
	files := make([]string, 0)

	for _, path := range paths {
		dir, err := os.Open(path)

		defer func() {
			err := dir.Close()
			if err != nil {
				log.Print(err)
			}
		}()

		if err != nil {
			log.Printf("failed opening directory: %s", err)

			return nil, err
		}

		list, err := dir.Readdirnames(0)
		if err != nil {
			log.Printf("failed reading plugins: %s", err)

			return nil, err
		}

		if len(list) == 0 {
			log.Printf("no plugins found in: %s", path)

			return files, nil
		}

		for _, name := range list {
			files = append(files, path+name)
		}
	}

	return files, nil
}

func isPlugin(fileName string) (Plugin, bool) {
	if !strings.HasSuffix(fileName, ".so") {
		return nil, false
	}

	// Load binary module
	goPlugin, err := plugin.Open(fileName)
	if err != nil {
		log.Println(err)

		return nil, false
	}

	// Look up exported "Plugin" symbol
	symPlugin, err := goPlugin.Lookup("Plugin")
	if err != nil {
		log.Println(err)

		return nil, false
	}

	// Assert that loaded symbol is of interface type Plugin
	p, ok := symPlugin.(Plugin)
	if !ok {
		log.Printf("unexpected type from module symbol in file %s", fileName)

		return nil, false
	}

	return p, true
}

func (p Plugins) loadAll(config Config, gofra *Gofra) error {
	fileList, err := getFileNamesInPaths(config.PluginPaths)
	if err != nil {
		return err
	}

	for _, f := range fileList {
		p.load(f, config, gofra)
	}

	return nil
}

func (p Plugins) load(fileName string, config Config, gofra *Gofra) bool {
	plugin, ok := isPlugin(fileName)
	if !ok {
		log.Printf("file %s does not contain a plugin", fileName)

		return false
	}

	p[plugin.Name()] = plugin

	InitPlugin(plugin, config, gofra)

	_, ok = plugin.(Runnable)
	if ok {
		go RunPlugin(plugin.Name(), plugin.(Runnable))
	}

	return true
}

// Improve comment
// Wrapper to prevent a plugin initialization error from bleeding into the bot engine
func InitPlugin(plugin Plugin, config Config, gofra *Gofra) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("init method of plugin %s failed: %s", plugin.Name(), err)
		}
	}()

	plugin.Init(config, gofra)
}

// Improve comment
// Wrapper to prevent a plugin execution error from bleeding into the bot engine
func RunPlugin(pluginName string, plugin Runnable) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Run method of plugin %s failed: %s", pluginName, err)
		}
	}()

	plugin.Run()
}
