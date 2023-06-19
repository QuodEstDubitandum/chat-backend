package models

import "time"

type NewUser struct {
	Name      string
	JWT       string
	CreatedAt time.Time
}