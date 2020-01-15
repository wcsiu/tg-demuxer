package entity

// Video videos schema in db.
type Video struct {
	ID                     int64  `db:"id"`
	Caption                string `db:"caption"`
	ChatID                 int64  `db:"chat_id"`
	MessageID              int64  `db:"message_id"`
	MediaAlbumID           int64  `db:"media_album_id"`
	FileID                 int32  `db:"file_id"`
	MIMEType               string `db:"mime_type"`
	SenderUserID           int32  `db:"sender_user_id"`
	IsDownloadingActive    bool   `db:"is_downloading_active"`
	IsDownloadingCompleted bool   `db:"is_downloading_completed"`
	IsUploadingActive      bool   `db:"is_uploading_active"`
	IsUploadingCompleted   bool   `db:"is_uploading_completed"`
	FilePath               string `db:"file_path"`
	CreatedAt              int64  `db:"created_at"`
	ModifiedAt             int64  `db:"modified_at"`
	PublishedAt            int32  `db:"published_at"`
}
