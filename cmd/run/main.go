package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/wcsiu/go-tdlib"
	"github.com/wcsiu/tg-demuxer/internal/config"
	"github.com/wcsiu/tg-demuxer/internal/db/postgres"
	"github.com/wcsiu/tg-demuxer/pkg/entity"
)

var allChats []*tdlib.Chat

func main() {
	tdlib.SetLogVerbosityLevel(1)
	// tdlib.SetFilePath("./dev/errors.txt")

	var path = flag.String("path", "/configs/configs.yml", "configuration location")

	// load configs.
	if err := config.Load(*path); err != nil {
		panic(err)
	}

	// connect to database.
	postgres.Connect()
	defer postgres.Close()

	// Create new instance of client
	client := tdlib.NewClient(tdlib.Config{
		APIID:               config.C.TG.APIID,
		APIHash:             config.C.TG.APIHash,
		SystemLanguageCode:  "en",
		DeviceModel:         "Server",
		SystemVersion:       "1.0.0",
		ApplicationVersion:  "1.0.0",
		UseMessageDatabase:  false,
		UseFileDatabase:     false,
		UseChatInfoDatabase: false,
		UseTestDataCenter:   false,
		DatabaseDirectory:   "./dev/tdlib-db",
		FileDirectory:       "./dev/tdlib-files",
		IgnoreFileNames:     false,
	})

	// Handle Ctrl+C , Gracefully exit and shutdown tdlib
	ch := make(chan os.Signal, 2)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ch
		client.DestroyInstance()
		os.Exit(1)
	}()

	go func() {
		for {
			currentState, _ := client.Authorize()
			if currentState.GetAuthorizationStateEnum() == tdlib.AuthorizationStateWaitPhoneNumberType {
				fmt.Print("Enter phone: ")
				var number string
				fmt.Scanln(&number)
				_, err := client.SendPhoneNumber(number)
				if err != nil {
					fmt.Printf("Error sending phone number: %v\n", err)
				}
			} else if currentState.GetAuthorizationStateEnum() == tdlib.AuthorizationStateWaitCodeType {
				fmt.Print("Enter code: ")
				var code string
				fmt.Scanln(&code)
				_, err := client.SendAuthCode(code)
				if err != nil {
					fmt.Printf("Error sending auth code : %v\n", err)
				}
			} else if currentState.GetAuthorizationStateEnum() == tdlib.AuthorizationStateWaitPasswordType {
				fmt.Print("Enter Password: ")
				var password string
				fmt.Scanln(&password)
				_, err := client.SendAuthPassword(password)
				if err != nil {
					fmt.Printf("Error sending auth password: %v\n", err)
				}
			} else if currentState.GetAuthorizationStateEnum() == tdlib.AuthorizationStateReadyType {
				fmt.Println("Authorization Ready! Let's rock")
				break
			}
		}
	}()

	// Wait while we get AuthorizationReady!
	// Note: See authorization example for complete auhtorization sequence example
	currentState, _ := client.Authorize()
	for ; currentState.GetAuthorizationStateEnum() != tdlib.AuthorizationStateReadyType; currentState, _ = client.Authorize() {
		time.Sleep(300 * time.Millisecond)
	}

	go addUpdateFileMessageFitler(client)

	// update chat list
	if err := updateChatList(client); err != nil {
		fmt.Printf("fail to update chat list, err: %+v\n", err)
		return
	}
	fmt.Printf("got %d chats\n", len(allChats))

	for _, chat := range allChats {
		if err := retrieveAllPreviousPhotosFromChat(client, chat.ID); err != nil {
			log.Println("Error fail to retrieve all previous photos, error: ", err)
		}
	}

	for {
		time.Sleep(1 * time.Second)
	}
}

// see https://stackoverflow.com/questions/37782348/how-to-use-getchats-in-tdlib
func updateChatList(client *tdlib.Client) error {
	// need to call getChats to retrieve chats first into tdlib first.
	if _, getChatsErr := client.GetChats(tdlib.JSONInt64(int64(math.MaxInt64)), 0, 1000); getChatsErr != nil {
		return getChatsErr
	}
	allChats = nil
	for i := range config.C.ChatList {
		var chat, getChatErr = client.GetChat(config.C.ChatList[i])
		if getChatErr != nil {
			return getChatErr
		}
		allChats = append(allChats, chat)
	}

	return nil
}

func retrieveAllPreviousPhotosFromChat(client *tdlib.Client, chatID int64) error {
	var msgs *tdlib.Messages
	var lastMessageID = int64(0)

	for lastMessageID == 0 || msgs.TotalCount > 0 {
		var getMsgsErr error
		msgs, getMsgsErr = client.GetChatHistory(chatID, lastMessageID, 0, 10, false)
		if getMsgsErr != nil {
			log.Println("ERROR: fail to get messages from chat: ", chatID, ", error: ", getMsgsErr)
		}
		for _, v := range msgs.Messages {
			switch v.Content.GetMessageContentEnum() {
			case tdlib.MessagePhotoType:
				var p, photoCastOK = v.Content.(*tdlib.MessagePhoto)
				if !photoCastOK {
					log.Println("ERROR: fail to type cast to MessagePhoto")
					continue
				}
				var largest = getLargestResolution(p.Photo.Sizes)
				var f, downloadErr = client.DownloadFile(largest.Photo.ID, 32)
				if downloadErr != nil {
					return downloadErr
				}
				if err := postgres.InsertTGPhoto(&entity.Photo{
					Caption:                p.Caption.Text,
					ChatID:                 chatID,
					MessageID:              v.ID,
					PhotoID:                int64(p.Photo.ID),
					MediaAlbumID:           int64(v.MediaAlbumID),
					FileID:                 largest.Photo.ID,
					SendUserID:             v.SenderUserID,
					IsDownloadingActive:    f.Local.IsDownloadingActive,
					IsDownloadingCompleted: f.Local.IsDownloadingActive,
					IsUploadingActive:      f.Remote.IsUploadingActive,
					IsUploadingCompleted:   f.Remote.IsUploadingCompleted,
					CreatedAt:              time.Now().Unix(),
					PublishedAt:            v.Date,
				}); err != nil {
					return err
				}
			case tdlib.MessageVideoType:
				var vi, videoCastOK = v.Content.(*tdlib.MessageVideo)
				if !videoCastOK {
					log.Println("ERROR: fail to type cast to MessageVideo")
					continue
				}
				var f, downloadErr = client.DownloadFile(vi.Video.Video.ID, 32)
				if downloadErr != nil {
					return downloadErr
				}
				if err := postgres.InsertTGVideo(&entity.Video{
					Caption:                vi.Caption.Text,
					ChatID:                 chatID,
					MessageID:              v.ID,
					MediaAlbumID:           int64(v.MediaAlbumID),
					FileID:                 vi.Video.Video.ID,
					SenderUserID:           v.SenderUserID,
					IsDownloadingActive:    f.Local.IsDownloadingActive,
					IsDownloadingCompleted: f.Local.IsDownloadingCompleted,
					IsUploadingActive:      f.Remote.IsUploadingActive,
					IsUploadingCompleted:   f.Remote.IsUploadingCompleted,
					CreatedAt:              time.Now().Unix(),
					PublishedAt:            v.Date,
				}); err != nil {
					return err
				}
			}
			lastMessageID = v.ID
		}
		time.Sleep(time.Second)
	}
	return nil
}

func getLargestResolution(sizes []tdlib.PhotoSize) tdlib.PhotoSize {
	var largestResolution int64
	var retIdx int
	for i := range sizes {
		if int64(sizes[i].Width*sizes[i].Height) > largestResolution {
			largestResolution = int64(sizes[i].Width * sizes[i].Height)
			retIdx = i
		}
	}
	return sizes[retIdx]
}

func addUpdateFileMessageFitler(client *tdlib.Client) {
	var eventFilter = func(msg *tdlib.TdMessage) bool {
		var updateFileMsg = (*msg).(*tdlib.UpdateFile)
		// For example, we want incomming messages from user with below id:
		if updateFileMsg.File.Local.IsDownloadingCompleted == true {
			return true
		}
		return false
	}

	var receiver = client.AddEventReceiver(&tdlib.UpdateFile{}, eventFilter, 10)
	for newMsg := range receiver.Chan {
		var updateFileMsg = (newMsg).(*tdlib.UpdateFile)

		// see if file_id belongs to photos.
		var p, getPErr = postgres.GetTGPhotoByFileID(updateFileMsg.File.ID)
		if getPErr != nil {
			if getPErr != sql.ErrNoRows {
				log.Println("Error fail to get photo from db, err: ", getPErr)
				continue
			}
		} else {
			if updatePhotoErr := updateTGPhotoAndBackup(updateFileMsg, p); updatePhotoErr != nil {
				log.Printf("fail to update db photo record and backup, error: %+v\n", updatePhotoErr)
				continue
			}
			log.Printf("moved photo with id %d to backup folder\n", p.ID)
			continue
		}

		// see if file_id belongs to videos.
		var v, getVErr = postgres.GetTGVideoByFileID(updateFileMsg.File.ID)
		if getVErr != nil {
			if getVErr != sql.ErrNoRows {
				log.Println("Error fail to get video from db, err: ", getVErr)
				continue
			}
		} else {
			if updatePhotoErr := updateTGVideoAndBackup(updateFileMsg, v); updatePhotoErr != nil {
				log.Printf("fail to update db video record and backup, error: %+v\n", updatePhotoErr)
				continue
			}
			log.Printf("moved video with id %d to backup folder\n", v.ID)
		}
	}
}

func updateTGPhotoAndBackup(updateFileMsg *tdlib.UpdateFile, p *entity.Photo) error {
	if updateErr := postgres.UpdateTGPhoto(updateFileMsg.File); updateErr != nil {
		return errors.Wrap(updateErr, fmt.Sprintf("fail to update db for downloaded photo, file ID: %d", updateFileMsg.File.ID))
	}

	// rename and move file to a more renaming location(S3?)
	var publishTime = time.Unix(int64(p.PublishedAt), 0).UTC()
	var dest string
	if p.MediaAlbumID == 0 {
		dest = filepath.Join(config.C.TG.Backup, publishTime.Format("2006-01-02"), "photos", "photos_id_"+strconv.FormatInt(p.ID, 10)+filepath.Ext(updateFileMsg.File.Local.Path))
	} else {
		dest = filepath.Join(config.C.TG.Backup, publishTime.Format("2006-01-02"), "albums", "album_id_"+strconv.FormatInt(p.MediaAlbumID, 10), "photos_id_"+strconv.FormatInt(p.ID, 10)+filepath.Ext(updateFileMsg.File.Local.Path))
	}
	if mkdirErr := os.MkdirAll(filepath.Dir(dest), os.ModePerm); mkdirErr != nil {
		return errors.Wrap(mkdirErr, fmt.Sprintf("fail to make dir for backup, file ID: %d", updateFileMsg.File.ID))
	}
	if backupErr := os.Rename(updateFileMsg.File.Local.Path, dest); backupErr != nil {
		return errors.Wrap(backupErr, fmt.Sprintf("fail to move photo to backup, path: %s", updateFileMsg.File.Local.Path))
	}
	return nil
}

func updateTGVideoAndBackup(updateFileMsg *tdlib.UpdateFile, v *entity.Video) error {
	if updateErr := postgres.UpdateTGVideo(updateFileMsg.File); updateErr != nil {
		return errors.Wrap(updateErr, fmt.Sprintf("fail to update db for downloaded video, file ID: %d", updateFileMsg.File.ID))
	}

	// rename and move file to a more renaming location(S3?)
	var publishTime = time.Unix(int64(v.PublishedAt), 0).UTC()
	var dest string
	if v.MediaAlbumID == 0 {
		dest = filepath.Join(config.C.TG.Backup, publishTime.Format("2006-01-02"), "videos", "videos_id_"+strconv.FormatInt(v.ID, 10)+filepath.Ext(updateFileMsg.File.Local.Path))
	} else {
		dest = filepath.Join(config.C.TG.Backup, publishTime.Format("2006-01-02"), "albums", "album_id_"+strconv.FormatInt(v.MediaAlbumID, 10), "videos_id_"+strconv.FormatInt(v.ID, 10)+filepath.Ext(updateFileMsg.File.Local.Path))
	}
	if mkdirErr := os.MkdirAll(filepath.Dir(dest), os.ModePerm); mkdirErr != nil {
		return errors.Wrap(mkdirErr, fmt.Sprintf("fail to make dir for backup, file ID: %d", updateFileMsg.File.ID))
	}
	if backupErr := os.Rename(updateFileMsg.File.Local.Path, dest); backupErr != nil {
		return errors.Wrap(backupErr, fmt.Sprintf("fail to move photo to backup, path: %s", updateFileMsg.File.Local.Path))
	}
	return nil
}
