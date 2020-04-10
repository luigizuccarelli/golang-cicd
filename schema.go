package main

import (
	"strings"
	"time"
)

type CustomTime struct {
	time.Time
}

func (ct *CustomTime) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), "\"")
	if s == "null" {
		ct.Time = time.Time{}
		return
	}
	ct.Time, err = time.Parse("2006-01-02 15:04", s)
	return
}

// ShcemaInterface - acts as an interface wrapper for our profile schema
// All the go microservices will using this schema
type Pipeline struct {
	Project    string        `json:"project"`
	Scm        string        `json:"scm"`
	Workdir    string        `json:"workdir"`
	Stages     []StageDetail `json:"stages"`
	LastUpdate int64         `json:"lastupdate,omitempty"`
	MetaInfo   string        `json:"metainfo,omitempty"`
}

type StageDetail struct {
	Id       int      `json:"id"`
	Name     string   `json:"name"`
	Exec     string   `json:"exec"`
	Wait     int      `json:"wait"`
	Skip     bool     `json:"skip"`
	Commands []string `json:"commands"`
}

type ProjectList struct {
	Projects []ProjectDetail `json:"projects"`
}

type ProjectDetail struct {
	Name     string `json:"name"`
	Scm      string `json:"scm"`
	Workdir  string `json:"workdir"`
	Path     string `json:"path"`
	MetaInfo string `json:"metainfo,omitempty"`
}

// Response schema
type Response struct {
	StatusCode string `json:"statuscode"`
	Status     string `json:"status"`
	Message    string `json:"message"`
}
