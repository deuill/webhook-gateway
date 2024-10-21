// Copyright 2016 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package stanza

import (
	"encoding/xml"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal/ns"
	"mellium.im/xmpp/jid"
)

// Message is an XMPP stanza that contains a payload for direct one-to-one
// communication with another network entity. It is often used for sending chat
// messages to an individual or group chat server, or for notifications and
// alerts that don't require a response.
type Message struct {
	XMLName xml.Name    `xml:"message"`
	ID      string      `xml:"id,attr,omitempty"`
	To      jid.JID     `xml:"to,attr,omitempty"`
	From    jid.JID     `xml:"from,attr,omitempty"`
	Lang    string      `xml:"http://www.w3.org/XML/1998/namespace lang,attr,omitempty"`
	Type    MessageType `xml:"type,attr,omitempty"`
}

// UnmarshalXMLAttr converts the provided XML attribute to a valid message type.
// Falls back to normal message type, if an unknown, or empty value is passed.
func (t *MessageType) UnmarshalXMLAttr(attr xml.Attr) error {
	switch attr.Value {
	case "normal":
		*t = NormalMessage
	case "chat":
		*t = ChatMessage
	case "error":
		*t = ErrorMessage
	case "groupchat":
		*t = GroupChatMessage
	case "headline":
		*t = HeadlineMessage
	default:
		*t = NormalMessage
	}
	return nil
}

// NewMessage unmarshals an XML token into a Message.
func NewMessage(start xml.StartElement) (Message, error) {
	v := Message{
		XMLName: start.Name,
		Type:    "normal",
	}
	for _, attr := range start.Attr {
		if attr.Name.Local == "lang" && attr.Name.Space == ns.XML {
			v.Lang = attr.Value
			continue
		}
		if attr.Name.Space != "" && attr.Name.Space != start.Name.Space {
			continue
		}

		var err error
		switch attr.Name.Local {
		case "id":
			v.ID = attr.Value
		case "to":
			if attr.Value != "" {
				v.To, err = jid.Parse(attr.Value)
				if err != nil {
					return v, err
				}
			}
		case "from":
			if attr.Value != "" {
				v.From, err = jid.Parse(attr.Value)
				if err != nil {
					return v, err
				}
			}
		case "type":
			err = (&v.Type).UnmarshalXMLAttr(attr)
			if err != nil {
				return v, err
			}
		}
	}
	return v, nil
}

// StartElement converts the Message into an XML token.
func (msg Message) StartElement() xml.StartElement {
	// Keep whatever namespace we're already using but make sure the localname is
	// "message".
	name := msg.XMLName
	name.Local = "message"

	attr := make([]xml.Attr, 0, 5)
	attr = append(attr, xml.Attr{Name: xml.Name{Local: "type"}, Value: string(msg.Type)})
	if !msg.To.Equal(jid.JID{}) {
		attr = append(attr, xml.Attr{Name: xml.Name{Local: "to"}, Value: msg.To.String()})
	}
	if !msg.From.Equal(jid.JID{}) {
		attr = append(attr, xml.Attr{Name: xml.Name{Local: "from"}, Value: msg.From.String()})
	}
	if msg.ID != "" {
		attr = append(attr, xml.Attr{Name: xml.Name{Local: "id"}, Value: msg.ID})
	}
	if msg.Lang != "" {
		attr = append(attr, xml.Attr{Name: xml.Name{Space: ns.XML, Local: "lang"}, Value: msg.Lang})
	}

	return xml.StartElement{
		Name: name,
		Attr: attr,
	}
}

// Wrap wraps the payload in a stanza.
func (msg Message) Wrap(payload xml.TokenReader) xml.TokenReader {
	return xmlstream.Wrap(payload, msg.StartElement())
}

// Error returns a token reader that wraps the provided Error in a message
// stanza with the to and from attributes switched and the type set to
// ErrorMessage.
func (msg Message) Error(err Error) xml.TokenReader {
	msg.Type = ErrorMessage
	msg.From, msg.To = msg.To, msg.From
	return msg.Wrap(err.TokenReader())
}

// MessageType is the type of a message stanza.
// It should normally be one of the constants defined in this package.
type MessageType string

const (
	// NormalMessage is a standalone message that is sent outside the context of a
	// one-to-one conversation or groupchat, and to which it is expected that the
	// recipient will reply. Typically a receiving client will present a message
	// of type "normal" in an interface that enables the recipient to reply, but
	// without a conversation history.
	NormalMessage MessageType = "normal"

	// ChatMessage represents a message sent in the context of a one-to-one chat
	// session.  Typically an interactive client will present a message of type
	// "chat" in an interface that enables one-to-one chat between the two
	// parties, including an appropriate conversation history.
	ChatMessage MessageType = "chat"

	// ErrorMessage is generated by an entity that experiences an error when
	// processing a message received from another entity.
	ErrorMessage MessageType = "error"

	// GroupChatMessage is sent in the context of a multi-user chat environment.
	// Typically a receiving client will present a message of type "groupchat" in
	// an interface that enables many-to-many chat between the parties, including
	// a roster of parties in the chatroom and an appropriate conversation
	// history.
	GroupChatMessage MessageType = "groupchat"

	// HeadlineMessage provides an alert, a notification, or other transient
	// information to which no reply is expected (e.g., news headlines, sports
	// updates, near-real-time market data, or syndicated content). Because no
	// reply to the message is expected, typically a receiving client will present
	// a message of type "headline" in an interface that appropriately
	// differentiates the message from standalone messages, chat messages, and
	// groupchat messages (e.g., by not providing the recipient with the ability
	// to reply).
	HeadlineMessage MessageType = "headline"
)

// MarshalText ensures that the default value for MessageType is marshaled to XML as a
// valid normal Message type, as per RFC 6121 § 5.2.2
// It satisfies the encoding.TextMarshaler interface for MessageType.
func (t MessageType) MarshalText() ([]byte, error) {
	if t != NormalMessage &&
		t != ChatMessage &&
		t != ErrorMessage &&
		t != GroupChatMessage &&
		t != HeadlineMessage {
		t = NormalMessage
	}
	return []byte(t), nil
}