package gofra

import (
	"log"
	"os"

	"gosrc.io/xmpp"
	"gosrc.io/xmpp/stanza"
)

type Gofra struct {
	config Config
	events Events
	plugins Plugins
	client *xmpp.Client
}

var gofra *Gofra
var initialized bool

func NewGofra(config Config) *Gofra {
	// Singleton
	if gofra != nil {
		return gofra
	}
	gofra = &Gofra{
		config: config,
		events: NewEvents(config),
		plugins: NewPlugins(config),
		client: newXmppClient(config),
	}
	return gofra
}

///////////////////// API ///////////////////////

// Send function wrapper to make sending messages easier
func (g *Gofra) Send(to, message string, msgType stanza.StanzaType) error {
	reply := stanza.Message{Attrs: stanza.Attrs{To: to, Type: msgType}, Body: message}
	err := g.client.Send(reply)
	return err
}

func (g *Gofra) SendStanza(s stanza.Packet) error {
	err := g.client.Send(s)
	return err
}

// Adds an event listener for a given event. Event listeners are executed in descending
// priority order, so a higher priority grants earlier execution in the queue.
// For accumulative handlers, that is, handlers that take the original set of values of
// the event and pass on a modified set, there's the chain option. Handlers set to chain
// are executed after all non-accumulative ones by descending priority order. Accumulated
// event values are received through the event pointer argument where changes are expecteted
// to be performed in order for the following chained handlers to recieve them.
func (g *Gofra) Subscribe(eventName, pluginName string, handler Handler, options Options) {
	log.Println("Plugin "+pluginName+" subscribed to event "+eventName)
	g.events.Subscribe(eventName, pluginName, handler, options)
}

// Executes all event handlers subscribed to a particular event
func (g *Gofra) Publish(event Event) Reply{
	defer func() {
        if err := recover(); err != nil {
            log.Println("handler failed:", err)
        }
    }()
	return g.events.Publish(event)
}

func (g *Gofra) SetPriority(eventName, pluginName string, options Options) error {
	return g.events.SetPriority(eventName, pluginName, options)
}

/////////////////////////////////////////////////

func (g *Gofra) Init() error{
	// Initialize just once
	if initialized {return nil}
	initialized = true

	//Initialize plugins
	err := g.plugins.Init(g.config, g); if err != nil {return err}
	g.Publish(Event{Name: "initialized"})
	return nil
}

func (g *Gofra) Connect() error{
	//Connect XMPP client
	err := g.client.Connect()
	if err != nil {
		log.Fatalf("%+v", err)
		return err
	}
	g.Publish(Event{Name: "connected"})

	// Connection manager handles reconnect policy automatically.
	cm := xmpp.NewStreamManager(g.client, nil)
	log.Println(cm)
	//log.Fatal(cm.Run())
	return nil
}

// Entry point for presence stanzas
func handlePresence(s xmpp.Sender, p stanza.Packet) {
	pres, ok := p.(stanza.Presence)
	if !ok {
		log.Printf("Ignoring packet: %T\n", p)
		return
	} 
	log.Printf("Body = %s - from = %s\n", pres.Name(), pres.From)
	log.Println(gofra.Publish(
		Event{
			Name: "presenceReceived",
			Payload: make(map[string]interface{}),
			Stanza: p,
		}))
}

// Entry point for message stanzas
func handleMessage(s xmpp.Sender, p stanza.Packet) {
	msg, ok := p.(stanza.Message)
	if !ok {
		log.Printf("Ignoring packet: %T\n", p)
		return
	}

	gofra.Publish(
		Event{
			Name: "messageReceived",
			Payload: make(map[string]interface{}),
			Stanza: p,
	})
	log.Printf("Body = %s - from = %s\n", msg.Body, msg.From)
}

func newXmppClient(config Config) *xmpp.Client {
	xmppConfig := xmpp.Config{
		TransportConfiguration: xmpp.TransportConfiguration{
			Address: config.ServerURL + ":" + config.ServerPort,
		},
		Jid:          config.Jid,
		Credential:   xmpp.Password(config.Password),
		StreamLogger: os.Stdout,
	}

	router := xmpp.NewRouter()
	router.HandleFunc("presence", handlePresence)
	router.HandleFunc("message", handleMessage)

	client, err := xmpp.NewClient(&xmppConfig, router, errorHandler)
	if err != nil {
		log.Fatalf("%+v", err)
	}
	return client
}

func errorHandler(err error) {
	log.Println(err.Error())
}