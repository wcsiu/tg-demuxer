package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/wcsiu/go-tdlib"
	"github.com/wcsiu/tg-demuxer/internal/aws/s3"
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

	// connect s3
	s3.Load()

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
			var currentState, _ = client.Authorize()
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
	var currentState, _ = client.Authorize()
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

	go func() {
		for _, chat := range allChats {
			var nowUnix = time.Now().Unix()
			if insertErr := postgres.InsertChat(&entity.Chat{Title: chat.Title, ChatID: chat.ID, CreatedAt: nowUnix, ModifiedAt: nowUnix}); insertErr != nil {
				log.Printf("fail to insert chat info to db, error: %+v", insertErr)
			}
			if err := retrieveAllPreviousPhotosFromChat(client, chat.ID); err != nil {
				log.Println("Error fail to retrieve all previous photos, error: ", err)
			}
			var lastMessageID, lastMsgIDErr = postgres.GetMaxMsgIDInChat(chat.ID)
			if lastMsgIDErr != nil {
				log.Printf("fail to get last message id from db, error: %+v", lastMsgIDErr)
			}
			go TgChatUpdate(client, chat.ID, lastMessageID, time.Minute)
		}
	}()

	http.HandleFunc("/", helloServer)
	http.ListenAndServe(":3000", nil)
}

func helloServer(w http.ResponseWriter, req *http.Request) {
	io.WriteString(w, "hello socket activated world!\n")
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
			time.Sleep(5 * time.Second)
			msgs.TotalCount = 1
			continue
		}
		for _, v := range msgs.Messages {
			switch v.Content.GetMessageContentEnum() {
			case tdlib.MessagePhotoType:
				var nowUnix = time.Now().Unix()
				if insertMsgErr := postgres.InsertMessage(&entity.Message{
					MessageType:  entity.MessageTypePhoto,
					MessageID:    v.ID,
					ChatID:       v.ChatID,
					MediaAlbumID: int64(v.MediaAlbumID),
					Uploaded:     false,
					CreateAt:     nowUnix,
					ModifiedAt:   nowUnix,
					PublishedAt:  v.Date,
				}); insertMsgErr != nil {
					log.Printf("ERROR: fail to insert message into db, message id: %d, chat id: %d, error: %+v", v.ID, v.ChatID, insertMsgErr)
					break
				}
				var p, photoCastOK = v.Content.(*tdlib.MessagePhoto)
				if !photoCastOK {
					log.Println("ERROR: fail to type cast to MessagePhoto")
					break
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
					CreatedAt:              nowUnix,
					ModifiedAt:             nowUnix,
					PublishedAt:            v.Date,
				}); err != nil {
					return err
				}
			case tdlib.MessageVideoType:
				var nowUnix = time.Now().Unix()
				if insertMsgErr := postgres.InsertMessage(&entity.Message{
					MessageType:  entity.MessageTypeVideo,
					MessageID:    v.ID,
					ChatID:       v.ChatID,
					MediaAlbumID: int64(v.MediaAlbumID),
					Uploaded:     false,
					CreateAt:     nowUnix,
					ModifiedAt:   nowUnix,
					PublishedAt:  v.Date,
				}); insertMsgErr != nil {
					log.Printf("ERROR: fail to insert message into db, message id: %d, chat id: %d, error: %+v", v.ID, v.ChatID, insertMsgErr)
					continue
				}
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
					CreatedAt:              nowUnix,
					ModifiedAt:             nowUnix,
					PublishedAt:            v.Date,
				}); err != nil {
					return err
				}
			}
			lastMessageID = v.ID
		}
		time.Sleep(5 * time.Second)
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
				log.Printf("fail to update db photo record and backup, error: %+v", updatePhotoErr)
				continue
			}
			if updateMessageErr := updateMessage(p.MessageID, p.ChatID); updateMessageErr != nil {
				log.Printf("fail to update message, error: %+v", updateMessageErr)
				continue
			}
			log.Printf("moved photo with id %d to backup folder", p.ID)
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
			if updateVideoErr := updateTGVideoAndBackup(updateFileMsg, v); updateVideoErr != nil {
				log.Printf("fail to update db video record and backup, error: %+v", updateVideoErr)
				continue
			}
			if updateMessageErr := updateMessage(v.MessageID, v.ChatID); updateMessageErr != nil {
				log.Printf("fail to update message, error: %+v", updateMessageErr)
				continue
			}
			log.Printf("moved video with id %d to backup folder\n", v.ID)
		}

	}
}

func updateMessage(messageID, chatID int64) error {
	var m entity.Message
	if getMErr := postgres.GetMessageByMessageID(&m, messageID, chatID); getMErr != nil {
		log.Printf("fail to get message from db, error: %+v", getMErr)
		return errors.Wrap(getMErr, "fail to get message from db")
	}
	if m.Uploaded {
		log.Printf("message is already updated, message id: %d, chat id: %d", messageID, chatID)
		return errors.New("message is already updated")
	}
	if updateMErr := postgres.UpdateMessageUploadStatus(&m, true); updateMErr != nil {
		log.Printf("fail to update message for db, error; %+v", updateMErr)
		return errors.Wrap(updateMErr, "fail to update message for db")
	}
	return nil
}

func updateTGPhotoAndBackup(updateFileMsg *tdlib.UpdateFile, p *entity.Photo) error {
	var publishTime = time.Unix(int64(p.PublishedAt), 0).UTC()
	var path string
	if p.MediaAlbumID == 0 {
		path = filepath.Join(publishTime.Format("2006-01-02"), "photos", "photos_id_"+strconv.FormatInt(p.ID, 10)+filepath.Ext(updateFileMsg.File.Local.Path))
	} else {
		path = filepath.Join(publishTime.Format("2006-01-02"), "albums", "album_id_"+strconv.FormatInt(p.MediaAlbumID, 10), "photos_id_"+strconv.FormatInt(p.ID, 10)+filepath.Ext(updateFileMsg.File.Local.Path))
	}

	if updateErr := postgres.UpdateTGPhoto(updateFileMsg.File, path); updateErr != nil {
		return errors.Wrap(updateErr, fmt.Sprintf("fail to update db for downloaded photo, file ID: %d", updateFileMsg.File.ID))
	}

	// upload to s3
	var f, openErr = os.Open(updateFileMsg.File.Local.Path)
	if openErr != nil {
		return errors.Wrap(openErr, fmt.Sprintf("fail to open downloaded file, path: %s", updateFileMsg.File.Local.Path))
	}
	var _, uploadErr = s3.Upload(path, f)
	if uploadErr != nil {
		return errors.Wrap(uploadErr, fmt.Sprintf("fail to upload file, path: %s", path))
	}

	// delete uploaded file from local
	if removeErr := os.Remove(updateFileMsg.File.Local.Path); removeErr != nil {
		return errors.Wrap(removeErr, fmt.Sprintf("fail to remove uploaded file, path: %s", updateFileMsg.File.Local.Path))
	}

	return nil
}

func updateTGVideoAndBackup(updateFileMsg *tdlib.UpdateFile, v *entity.Video) error {
	// var dest string
	var path string
	var publishTime = time.Unix(int64(v.PublishedAt), 0).UTC()
	if v.MediaAlbumID == 0 {
		path = filepath.Join(publishTime.Format("2006-01-02"), "videos", "videos_id_"+strconv.FormatInt(v.ID, 10)+filepath.Ext(updateFileMsg.File.Local.Path))
		// dest = filepath.Join(config.C.TG.Backup, path)
	} else {
		path = filepath.Join(publishTime.Format("2006-01-02"), "albums", "album_id_"+strconv.FormatInt(v.MediaAlbumID, 10), "videos_id_"+strconv.FormatInt(v.ID, 10)+filepath.Ext(updateFileMsg.File.Local.Path))
		// dest = filepath.Join(config.C.TG.Backup, path)
	}
	if updateErr := postgres.UpdateTGVideo(updateFileMsg.File, path); updateErr != nil {
		return errors.Wrap(updateErr, fmt.Sprintf("fail to update db for downloaded video, file ID: %d", updateFileMsg.File.ID))
	}

	// upload to s3
	var f, openErr = os.Open(updateFileMsg.File.Local.Path)
	if openErr != nil {
		return errors.Wrap(openErr, fmt.Sprintf("fail to open downloaded file, path: %s", updateFileMsg.File.Local.Path))
	}
	var _, uploadErr = s3.Upload(path, f)
	if uploadErr != nil {
		return errors.Wrap(uploadErr, fmt.Sprintf("fail to upload file, path: %s", path))
	}

	// delete uploaded file from local
	if removeErr := os.Remove(updateFileMsg.File.Local.Path); removeErr != nil {
		return errors.Wrap(removeErr, fmt.Sprintf("fail to remove uploaded file, path: %s", updateFileMsg.File.Local.Path))
	}
	return nil
}

// TgChatUpdate telegram chat update
func TgChatUpdate(client *tdlib.Client, chatID, initialLastMessageID int64, interval time.Duration) {
	var ticker = time.NewTicker(interval)
	defer ticker.Stop()
	var lastMessageID = initialLastMessageID
	for {
		select {
		case t := <-ticker.C:
			log.Printf("UPDATE: time: %+v", t)
			var msgs = &tdlib.Messages{
				TotalCount: 2,
			}
			for msgs.TotalCount > 1 {
				var next bool
				var getMsgsErr error
				msgs, getMsgsErr = client.GetChatHistory(chatID, lastMessageID, -10, 10, false)
				if getMsgsErr != nil {
					log.Println("ERROR: fail to get messages from chat: ", chatID, ", error: ", getMsgsErr)
				}
				var tempInitialLastMessageID = lastMessageID
				for _, v := range msgs.Messages {
					if v.ID == tempInitialLastMessageID {
						continue
					}
					switch v.Content.GetMessageContentEnum() {
					case tdlib.MessagePhotoType:
						var nowUnix = time.Now().Unix()
						if insertMsgErr := postgres.InsertMessage(&entity.Message{
							MessageType:  entity.MessageTypePhoto,
							MessageID:    v.ID,
							ChatID:       v.ChatID,
							MediaAlbumID: int64(v.MediaAlbumID),
							Uploaded:     false,
							CreateAt:     nowUnix,
							ModifiedAt:   nowUnix,
							PublishedAt:  v.Date,
						}); insertMsgErr != nil {
							log.Printf("ERROR: fail to insert message to db, message id: %d, chat id: %d, error: %+v", v.ID, v.ChatID, insertMsgErr)
							next = true
							break
						}
						var p, photoCastOK = v.Content.(*tdlib.MessagePhoto)
						if !photoCastOK {
							log.Println("ERROR: fail to type cast to MessagePhoto")
							next = true
							break
						}
						var largest = getLargestResolution(p.Photo.Sizes)
						var f, downloadErr = client.DownloadFile(largest.Photo.ID, 32)
						if downloadErr != nil {
							log.Println("ERROR: fail to download photo, photo id: ", largest.Photo.ID, ", error: ", downloadErr)
							next = true
							break
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
							CreatedAt:              nowUnix,
							PublishedAt:            v.Date,
						}); err != nil {
							log.Println("ERROR: fail to insert photo into db, message id: ", v.ID, ", error: ", err)
							next = true
							break
						}
					case tdlib.MessageVideoType:
						var nowUnix = time.Now().Unix()
						if insertMsgErr := postgres.InsertMessage(&entity.Message{
							MessageType:  entity.MessageTypeVideo,
							MessageID:    v.ID,
							ChatID:       v.ChatID,
							MediaAlbumID: int64(v.MediaAlbumID),
							Uploaded:     false,
							CreateAt:     nowUnix,
							ModifiedAt:   nowUnix,
							PublishedAt:  v.Date,
						}); insertMsgErr != nil {
							log.Printf("ERROR: fail to insert message to db, message id: %d, chat id: %d, error: %+v", v.ID, v.ChatID, insertMsgErr)
							next = true
							break
						}
						var vi, videoCastOK = v.Content.(*tdlib.MessageVideo)
						if !videoCastOK {
							log.Println("ERROR: fail to type cast to MessageVideo")
							next = true
							break
						}
						var f, downloadErr = client.DownloadFile(vi.Video.Video.ID, 32)
						if downloadErr != nil {
							log.Println("ERROR: fail to download video, video id: ", vi.Video.Video.ID, ", error: ", downloadErr)
							next = true
							break
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
							log.Printf("ERROR: fail to insert video to db, message id: %d, chat id: %d, error: %+v", v.ID, v.ChatID, err)
							next = true
							break
						}
					}
					if next {
						break
					}
					lastMessageID = v.ID
				}
				if next {
					break
				}
				time.Sleep(time.Second)
			}
		}
	}
}
