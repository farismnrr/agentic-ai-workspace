package application

import (
	"sync"
	"time"
)

const subscriberBuffer = 128

// Update is a wake-up call for someone following a task or the whole workspace.
// A Delta carries streamed text; an update without one means "state changed, look again".
type Update struct {
	TaskID string
	Delta  string
}

// Subscription receives updates for one topic until it is closed.
type Subscription struct {
	C     chan Update
	hub   *Hub
	topic string
}

// Close stops delivery and releases the subscription.
func (s *Subscription) Close() {
	s.hub.remove(s)
}

// Hub fans updates out to everyone following a topic. Delivery never blocks the
// agent: a slow follower simply misses wake-ups, and re-reads state on the next one.
type Hub struct {
	topics map[string]map[*Subscription]struct{}
	mu     sync.Mutex
	closed bool
}

// NewHub builds an empty hub.
func NewHub() *Hub {
	return &Hub{topics: map[string]map[*Subscription]struct{}{}}
}

// TaskTopic is the topic of one task's thread.
func TaskTopic(taskID string) string { return "task:" + taskID }

// UserTopic is the topic of everything a person owns.
func UserTopic(userID string) string { return "user:" + userID }

// Subscribe starts following a topic.
func (h *Hub) Subscribe(topic string) *Subscription {
	h.mu.Lock()
	defer h.mu.Unlock()

	sub := &Subscription{C: make(chan Update, subscriberBuffer), hub: h, topic: topic}
	if h.closed {
		close(sub.C)

		return sub
	}

	if h.topics[topic] == nil {
		h.topics[topic] = map[*Subscription]struct{}{}
	}

	h.topics[topic][sub] = struct{}{}

	return sub
}

// Publish delivers an update to everyone following the topic.
func (h *Hub) Publish(topic string, update Update) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for sub := range h.topics[topic] {
		select {
		case sub.C <- update:
		default:
		}
	}
}

// Close ends every subscription, so streams can finish when the app stops.
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.closed = true

	for _, subs := range h.topics {
		for sub := range subs {
			close(sub.C)
		}
	}

	h.topics = map[string]map[*Subscription]struct{}{}
}

func (h *Hub) remove(sub *Subscription) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if subs, ok := h.topics[sub.topic]; ok {
		if _, present := subs[sub]; present {
			delete(subs, sub)
			close(sub.C)
		}
	}
}

// Changed implements Notifier: a task's state moved on.
func (h *Hub) Changed(userID, taskID string) {
	update := Update{TaskID: taskID}
	h.Publish(TaskTopic(taskID), update)
	h.Publish(UserTopic(userID), update)
}

// ProjectTopic is the topic of the files of one project.
func ProjectTopic(projectID string) string { return "project:" + projectID }

// FilesChanged implements Notifier: whoever has the project's files open should read them again.
func (h *Hub) FilesChanged(projectID string) {
	h.Publish(ProjectTopic(projectID), Update{})
}

// Delta implements Notifier: the model produced more text.
func (h *Hub) Delta(taskID, text string) {
	h.Publish(TaskTopic(taskID), Update{TaskID: taskID, Delta: text})
}

// HeartbeatInterval is how often idle streams are touched so proxies keep them open.
const HeartbeatInterval = 15 * time.Second
