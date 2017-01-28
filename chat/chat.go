package chat

type Adapter interface {
	Run()
	Disconnect()
	SendMessage(msg *Message) error
}

type Message struct {
	Channel      string
	User         string
	Text         string
	Attachements []MessageAttachement

	// # Received message specifics
	// IsIM is for direct messages
	IsIM bool
	// IsGroup is for private channels
	IsGroup bool
}

type MessageAttachement struct {
	Header string
	Color  string
	Text   string
}
