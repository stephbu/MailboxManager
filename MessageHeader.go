package main

import (
	"fmt"
	"time"
)

type MessageHeader struct {
	Id        string
	Subject   string
	From      string
	Time      time.Time
	MessageId string
}

// converting the struct to String format.
func (mh MessageHeader) String() string {
	return fmt.Sprintf(mh.Id, mh.Time)
}
