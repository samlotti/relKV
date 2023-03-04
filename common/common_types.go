package common

import (
	"time"
)

type KV struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
	Error string `json:"error,omitempty"`
}

type BucketData struct {
	Name     string `json:"name"`
	Error    string `json:"error,omitempty"`
	LsmSize  int64  `json:"lsmSize"`
	VlogSize int64  `json:"VlogSize"`
}

type BucketName string

type ScpStatus int

const (
	ScpPending  ScpStatus = 0
	ScpRunning  ScpStatus = 1
	ScpComplete ScpStatus = 2
	ScpError    ScpStatus = 3
)

type ScpJob struct {
	Fname      string
	BucketName BucketName
	Status     ScpStatus
	Message    string
	LastStart  time.Time
	LastEnd    time.Time
	NextSend   time.Time
}
