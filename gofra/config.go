package gofra

type Config struct {
	ServerURL   string                            `yaml:"serverUrl"`
	ServerPort  string                            `yaml:"serverPort"`
	Password    string                            `yaml:"password"`
	PluginPaths []string                          `yaml:"pluginPaths"`
	Jid         string                            `yaml:"jid"`
	Nick        string                            `yaml:"nick"`
	LogXML      bool                              `yaml:"logXML"`
	Debug       bool                              `yaml:"debug"`
	MUCs        []MUCConfig                       `yaml:"mucs"`
	Plugins     map[string]map[string]interface{} `yaml:"plugins"`
	Extra       map[string]interface{}            `yaml:"extra"`
}

// Per-MUC configuration
type MUCConfig struct {
	Nick        string `yaml:"mucNick"`
	JoinHistory int    `yaml:"mucJoinHistory"`
	Jid         string `yaml:"mucJid"`
	Password    string `yaml:"mucPasword"`
}
