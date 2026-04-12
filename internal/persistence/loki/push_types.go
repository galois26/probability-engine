package loki

import "time"

type PushRequest struct {
	Streams []PushStream
}

type PushStream struct {
	Stream map[string]string
	Values []PushValue
}

type PushValue struct {
	Timestamp time.Time
	Line      string
}
