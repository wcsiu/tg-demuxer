package entity

const (
	// MessageTypeVideo video message type
	MessageTypeVideo MessageType = iota + 1
	// MessageTypePhoto photo message type
	MessageTypePhoto
)

// MessageType message type
type MessageType int64

// Message message schema in db
type Message struct {
	ID           int64       `db:"id"`
	MessageType  MessageType `db:"message_type"`
	MessageID    int64       `db:"message_id"`
	ChatID       int64       `db:"chat_id"`
	MediaAlbumID int64       `db:"media_album_id"`
	Uploaded     bool        `db:"uploaded"`
	CreateAt     int64       `db:"created_at"`
	ModifiedAt   int64       `db:"modified_at"`
	PublishedAt  int32       `db:"published_at"`
}
