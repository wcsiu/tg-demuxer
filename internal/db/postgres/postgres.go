package postgres

import (
	"fmt"
	"log"
	"time"

	"github.com/Arman92/go-tdlib"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/wcsiu/tg-demuxer/internal/config"
	"github.com/wcsiu/tg-demuxer/pkg/entity"
)

var db *sqlx.DB

// Connect connect to db.
func Connect() {
	var err error
	db, err = sqlx.Connect("postgres", fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=disable", config.C.DB.Host, config.C.DB.Port, config.C.DB.Name, config.C.DB.User, config.C.DB.Password))
	if err != nil {
		log.Fatal(err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal((err))
	}
}

// Close close db connection
func Close() {
	db.Close()
}

// InsertTGPhoto insert TG photo to db
func InsertTGPhoto(p *entity.Photo) error {
	if _, err := db.NamedExec(
		`INSERT INTO
			photos(caption,chat_id,message_id,photo_id,media_album_id,file_id,sender_user_id,is_downloading_active,is_downloading_completed,is_uploading_active,is_uploading_completed,file_path,created_at,modified_at,published_at)
		VALUES
			(:caption,:chat_id,:message_id,:photo_id,:media_album_id,:file_id,:sender_user_id,:is_downloading_active,:is_downloading_completed,:is_uploading_active,:is_uploading_completed,:file_path,:created_at,:modified_at,:published_at)`, &p); err != nil {
		return err
	}
	return nil
}

// UpdateTGPhoto update TG photo
func UpdateTGPhoto(f *tdlib.File, path string) error {
	if _, err := db.Exec(`
	UPDATE
		photos
	SET
		is_downloading_active=$1,
		is_downloading_completed=$2,
		is_uploading_active=$3,
		is_uploading_completed=$4,
		file_path=$5,
		modified_at=$6
	WHERE
		file_id=$7`,
		f.Local.IsDownloadingActive, f.Local.IsDownloadingCompleted, f.Remote.IsUploadingActive, f.Remote.IsUploadingCompleted, path, time.Now().Unix(), f.ID); err != nil {
		return err
	}
	return nil
}

// IfFileExists check if file exists in db
func IfFileExists(f *tdlib.File) (bool, error) {
	var ret bool
	if err := db.Get(&ret, "SELECT EXISTS(SELECT 1 FROM photos WHERE file_id=$1)", f.ID); err != nil {
		return false, err
	}
	return ret, nil
}

// GetTGPhotoByFileID get file ID by file id
func GetTGPhotoByFileID(fileID int32) (*entity.Photo, error) {
	var ret entity.Photo
	if err := db.Get(&ret, "SELECT * FROM photos WHERE file_id=$1", fileID); err != nil {
		return nil, err
	}
	return &ret, nil
}

// InsertTGVideo insert TG video to db.
func InsertTGVideo(v *entity.Video) error {
	if _, err := db.NamedExec(
		`INSERT INTO
			videos(caption,chat_id,message_id,media_album_id,file_id,mime_type,sender_user_id,is_downloading_active,is_downloading_completed,is_uploading_active,is_uploading_completed,file_path,created_at,modified_at,published_at)
		VALUES
			(:caption,:chat_id,:message_id,:media_album_id,:file_id,:mime_type,:sender_user_id,:is_downloading_active,:is_downloading_completed,:is_uploading_active,:is_uploading_completed,:file_path,:created_at,:modified_at,:published_at)`, &v); err != nil {
		return err
	}
	return nil
}

// GetTGVideoByFileID get file ID by file id
func GetTGVideoByFileID(fileID int32) (*entity.Video, error) {
	var ret entity.Video
	if err := db.Get(&ret, "SELECT * FROM videos WHERE file_id=$1", fileID); err != nil {
		return nil, err
	}
	return &ret, nil
}

// UpdateTGVideo update TG video
func UpdateTGVideo(f *tdlib.File, path string) error {
	if _, err := db.Exec(`
	UPDATE
		videos
	SET
		is_downloading_active=$1,
		is_downloading_completed=$2,
		is_uploading_active=$3,
		is_uploading_completed=$4,
		file_path=$5,
		modified_at=$6
	WHERE
		file_id=$7`,
		f.Local.IsDownloadingActive, f.Local.IsDownloadingCompleted, f.Remote.IsUploadingActive, f.Remote.IsUploadingCompleted, path, time.Now().Unix(), f.ID); err != nil {
		return err
	}
	return nil
}

// InsertChat insert new chat to db
func InsertChat(c *entity.Chat) error {
	if _, err := db.NamedExec(
		`INSERT INTO
			chats(title,chat_id,created_at,modified_at)
		VALUES
			(:title,:chat_id,:created_at,:modified_at)`, &c); err != nil {
		return err
	}
	return nil
}

// InsertMessage insert new message to db
func InsertMessage(m *entity.Message) error {
	if _, err := db.NamedExec(
		`INSERT INTO
			messages(message_type,message_id,chat_id,media_album_id,uploaded,created_at,modified_at,published_at)
		VALUES
			(:message_type,:message_id,:chat_id,:media_album_id,:uploaded,:created_at,:modified_at,:published_at)`, &m); err != nil {
		return err
	}
	return nil
}

// GetMessageByMessageID get message by message id and chat id
func GetMessageByMessageID(m *entity.Message, MessageID, ChatID int64) error {
	if err := db.Get(m, "SELECT * FROM messages WHERE message_id=$1 AND chat_id=$2", MessageID, ChatID); err != nil {
		return err
	}
	return nil
}

// UpdateMessageUploadStatus update message upload status
func UpdateMessageUploadStatus(m *entity.Message, uploaded bool) error {
	if _, err := db.Exec(`
	UPDATE
		messages
	SET
		uploaded=$1,
		modified_at=$2
	WHERE
		chat_id=$3
		AND
		message_id=$4
		AND
		modified_at=$5`, uploaded, time.Now().Unix(), m.ChatID, m.MessageID, m.ModifiedAt); err != nil {
		return err
	}
	return nil
}

// GetMaxMsgIDInChat get max message id in chat
func GetMaxMsgIDInChat(chatID int64) (int64, error) {
	var ret int64
	if err := db.Get(&ret, "SELECT MAX(message_id) FROM messages WHERE chat_id=$1", chatID); err != nil {
		return ret, err
	}
	return ret, nil
}
