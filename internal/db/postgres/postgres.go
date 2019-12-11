package postgres

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/wcsiu/go-tdlib"
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

	// migration
	var schema = `
	CREATE TABLE IF NOT EXISTS photos (
		id SERIAL NOT NULL PRIMARY KEY,
		image_hash BYTEA,
		caption TEXT NOT NULL,
		chat_id BIGINT NOT NULL,
		message_id BIGINT NOT NULL,
		photo_id BIGINT NOT NULL,
		media_album_id BIGINT NOT NULL,
		file_id BIGINT NOT NULL,
		sender_user_id BIGINT NOT NULL,
		is_downloading_active BOOLEAN NOT NULL,
		is_downloading_completed BOOLEAN NOT NULL,
		is_uploading_active BOOLEAN NOT NULL,
		is_uploading_completed BOOLEAN NOT NULL,
		file_path TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		published_at INTEGER NOT NULL
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_image_hash ON photos (image_hash);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_message_id ON photos (chat_id, message_id);
	CREATE INDEX IF NOT EXISTS idx_published_at ON photos (published_at);
	`
	db.MustExec(schema)
}

//Close close db connection
func Close() {
	db.Close()
}

//InsertTGPhoto insert TG photo to db
func InsertTGPhoto(p *entity.Photo) error {
	if _, err := db.NamedExec(
		`INSERT INTO
			photos(caption,chat_id,message_id,photo_id,media_album_id,file_id,sender_user_id,is_downloading_active,is_downloading_completed,is_uploading_active,is_uploading_completed,file_path,created_at,published_at)
		VALUES
			(:caption,:chat_id,:message_id,:photo_id,:media_album_id,:file_id,:sender_user_id,:is_downloading_active,:is_downloading_completed,:is_uploading_active,:is_uploading_completed,:file_path,:created_at,:published_at)`, &p); err != nil {
		return err
	}
	return nil
}

//UpdateTGPhoto update TG photos
func UpdateTGPhoto(f *tdlib.File) error {
	var photo = entity.Photo{FileID: f.ID, IsDownloadingActive: f.Local.IsDownloadingActive, IsDownloadingCompleted: f.Local.IsDownloadingCompleted, IsUploadingActive: f.Remote.IsUploadingActive, IsUploadingCompleted: f.Remote.IsUploadingCompleted, FilePath: f.Local.Path}
	if _, err := db.NamedExec("UPDATE photos SET is_downloading_active=:is_downloading_active,is_downloading_completed=:is_downloading_completed,is_uploading_active=:is_uploading_active,is_uploading_completed=:is_uploading_completed,file_path=:file_path WHERE file_id=:file_id", &photo); err != nil {
		return err
	}
	return nil
}

//IfFileExists check if file exists in db
func IfFileExists(f *tdlib.File) (bool, error) {
	var ret bool
	if err := db.Get(&ret, "SELECT EXISTS(SELECT 1 FROM photos WHERE file_id=$1)", f.ID); err != nil {
		return false, err
	}
	return ret, nil
}

//GetTGPhotoByFileID get file ID by file id
func GetTGPhotoByFileID(fileID int32) (*entity.Photo, error) {
	var ret entity.Photo
	if err := db.Get(&ret, "SELECT * FROM photos WHERE file_id=$1", fileID); err != nil {
		return nil, err
	}
	return &ret, nil
}
