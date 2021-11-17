package gofra

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"

	"mellium.im/sasl"
	"mellium.im/xmlstream"
	"mellium.im/xmpp"
	"mellium.im/xmpp/dial"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/mux"
	"mellium.im/xmpp/stanza"
)

type Gofra struct {
	config Config
	events Events
	plugins Plugins
	Client *xmpp.Session
	context context.Context
	Logger *log.Logger
	Debug *log.Logger
	mux *mux.ServeMux
}

type stanzaHandler struct{}

var gofra *Gofra
var initialized bool

func NewGofra(ctx context.Context, config Config, xmlIn, xmlOut *io.Writer, logger, debug *log.Logger) *Gofra {
	// Singleton
	if gofra != nil {
		return gofra
	}
	c, err := newXmppClient(ctx, config, xmlIn, xmlOut, logger, debug)
	if err != nil {
		log.Fatal(err.Error())
	}
	opts := []mux.Option{
		mux.Presence(stanza.AvailablePresence, xml.Name{}, stanzaHandler{}),
		mux.Message(stanza.ChatMessage, xml.Name{}, stanzaHandler{}),
		mux.Message(stanza.GroupChatMessage, xml.Name{}, stanzaHandler{}),
	}
	mux := mux.New("jabber:client", opts...)
	gofra = &Gofra{
		config: config,
		events: NewEvents(config),
		plugins: NewPlugins(config),
		Client: c,
		context: ctx,
		Logger: logger,
		Debug: debug,
		mux: mux,
	}
	return gofra
}

///////////////////// API ///////////////////////
type MessageBody struct {
	stanza.Message
	Body string `xml:"body"`
}

// Send function wrapper to make sending messages easier
func (g *Gofra) SendMessage(to, body string, msgType stanza.MessageType) error {
	j, err := jid.Parse(to)
	if err != nil {
		return err
	}
	msg := MessageBody{Message: stanza.Message{Type: msgType, To: j.Bare()}, Body: body}
	err = g.Client.Encode(g.context, msg)
	if err != nil {
		return err
	}

	start := msg.StartElement()

	d := xml.NewTokenDecoder(xmlstream.Token(start))
	if _, err := d.Token(); err != nil {
		return err
	}
	log.Println("GOFRA JUST BEFORE CALLING SEND IN SENDMESSAGE")
	err = g.Client.Send(g.context, d)
	log.Println("GOFRA JUST AFTER CALLING SEND IN SENDMESSAGE")
	return err
}

func (g *Gofra) EncodeMessage(to, body string, msgType stanza.MessageType, t xmlstream.TokenReadEncoder) error {
	j, err := jid.Parse(to)
	if err != nil {
		return err
	}
	msg := MessageBody{Message: stanza.Message{Type: msgType, To: j.Bare()}, Body: body}
	err = t.Encode(msg)
	if err != nil {
		return err
	}
	g.Logger.Println("BEFORE GOFRA's ENCODE")
/* 	err = g.Client.Encode(g.context, msg)
	g.Logger.Println("AFTER GOFRA's ENCODE", err) */
	return nil
}

func (g *Gofra) SendStanza(s interface{}) error {
	err := g.Client.Encode(g.context, s)
	return err
}

/*
Adds an event listener for a given event. Event listeners are executed in descending
priority order, so a higher priority grants earlier execution in the queue.
For accumulative handlers, that is, handlers that take the original set of values of
the event and pass on a modified set, there's the chain option. Handlers set to chain
are executed after all non-accumulative ones by descending priority order. Accumulated
event values are received through the event pointer argument where changes are expecteted
to be performed in order for the following chained handlers to recieve them. */
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
	g.Logger.Println("GOFRA GOT TIL HERE")
	return g.events.Publish(event)
}

func (g *Gofra) SetPriority(eventName, pluginName string, options Options) error {
	return g.events.SetPriority(eventName, pluginName, options)
}

/////////////////////////////////////////////////

func (g *Gofra) Init() error{
	// Initialize just once
	if initialized {
		return nil
	}

	initialized = true
	//Initialize plugins
	err := g.plugins.Init(g.config, g); if err != nil {
		return err
	}
	g.Publish(Event{Name: "initialized"})
	return nil
}

func (g *Gofra) Connect() error{
	// Send initial presence to let the server know we want to receive messages.
	err := gofra.Client.Send(gofra.context, stanza.Presence{Type: stanza.AvailablePresence}.Wrap(nil))
	if err != nil {
		return fmt.Errorf("error sending initial presence: %w", err)
	}

	g.Publish(Event{Name: "connected"})
	//g.SendMessage("vaulor@blastersklan.com", "Harooooo", stanza.ChatMessage)
	//return gofra.Client.Serve(xmpp.HandlerFunc(g.mux.HandleXMPP))
	return gofra.Client.Serve(xmpp.HandlerFunc(func(t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
		// This is a workaround for https://github.com/mellium/xmpp/issues/196
		// until a cleaner permanent fix is devised (see https://github.com/mellium/xmpp/issues/197)
		g.Debug.Println(g.Client.ConnectionState())
		st := start.Copy()
		// Ignore anything that's not a message. In a real system we'd want to at
		// least respond to IQs.
		if start.Name.Local == "message" {
			d := xml.NewTokenDecoder(xmlstream.MultiReader(xmlstream.Token(st), t))
			if _, err := d.Token(); err != nil {
				return err
			}

			msg := MessageBody{}
			err = d.DecodeElement(&msg, &st)
			if err != nil && err != io.EOF {
				g.Logger.Printf("Error decoding message: %q", err)
				return nil
			}

			if msg.Body == "" || msg.Type != stanza.ChatMessage {
				return nil
			}
			g.Debug.Println("THIS STANZA WAS A MESSAGE")
			e := Event{
				Name: "messageReceived",
				Payload: make(map[string]interface{}),
			}
			e.SetStanza(&msg)
			e.SetTokenReadEncoder(t)

			log.Println(gofra.Publish(e))
			return nil
		}
		if start.Name.Local == "presence" {
			g.Debug.Println("THIS STANZA WAS A PRESENCE")
		}
		return nil
	}))

} 

func (stanzaHandler) HandleMessage(msg stanza.Message, t xmlstream.TokenReadEncoder) error {
	start := msg.StartElement()

	d := xml.NewTokenDecoder(xmlstream.MultiReader(xmlstream.Token(&start), t))
	if _, err := d.Token(); err != nil {
		return err
	}

	msgStruct := MessageBody{}
	err := d.DecodeElement(&msgStruct, &start)
	if err != nil && err != io.EOF {
		gofra.Logger.Printf("Error decoding message: %q", err)
		return nil
	}

	if msgStruct.Body == "" || msgStruct.Type != stanza.ChatMessage {
		gofra.Logger.Printf("Message received has no body")
	}

	gofra.Logger.Printf("Message received: %v, with body: %q", msgStruct, msgStruct.Body)
	e := Event{
		Name: "messageReceived",
		Payload: make(map[string]interface{}),
	}
	e.SetStanza(msg)
	log.Println(gofra.Publish(e))
	return nil
}

func (stanzaHandler) HandlePresence(p stanza.Presence, t xmlstream.TokenReadEncoder) error {
	gofra.Logger.Printf("Presence received: %v", p)
	e := Event{
		Name: "presenceReceived",
		Payload: make(map[string]interface{}),
	}
	e.SetStanza(p)
	log.Println(gofra.Publish(e))
	return nil
}

/* func (stanzaHandler) HandleIQ(iq stanza.IQ, t xmlstream.TokenReadEncoder, start *xml.StartElement) error {
	return errFailTest
} */
/* // Entry point for presence stanzas
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
} */

type logWriter struct {
	logger *log.Logger
}

func (lw logWriter) Write(p []byte) (int, error) {
	lw.logger.Printf("%s", p)
	return len(p), nil
}

func newXmppClient(ctx context.Context, config Config, xmlIn, xmlOut *io.Writer, logger, debug *log.Logger) (*xmpp.Session, error){
	j, err := jid.Parse(config.Jid)
	if err != nil {
		return nil, fmt.Errorf("error parsing address %q: %w", config.Jid, err)
	}
	var d dial.Dialer
	d.NoLookup = true
	conn, err := d.Dial(ctx, "tcp", j)
	if err != nil {
		return nil, fmt.Errorf("error dialing sesion: %w", err)
	}

	s, err := xmpp.NewSession(ctx, j.Domain(), j, conn, 0, xmpp.NewNegotiator(func(*xmpp.Session, *xmpp.StreamConfig) xmpp.StreamConfig {
		return xmpp.StreamConfig{
			Lang: "en",
			Features: []xmpp.StreamFeature{
				xmpp.BindResource(),
				xmpp.StartTLS(&tls.Config{
					ServerName: j.Domain().String(),
					MinVersion: tls.VersionTLS12,
				}),
				xmpp.SASL("", config.Password, sasl.ScramSha1Plus, sasl.ScramSha1, sasl.Plain),
			},
			TeeIn:  logWriter{log.New(os.Stdout, "IN ", log.LstdFlags)},
			TeeOut: logWriter{log.New(os.Stdout, "OUT ", log.LstdFlags)},
		}
	}))
	if err != nil {
		return nil, fmt.Errorf("error establishing a session: %w", err)
	}
	return s, nil
}
