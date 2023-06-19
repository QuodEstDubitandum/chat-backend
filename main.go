package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"net/url"
	"realtime-chat/api"
	"realtime-chat/database"
	"realtime-chat/models"

	"sync"

	"cloud.google.com/go/firestore"
	"golang.org/x/net/websocket"
)

type Server struct {
	connections map[*websocket.Conn]string
	mutex sync.RWMutex
}

func (s *Server) ListenForMessages(ws *websocket.Conn, messageChan chan<- string, ctx context.Context, dbClient *firestore.Client) {
	reader := make([]byte, 512)
	for {
		n, err := ws.Read(reader)
		if err != nil {
			if err == io.EOF {
				close(messageChan)
				ws.Close()
				break
			}
			fmt.Println("Read Error: ", err)
			continue
		}
		msg := reader[:n]
		messageChan <- string(msg)


		messageForDB := &models.LatestMessage{
			Message: string(msg),
			SendBy: s.connections[ws],
			CreatedAt: time.Now(),
		}
		
		err = database.AddNewMessage(*messageForDB, ctx, dbClient)
		if err != nil{
			close(messageChan)
			ws.Close()
			break
		}
	}
}

func (s *Server) BroadcastMessages(ws *websocket.Conn, messageChan <-chan string) {
	for message := range messageChan {
		s.mutex.RLock()
		fmt.Println("New message: ", message)
		for conn := range s.connections {
			_, err := conn.Write([]byte(fmt.Sprintf(`%s:%s`, s.connections[ws], message)))
			if err != nil {
				fmt.Println("Write Error: ", err)
				continue
			}
		}
		s.mutex.RUnlock()
	}
}

func (s* Server) handleNewConnection(ws *websocket.Conn){
	headers := strings.Split(ws.Request().Header.Get("Sec-WebSocket-Protocol"), ".")
	apiKey := headers[0]
	user := headers[1]

	decodedUsername, err := url.PathUnescape(user)
	if err != nil {
		errMsg := []byte("Username decoding error")
		ws.Write(errMsg)
		ws.Close()
		return
	}

	if apiKey != os.Getenv("API_KEY") {
		errMsg := []byte("Invalid API key")
		ws.Write(errMsg)
		ws.Close()
		return
	}

	if decodedUsername == ""{
		errMsg := []byte("Invalid Username")
		ws.Write(errMsg)
		ws.Close()
		return
	}

	messageChan := make(chan string) 

	s.mutex.Lock()
	s.connections[ws] = decodedUsername
	s.mutex.Unlock()
	
	go s.BroadcastMessages(ws, messageChan)
	messageChan<-fmt.Sprintf("%s just connected", s.connections[ws])

	ctx := context.Background()
	dbClient := database.GetDbClient()

	s.ListenForMessages(ws, messageChan, ctx, dbClient)

	if err := ws.Close(); err != nil{
		s.mutex.RLock()
		for conn := range s.connections {
			if conn != ws {
				_, err := conn.Write([]byte(fmt.Sprintf("%s just disconnected", s.connections[ws])))
				if err != nil {
					fmt.Println("Write Error: ", err)
					continue
				}
			}
		}
		s.mutex.RUnlock()
		delete(s.connections, ws)
		return
	}
}

func WithMiddleware(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", os.Getenv("FRONTEND_URL"))
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		handler(w, r)
	}
}

func main(){
	// err := godotenv.Load()
	// if err != nil {
	// 	log.Fatal("Error loading .env file")
	// }

	ctx := context.Background()
	firestore := database.SetupDatabase(ctx)
	defer firestore.Close()

	server := &Server{connections: make(map[*websocket.Conn]string), mutex: sync.RWMutex{}}

	http.Handle("/checkJWT", WithMiddleware(api.HandleJWT))
	http.Handle("/auth", WithMiddleware(api.HandleAuth))
	http.Handle("/getMessages", WithMiddleware(api.GetLastMessages))
	http.Handle("/chat", websocket.Handler(server.handleNewConnection))

	fmt.Println("Server listening on port 8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil{
		log.Fatal("Server connection error: ", err)
	}
	defer fmt.Println("Server disconnected")
}
