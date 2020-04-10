package connectors

import (
	"net/http"
	"time"

	"gitea-cicd.apps.aws2-dev.ocp.14west.io/cicd/golang-mongodbinterface/pkg/schema"
)

type Clients interface {
	Error(string, ...interface{})
	Info(string, ...interface{})
	Debug(string, ...interface{})
	Trace(string, ...interface{})
	DBInsert(body []byte) error
	DBUpdate(body []byte) (schema.SchemaInterface, error)
	DBGet(string) (schema.SchemaInterface, error)
	DBDelete(string) error
	DBList(*schema.ListRange) ([]schema.SchemaInterface, error)
	Do(req *http.Request) (*http.Response, error)
	Get(string) (string, error)
	Set(string, string, time.Duration) (string, error)
	Close() error
}
