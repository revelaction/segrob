package storage

import "github.com/revelaction/segrob/topic"

// TopicReader defines read operations for topic storage
type TopicReader interface {
	// All returns all topics from storage
	All() ([]topic.Topic, error)

	// Topic returns a single topic by name
	Topic(name string) (topic.Topic, error)

	// Names returns the names of all available topics
	Names() ([]string, error)
}

// TopicWriter defines write operations for topic storage
type TopicWriter interface {
	// Write persists a topic to storage
	Write(tp topic.Topic) error
}

// TopicRepository combines read and write operations
type TopicRepository interface {
	TopicReader
	TopicWriter
}
