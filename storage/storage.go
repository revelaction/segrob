package storage

import "github.com/revelaction/segrob/topic"

// Re-export interfaces for backend implementations
type TopicReader = topic.TopicReader
type TopicWriter = topic.TopicWriter
type TopicRepository = topic.TopicRepository
