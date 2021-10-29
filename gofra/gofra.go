package gofra

import (
	"fmt"
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

///////////////////// API ////////////////////

func (g *Gofra) Send(to, message string, msgType stanza.StanzaType) error {
	reply := stanza.Message{Attrs: stanza.Attrs{To: to, Type: msgType}, Body: message}
	err := g.client.Send(reply)
	return err
}

func (g *Gofra) SendStanza(s stanza.Packet) error {
	err := g.client.Send(s)
	return err
}

func (g *Gofra) Subscribe(eventName, pluginName string, handler Handler, options Options) {
	fmt.Println("Plugin "+pluginName+" subscribed to event "+eventName)
	g.events.Subscribe(eventName, pluginName, handler, options)
}

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

//////////////// INTERNAL ////////////////

func (g *Gofra) Init() error{
	if initialized {return nil}
	initialized = true
	//Initialize plugins
	err := g.plugins.Init(g.config, g); if err != nil {return err}
	g.Publish(Event{Name: "initialized"})
	//Connect XMPP client
	err = g.Connect(); if err != nil {return err}
	g.Publish(Event{Name: "connected"})
	// If you pass the client to a connection manager, it will handle the reconnect policy
	// for you automatically.
	cm := xmpp.NewStreamManager(g.client, nil)
	fmt.Println(cm)
	log.Fatal(cm.Run())
	return nil
}

func (g *Gofra) Connect() error{
	//Connect
	err := g.client.Connect()
	if err != nil {
		log.Fatalf("%+v", err)
		return err
	}
	return nil
}

func handlePresence(s xmpp.Sender, p stanza.Packet) {
	pres, ok := p.(stanza.Presence)
	if !ok {
		_, _ = fmt.Fprintf(os.Stdout, "Ignoring packet: %T\n", p)
		return
	} 
	_, _ = fmt.Fprintf(os.Stdout, "Body = %s - from = %s\n", pres.Name(), pres.From)
	fmt.Println(gofra.Publish(
		Event{
			Name: "presenceReceived",
			Payload: make(map[string]interface{}),
			Stanza: p,
		}))
}

func handleMessage(s xmpp.Sender, p stanza.Packet) {
	msg, ok := p.(stanza.Message)
	if !ok {
		_, _ = fmt.Fprintf(os.Stdout, "Ignoring packet: %T\n", p)
		return
	}
	gofra.Publish(
		Event{
			Name: "messageReceived",
			Payload: make(map[string]interface{}),
			Stanza: p,
	})
	_, _ = fmt.Fprintf(os.Stdout, "Body = %s - from = %s\n", msg.Body, msg.From)
}

/* client := xmpp.Config{
	TransportConfiguration: xmpp.TransportConfiguration{
		Address: "blastersklan.com:5222",
	},
	Jid:          "golang@blastersklan.com",
	Credential:   xmpp.Password("1234"),
	StreamLogger: os.Stdout,
	Insecure:     true,
	// TLSConfig: tls.Config{InsecureSkipVerify: true},
} */

func newXmppClient(config Config) *xmpp.Client {
	xmppConfig := xmpp.Config{
		TransportConfiguration: xmpp.TransportConfiguration{
			Address: config.ServerURL + ":" + config.ServerPort,
		},
		Jid:          config.Jid,
		Credential:   xmpp.Password(config.Password),
		StreamLogger: os.Stdout,
		Insecure:     true,
		// TLSConfig: tls.Config{InsecureSkipVerify: true},
	}
	router := xmpp.NewRouter()
	router.HandleFunc("presence", handlePresence)
	router.HandleFunc("message", handleMessage)
	var err error
	client, err := xmpp.NewClient(&xmppConfig, router, errorHandler)
	if err != nil {
		log.Fatalf("%+v", err)
	}
	return client
}

func errorHandler(err error) {
	fmt.Println(err.Error())
}