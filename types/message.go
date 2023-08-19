package types

import (
	"strings"
	"time"
)

type MessageType string

const (
	Notification MessageType = "notification"
	Word         MessageType = "word"
)

type Message struct {
	Type      MessageType `json:"type"`
	Content   string      `json:"content"    bson:"content"`
	Owner     string      `json:"owner"      bson:"owner"`
	CreatedAt time.Time   `json:"created_at" bson:"created_at"`
}

func NewWord(content string, owner string) *Message {
	return &Message{
		Type:      Word,
		Content:   content,
		Owner:     owner,
		CreatedAt: time.Now(),
	}
}

func (m *Message) ValidateWord() string {
	contentLength := len(strings.Split(m.Content, " "))
	if contentLength != 1 && m.Type == Word {
		return "Invalid content, must be a single word"
	}
	return ""
}

func NewNotification(content string, owner string) *Message {
	return &Message{
		Type:      Notification,
		Content:   content,
		Owner:     owner,
		CreatedAt: time.Now(),
	}
}

func (m *Message) IsQuiting() bool {
	return m.Type == Notification && m.Content == "exit"
}
