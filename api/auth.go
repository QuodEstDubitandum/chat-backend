package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"realtime-chat/database"
	"realtime-chat/models"

	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)


type Body struct{
	Username string `json:"username"` 
} 
type JWTBody struct{
	JWT string `json:"jwt"`
}

type Response struct{
	Username string `json:"username"`
	JWT string `json:"jwt"`
	ExpirationDate time.Time `json:"expirationDate"`
}

var jwtKey = []byte("jwt_baby")

func HandleAuth(w http.ResponseWriter, r *http.Request){
	// first check for the auth token 
	auth_token:= r.Header.Get("Authorization")
	if auth_token != os.Getenv("AUTH_TOKEN"){
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Invalid Auth Token"))
		return
	}

	// decode body
	var body Body
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Not a viable payload"))
		return
	}
	defer r.Body.Close()

	
	// set expiration date 
	expirationTime := time.Now().AddDate(0,1,0)

	// check if there is already a user with that username in the database and fetch jwt if so
	ctx := context.Background()
	dbClient := database.GetDbClient()
	db_jwt, err := database.CheckIfUserExists(body.Username, ctx, dbClient)
	if db_jwt != ""{
		responseData := Response{Username: body.Username, JWT: db_jwt, ExpirationDate: expirationTime}
		jsonData, err := json.Marshal(responseData)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal Server Error"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonData)
		return
	}

	// construct new jwt
	var claims = struct {
		Username          string
		jwt.RegisteredClaims
	}{
		Username: body.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
		return
	}

	// parallelize adding new user to database and setting cookie
	var wg sync.WaitGroup
	wg.Add(1)

	go func (){
		defer wg.Done()
		payload := models.NewUser{
			Name: body.Username,
			JWT: tokenString,
			CreatedAt: time.Now(),
		}
		err = database.AddNewUser(payload , ctx, dbClient)
		if err != nil{
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("User was not persisted in database"))
		}
	}()

	responseData := Response{Username: body.Username, JWT: tokenString, ExpirationDate: expirationTime}
	jsonData, err := json.Marshal(responseData)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)

	wg.Wait()
	return
}

func HandleJWT(w http.ResponseWriter, r *http.Request){
	apiKey := r.Header.Get("X-API-Key")
	if apiKey != os.Getenv("API_KEY") {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`Invalid API Key`))
		return
	}

	var body JWTBody
	// decode body
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Not viable payload"))
		return
	}
	defer r.Body.Close()

	// check if the jwt exists in the database and if so return the username as response
	ctx := context.Background()
	dbClient := database.GetDbClient()
	username, err := database.CheckIfJWTExists(body.JWT, ctx, dbClient)
	if err == nil{
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(username))
		return
	}
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("JWT is not valid"))
	return
}
