package gofra

import (
	"log"
	"os"
	"plugin"
	"strings"
)

type Plugins map[string]Plugin


func (p Plugins)Init(config Config, gofra API) error{
	return p.loadAll(config, gofra)
}

func NewPlugins(config Config) Plugins {
	return make(Plugins)
}

func getPluginsInPaths(paths []string) ([]string, error) {
	plugins := make([]string, 0)
	for _, path := range paths {
		file, err := os.Open(path)
		defer func() {
			err := file.Close()
			if err != nil {
				log.Print(err)
			}
		}()
		if err != nil {
			log.Printf("failed opening directory: %s", err)
			return nil, err
		}
		
		list, err := file.Readdirnames(0)
		if err != nil {
			log.Printf("failed reading plugins: %s", err)
			return nil, err
		}
		if len(list) == 0 {
			log.Printf("no plugins found in: %s", path)
			return plugins, nil
		}
		for _, name := range list {
			_, ok := isPlugin(path + name); if ok {
				plugins = append(plugins, path + name)
			}
		}
	}
	return plugins, nil
}

func isPlugin(fileName string) (Plugin, bool) {
	if !strings.HasSuffix(fileName, ".so") {
		return nil, false
	}
	// load module
	// 1. open the so file to load the symbols
	plug, err := plugin.Open(fileName)
	if err != nil {
		log.Println(err)
		return nil, false
	}
	// 2. look up a symbol (an exported function or variable)
	// in this case, variable Plugin
	symPlugin, err := plug.Lookup("Plugin")
	if err != nil {
		log.Println(err)
		return nil, false
	}
	// 3. Assert that loaded symbol is of a desired type
	// in this case interface type Plugin (defined above)
	var botPlugin Plugin
	botPlugin, ok := symPlugin.(Plugin)
	if !ok {
		log.Printf("unexpected type from module symbol in file %s", fileName)
		return nil, false
	}
	log.Printf("file %s contains plugin %s", fileName,botPlugin.Name())
	return botPlugin, true
}

func (p Plugins)loadAll(config Config, gofra API) error {
	pluginList, err := getPluginsInPaths(config.Plugins_paths)
	if err != nil {
		return err
	}

	for _, plug := range pluginList {
		// Load the plugin
		plug, err := plugin.Open(plug)
		if err != nil {
			log.Println(err)
			return nil
		}
		// 2. look up a symbol (an exported function or variable)
		// in this case, variable Plugin
		symPlugin, err := plug.Lookup("Plugin")
		if err != nil {
			log.Println(err)
			return nil
		}
		gofraPlugin, ok := symPlugin.(Plugin)
		if !ok {
			log.Println("unexpected type from module symbol")
			return nil
		}
		
		p[gofraPlugin.Name()] = gofraPlugin

		safelyInit(gofraPlugin, config, gofra)

		_, ok = gofraPlugin.(Runnable)
		if ok {
			go safelyRun(gofraPlugin.Name(), gofraPlugin.(Runnable))
		}
	}
	return nil
}

// Wrapper to prevent a plugin initialization error from bleeding into the bot engine
func safelyInit(plugin Plugin, config Config, gofra API) {
	defer func() {
        if err := recover(); err != nil {
            log.Printf("init method of plugin %s failed: %s", plugin.Name(), err)
        }
    }()
	plugin.Init(config, gofra)
}

// Wrapper to prevent a plugin execution error from bleeding into the bot engine
func safelyRun(pluginName string, plugin Runnable) {
	defer func() {
        if err := recover(); err != nil {
			log.Printf("Run method of plugin %s failed: %s", pluginName, err)
        }
    }()
	plugin.Run()
}