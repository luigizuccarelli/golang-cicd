package connectors

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"gitea-cicd.apps.aws2-dev.ocp.14west.io/cicd/golang-mongodbinterface/pkg/schema"
	"github.com/globalsign/mgo/bson"
	"github.com/microlib/simple"
)

var (
	m map[string]string
)

type errReader int

func (errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("Injected error")
}

// RoundTripFunc .
type RoundTripFunc func(req *http.Request) *http.Response

// RoundTrip .
func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

//NewHttpTestClient returns *http.Client with Transport replaced to avoid making real calls
func NewHttpTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: RoundTripFunc(fn),
	}
}

// fake mongo db section

// DataLayer is an interface to access to the database struct.

func NewClientTestConnections(file string, status int, logger *simple.Logger) Clients {

	// we first load the json payload to simulate a call to middleware
	// for now just ignore failures.
	data, err := ioutil.ReadFile(file)
	if err != nil {
		logger.Error(fmt.Sprintf("file data %v\n", err))
		panic(err)
	}
	httpClient := NewHttpTestClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: status,
			// Send response to be tested
			Body: ioutil.NopCloser(bytes.NewBufferString(string(data))),
			// Must be set to non-nil value or it panics
			Header: make(http.Header),
		}
	})

	m = make(map[string]string)
	redisClient := &FakeRedis{}
	mgo := NewFakeSession()

	return &Connections{Http: httpClient, Redis: redisClient, DB: mgo, Name: "FakeConnections", l: logger}
}

// implementation

// fake redis Get
func (r *Connections) Get(key string) (string, error) {
	if key == "error" {
		return "", errors.New("Get method failed")
	}
	return m[key], nil
}

// fake redis Set
func (r *Connections) Set(key string, value string, expr time.Duration) (string, error) {
	if key == "error" {
		return "", errors.New("Set method failed")
	}
	m[key] = value
	return value, nil
}

// fake redis Close
func (r *Connections) Del(key string) error {
	if key == "error" {
		return errors.New("Del method failed")
	}
	delete(m, key)
	return nil
}

func (r *FakeRedis) Close() error {
	return nil
}

func (r *Connections) Close() error {
	return nil
}

func assertEqual(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Fatalf("%s != %s", a, b)
	}
}

func TestImplementation(t *testing.T) {

	logger := &simple.Logger{Level: "info"}

	t.Run("Insert : should pass", func(t *testing.T) {
		conn := NewClientTestConnections("../../tests/payload-example.json", 200, logger)
		custom := schema.CustomDetail{Name: "test", Surname: "test", Email: "test@test"}
		d := schema.SchemaInterface{LastUpdate: time.Now().UnixNano(), MetaInfo: "nada", Custom: custom}
		b, _ := json.Marshal(d)
		err := conn.DBInsert(b)
		if err != nil {
			t.Errorf(fmt.Sprintf("Test Insert %s returned with error - got (%s) wanted (%s)", "DBInsert", "nil", "error"))
		}
	})

	t.Run("Insert : should fail", func(t *testing.T) {
		conn := NewClientTestConnections("../../tests/payload-example.json", 500, logger)
		b, _ := json.Marshal([]byte("{ "))
		err := conn.DBInsert(b)
		if err == nil {
			t.Errorf(fmt.Sprintf("Test Insert %s returned with no error - got (%s) wanted (%s)", "DBInsert", "nil", "error"))
		}
	})

	t.Run("DBInsert : should fail (forced error)", func(t *testing.T) {
		conn := NewClientTestConnections("../../tests/payload-example.json", 500, logger)
		custom := schema.CustomDetail{Name: "test", Surname: "test", Email: "test@test"}
		d := schema.SchemaInterface{LastUpdate: time.Now().UnixNano(), MetaInfo: "ERROR", Custom: custom}
		b, _ := json.Marshal(d)
		err := conn.DBInsert(b)
		if err == nil {
			t.Errorf(fmt.Sprintf("Test Insert %s returned with no error - got (%s) wanted (%s)", "DBInsert", "nil", "error"))
		}
	})

	t.Run("DBUpdate : should pass", func(t *testing.T) {
		conn := NewClientTestConnections("../../tests/payload-example.json", 200, logger)
		custom := schema.CustomDetail{Name: "test", Surname: "test", Email: "test@test"}
		d := schema.SchemaInterface{ID: bson.ObjectIdHex("5cc042307ccc69ada893144c"), MetaInfo: "nada", Custom: custom}
		b, _ := json.Marshal(d)
		s, err := conn.DBUpdate(b)
		if err != nil {
			t.Errorf(fmt.Sprintf("Test Update %s returned with error - got (%s) wanted (%s)", "DBUpdate", "error", "nil"))
		}
		assertEqual(t, s.ID, d.ID)
	})

	t.Run("DBUpdate : should fail", func(t *testing.T) {
		conn := NewClientTestConnections("../../tests/payload-example.json", 500, logger)
		b, _ := json.Marshal([]byte("{ "))
		_, err := conn.DBUpdate(b)
		if err == nil {
			t.Errorf(fmt.Sprintf("Test Update %s returned with no error - got (%s) wanted (%s)", "DBUpdate", "nil", "error"))
		}
	})

	t.Run("DBUpdate : should fail (forced error)", func(t *testing.T) {
		conn := NewClientTestConnections("../../tests/payload-example.json", 200, logger)
		custom := schema.CustomDetail{Name: "test", Surname: "test", Email: "test@test"}
		d := schema.SchemaInterface{ID: bson.ObjectIdHex("5cc042307ccc69ada893144c"), MetaInfo: "ERROR", Custom: custom}
		b, _ := json.Marshal(d)
		s, err := conn.DBUpdate(b)
		if err == nil {
			t.Errorf(fmt.Sprintf("Test Update %s returned with no error - got (%s) wanted (%s)", "DBUpdate", "nil", "error"))
		}
		assertEqual(t, s.ID, d.ID)
	})

	t.Run("DBGet : should pass", func(t *testing.T) {
		conn := NewClientTestConnections("../../tests/payload-example.json", 200, logger)
		_, err := conn.DBGet("5cc042307ccc69ada893144c")
		if err != nil {
			t.Errorf(fmt.Sprintf("Test Get %s returned with error - got (%s) wanted (%s)", "DBGet", "error", "nil"))
		}
	})

	t.Run("DBDelete : should pass", func(t *testing.T) {
		conn := NewClientTestConnections("../../tests/payload-example.json", 200, logger)
		err := conn.DBDelete("5cc042307ccc69ada893144c")
		if err != nil {
			t.Errorf(fmt.Sprintf("Test Delete %s returned with error - got (%s) wanted (%s)", "DBDelete", "error", "nil"))
		}
	})

	t.Run("DBList : should pass", func(t *testing.T) {
		conn := NewClientTestConnections("../../tests/payload-example.json", 200, logger)
		lr := &schema.ListRange{From: 10, To: 20, Search: "NA"}
		s, err := conn.DBList(lr)
		conn.Info("DBList %v\n", s)
		if err != nil {
			t.Errorf(fmt.Sprintf("Test Update %s returned with error - got (%s) wanted (%s)", "DBUpdate", "nil", "error"))
		}
	})

}
