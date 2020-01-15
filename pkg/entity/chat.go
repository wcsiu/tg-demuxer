package entity

// Chat chat schema in db
type Chat struct {
	ID         int64  `db:"id"`
	ChatID     int64  `db:"chat_id"`
	Title      string `db:"title"`
	CreatedAt  int64  `db:"created_at"`
	ModifiedAt int64  `db:"modified_at"`
}
