package events

import (
	"encoding/json"
	"time"
)

const (
	EventTypePostCreated = "POST_CREATED"
	EventTypePostDeleted = "POST_DELETED"
)

type Event struct {
	EventID   int64   `json:"event_id"`
	Type      string  `json:"type"`
	ActorID   string  `json:"actor_id"`
	Payload   Payload `json:"payload"`
	Timestamp int64   `json:"timestamp"`
}

type Payload struct {
	PostID  int64  `json:"post_id,omitempty"`
	Content string `json:"content,omitempty"`
}

func NewPostCreatedEvent(eventID, postID int64, authodID, content string) *Event {
	return &Event{
		EventID:   eventID,
		Type:      EventTypePostCreated,
		ActorID:   authodID,
		Payload:   Payload{PostID: postID, Content: content},
		Timestamp: time.Now().Unix(),
	}
}

func (e *Event) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

func Unmarshal(data []byte) (*Event, error) {
	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, err
	}
	return &event, nil
}
