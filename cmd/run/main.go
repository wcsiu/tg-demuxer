package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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
		retrieveAllPreviousPhotosFromChat(client, chat.ID)
	}

	for {
		time.Sleep(1 * time.Second)
	}
}

// see https://stackoverflow.com/questions/37782348/how-to-use-getchats-in-tdlib
func updateChatList(client *tdlib.Client) error {
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
	var msgs, getMsgsErr = client.GetChatHistory(chatID, 0, 0, 1, false)
	if getMsgsErr != nil {
		log.Println("ERROR: fail to get messages from chat: ", chatID, ", error: ", getMsgsErr)
	}
	for _, v := range msgs.Messages {
		if v.Content.GetMessageContentEnum() == tdlib.MessagePhotoType {
			var p, photoCastOK = v.Content.(*tdlib.MessagePhoto)
			if !photoCastOK {
				log.Println("ERROR: fail to type cast to MessagePhoto")
			}
			var largest = getLargestResolution(p.Photo.Sizes)
			var f, downloadErr = client.DownloadFile(largest.Photo.ID, 32)
			if downloadErr != nil {
				return downloadErr
			}
			if err := postgres.InsertTGPhoto(&entity.Photos{
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
				CreateAt:               time.Now().Unix(),
				PublishedAt:            v.Date,
			}); err != nil {
				return err
			}

		}
	}
	for msgs.TotalCount > 0 {
		msgs, getMsgsErr = client.GetChatHistory(chatID, msgs.Messages[0].ID, 0, 1, false)
		if getMsgsErr != nil {
			log.Println("ERROR: fail to get messages from chat: ", chatID, ", error: ", getMsgsErr)
		}
		for _, v := range msgs.Messages {
			if v.Content.GetMessageContentEnum() == tdlib.MessagePhotoType {
				var p, photoCastOK = v.Content.(*tdlib.MessagePhoto)
				if !photoCastOK {
					log.Println("ERROR: fail to type cast to MessagePhoto")
				}
				var largest = getLargestResolution(p.Photo.Sizes)
				var f, downloadErr = client.DownloadFile(largest.Photo.ID, 32)
				if downloadErr != nil {
					return downloadErr
				}
				if err := postgres.InsertTGPhoto(&entity.Photos{
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
					CreateAt:               time.Now().Unix(),
					PublishedAt:            v.Date,
				}); err != nil {
					return err
				}
			}
		}
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
		fmt.Println(newMsg)
		var updateFileMsg = (newMsg).(*tdlib.UpdateFile)

		// TODO: compute image hash
		// TODO: rename and move file to a more renaming location(S3?)
		if updateErr := postgres.UpdateTGPhoto(updateFileMsg.File); updateErr != nil {
			log.Println("Error fail to update db for downloaded photo, err: ", updateErr)
		}
	}
}
