package schema

import (
	"github.com/globalsign/mgo/bson"
)

// ShcemaInterface - acts as an interface wrapper
type SchemaInterface struct {
	ID         bson.ObjectId `json:"_id" bson:"_id,omitempty"`
	LastUpdate int64         `json:"lastupdate,omitempty"`
	MetaInfo   string        `json:"metainfo,omitempty"`
	Custom     CustomDetail  `json:"custom,omitempty"`
}

// CustomDetail schema
type CustomDetail struct {
	Name    string `json:"name"`
	Surname string `json:"surname"`
	Email   string `json:"email"`
	Title   string `json:"title"`
	Mobile  string `json:"mobile"`
	Address string `json:"address"`
}

// ListRange - used for pagination
type ListRange struct {
	From   int    `json:"From"`
	To     int    `json:"to"`
	Search string `json:"search,omitempty"`
}

// Response schema
type Response struct {
	Code       int               `json:"code,omitempty"`
	StatusCode string            `json:"statuscode"`
	Status     string            `json:"status"`
	Message    string            `json:"message"`
	Payload    []SchemaInterface `json:"payload"`
}
