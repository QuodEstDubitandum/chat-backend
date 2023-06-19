package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"realtime-chat/models"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)



var dbClient *firestore.Client

func GetDbClient() *firestore.Client{
	return dbClient
}

func SetupDatabase(ctx context.Context) *firestore.Client{
	credentials_path := os.Getenv("FIREBASE_CREDENTIALS_PATH")

	opt := option.WithCredentialsFile(credentials_path)
	config := &firebase.Config{ProjectID: "go-chat-78ffc"}
	app, err := firebase.NewApp(ctx, config, opt)
	if err != nil {
		log.Fatalf("Failed to initialize Firebase app: %v", err)
	}

	dbClient, err = app.Firestore(ctx)
	if err != nil {
		log.Fatalf("Failed to access Firestore: %v", err)
	}
	fmt.Println("Database initialized successfully")
	return dbClient
}

func CheckIfUserExists(username string, ctx context.Context, db *firestore.Client) (string, error) {
	iter := db.Collection("users").Where("name", "==", username).Limit(1).Documents(ctx)

	doc, err := iter.Next()
	if err != nil {
		if err == iterator.Done {
			return "", nil
		}
		return "", err
	}
	return doc.Data()["jwt"].(string), nil
}

func CheckIfJWTExists(jwt string, ctx context.Context, db *firestore.Client) (string, error) {
	iter := db.Collection("users").Where("jwt", "==", jwt).Limit(1).Documents(ctx)

	data, err := iter.Next()
	if err != nil {
		return "", err
	}
	return data.Data()["name"].(string), nil
}

func AddNewUser(payload models.NewUser, ctx context.Context, db *firestore.Client) error {
	_, _, err := db.Collection("users").Add(ctx, map[string]interface{}{
			"name": payload.Name,
			"jwt": payload.JWT,
			"createdAt": payload.CreatedAt,
        })
	return err
}

func GetLatestChats(ctx context.Context, db *firestore.Client) []models.LatestMessage {
	var i = 0
	chatMessages := make([]models.LatestMessage,0, 20)

	iter := db.Collection("chat_messages").OrderBy("createdAt", firestore.Desc).Limit(20).Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {	
			break
		}
		if err != nil {
			break
		}
		chatMessages = append(chatMessages, models.LatestMessage{
			Message: doc.Data()["message"].(string),
			SendBy: doc.Data()["sendBy"].(string),
			CreatedAt: doc.Data()["createdAt"].(time.Time),
		})
		i++
	}
	return chatMessages
}

func AddNewMessage(payload models.LatestMessage, ctx context.Context, db *firestore.Client) error{
	_, _, err := db.Collection("chat_messages").Add(ctx, map[string]interface{}{
			"message": payload.Message,
			"sendBy": payload.SendBy,
			"createdAt": payload.CreatedAt,
        })
	return err
}