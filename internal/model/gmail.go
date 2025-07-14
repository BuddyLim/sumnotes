package model

import "google.golang.org/api/gmail/v1"

type GmailListResponse struct {
	Messages []gmail.Message `json:"messages"`
}
