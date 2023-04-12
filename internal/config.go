package gofra

type Config struct {
	Password    string                            `yaml:"password"`
	PluginPaths []string                          `yaml:"pluginPaths"`
	Jid         string                            `yaml:"jid"`
	Nick        string                            `yaml:"nick"`
	LogXML      bool                              `yaml:"logXML"`
	Debug       bool                              `yaml:"debug"`
	SkipSRV     bool                              `yaml:"skipSRV"`
	MUCs        []MUCConfig                       `yaml:"mucs"`
	Plugins     map[string]map[string]interface{} `yaml:"plugins"`
}

// Per-MUC configuration
type MUCConfig struct {
	Nick        string `yaml:"mucNick"`
	JoinHistory int    `yaml:"mucJoinHistory"`
	Jid         string `yaml:"mucJid"`
	Password    string `yaml:"mucPasword"`
}
