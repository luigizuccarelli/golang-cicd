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
	Id         string        `json:"id"`
	Project    string        `json:"project"`
	Scm        string        `json:"scm"`
	Workdir    string        `json:"workdir"`
	Force      bool          `json:"force"`
	Stages     []StageDetail `json:"stages"`
	LastUpdate int64         `json:"lastupdate,omitempty"`
	MetaInfo   string        `json:"metainfo,omitempty"`
}

type StageDetail struct {
	Id       int           `json:"id"`
	Name     string        `json:"name"`
	Exec     string        `json:"exec"`
	Wait     int           `json:"wait"`
	Service  string        `json:"service"`
	Replicas int           `json:"replicas"`
	Skip     bool          `json:"skip"`
	Envars   []EnvarDetail `json:"envars"`
	Commands []string      `json:"commands"`
	Status   string        `json:"status"`
	Log      string        `json:"log"`
}

type EnvarDetail struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Response schema
type Response struct {
	Name       string      `json:"name"`
	StatusCode string      `json:"statuscode"`
	Status     string      `json:"status"`
	Message    string      `json:"message"`
	Stage      StageDetail `json:"stage,omitempty"`
	MetaInfo   string      `json:"metainfo,omitempty"`
	Payload    []Pipeline  `json:"payload"`
}

type Repository struct {
	Name     string `json:"name"`
	MetaInfo string `json:"metainfo"`
	WorkDir  string `json:"workdir"`
	Path     string `json:"path"`
	Scm      string `json:"scm"`
	RawUrl   string `json:"cicd-raw-url"`
	Skip     bool   `json:"skip"`
	Force    bool   `json:"force"`
}

type ProjectDetail struct {
	Name         string       `json:"name"`
	Repositories []Repository `json:"repositories"`
}
