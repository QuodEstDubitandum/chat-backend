package models

import "time"

type LatestMessage struct {
	Message   string `json:"message"`
	SendBy string `json:"sendBy"`
	CreatedAt time.Time `json:"createdAt"`
}