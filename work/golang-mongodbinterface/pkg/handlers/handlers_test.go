package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	//"reflect"
	"sync"
	"testing"
	"time"

	"gitea-cicd.apps.aws2-dev.ocp.14west.io/cicd/golang-mongodbinterface/pkg/connectors"
	"gitea-cicd.apps.aws2-dev.ocp.14west.io/cicd/golang-mongodbinterface/pkg/schema"
	"github.com/globalsign/mgo/bson"
	"github.com/microlib/simple"
)

const (
	DBINSERT  string = "DBInsert : "
	DBUPDATE  string = "DBUpdate : "
	DBDELETE  string = "DBDelete : "
	DBLIST    string = "DBList : "
	DBGET     string = "DBGet : "
	DBCOUNT   string = "DBCount : "
	DBSCHEMA  string = "customer"
	DBSESSION string = "Failed to clone session"
	OK        string = "OK"
	DATABASE  string = ""
)

type FakeConnections struct {
	Http  *http.Client
	Redis *MemoryCache
	l     *simple.Logger
	DB    SessionInterface
	Name  string
}

// MemoryCache
type MemoryCache struct {
	m   map[string]string
	lck sync.RWMutex
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
type DataLayer interface {
	C(name string) Collection
}

// Collection is an interface to access to the collection struct.
type Collection interface {
	Find(query interface{}) Query
	FindId(query interface{}) Query
	Count() (n int, err error)
	Insert(docs ...interface{}) error
	Remove(selector interface{}) error
	Update(selector interface{}, update interface{}) error
	EnsureIndex(index interface{}) error
}

type Query interface {
	All(result interface{}) error
	One(result interface{}) error
	Sort(field interface{}) Query
	Count() (int, error)
	Limit(val int) Query
	Skip(val int) Query
	Iter() FakeIter
}

type Iter interface {
	Next(data interface{}) bool
	Err() error
	Close()
}

// Session is an interface to access to the Session struct.
type SessionInterface interface {
	DB(name string) DataLayer
	Close()
	Clone() SessionInterface
}

// FakeSession satisfies Session and act as a mock of *mgo.session.
type FakeSession struct{}

// NewFakeSession mock NewSession.
func NewFakeSession() SessionInterface {
	return FakeSession{}
}

func (fs FakeSession) Clone() SessionInterface {
	return FakeSession{}
}

// Close fakes mgo.Session.Close().
func (fs FakeSession) Close() {}

// DB fakes mgo.Session.DB().
func (fs FakeSession) DB(name string) DataLayer {
	fakeDatabase := FakeDatabase{}
	return fakeDatabase
}

// FakeDatabase satisfies DataLayer and act as a mock.
type FakeDatabase struct{}

// C fakes mgo.Database(name).Collection(name).
func (db FakeDatabase) C(name string) Collection {
	return FakeCollection{Name: name}
}

// FakeCollection satisfies Collection and act as a mock.
type FakeCollection struct {
	Name string
}

// Find fake.
func (fc FakeCollection) Find(query interface{}) Query {
	fq := FakeQuery{Name: fc.Name}
	return fq
}

// Find fake.
func (fc FakeCollection) FindId(query interface{}) Query {
	fq := FakeQuery{Name: fc.Name}
	return fq
}

// Count fake.
func (fc FakeCollection) Count() (n int, err error) {
	return 10, nil
}

// Insert fake.
func (fc FakeCollection) Insert(docs ...interface{}) error {
	return nil
}

// Remove fake.
func (fc FakeCollection) Remove(selector interface{}) error {
	return nil
}

// Update fake.
func (fc FakeCollection) Update(selector interface{}, update interface{}) error {
	return nil
}

// EnsurIndex fake.
func (fc FakeCollection) EnsureIndex(index interface{}) error {
	return nil
}

// GetMyDocuments fake.
func (fc FakeCollection) GetMyDocuments(file string) ([]interface{}, error) {
	var documents []interface{}
	content, _ := ioutil.ReadFile(file)
	json.Unmarshal(content, &documents)

	return documents, nil
}

// FakeQuery satisfies Query and act as a mock.
type FakeQuery struct {
	Name string
}

// All fake.
func (fq FakeQuery) All(result interface{}) error {
	return nil
}

// One fake.
func (fq FakeQuery) One(result interface{}) error {
	custom := schema.CustomDetail{Name: "test", Surname: "test", Email: "test@test.com"}
	*result.(*schema.SchemaInterface) = schema.SchemaInterface{ID: bson.ObjectIdHex("5cc042307ccc69ada893144c"), LastUpdate: 123434, MetaInfo: "Fake data", Custom: custom}
	return nil
}

// Distinct fake.
func (fq FakeQuery) Distinct(field string, result interface{}) error {
	return nil
}

// Distinct fake.
func (fq FakeQuery) Count() (int, error) {
	return 10, nil
}

type FakeIter struct {
}

func (f FakeQuery) Iter() FakeIter {
	fio := FakeIter{}
	return fio
}

// Sort fake.
func (fq FakeQuery) Sort(field interface{}) Query {
	return fq
}

// Limit fake.
func (fq FakeQuery) Limit(val int) Query {
	return fq
}

func (fq FakeQuery) Skip(val int) Query {
	return fq
}

func (fi FakeIter) Next(data interface{}) bool {
	custom := schema.CustomDetail{Name: "test", Surname: "test", Email: "test@test.com"}
	*data.(*schema.SchemaInterface) = schema.SchemaInterface{ID: bson.ObjectIdHex("5cc042307ccc69ada893144c"), LastUpdate: 123434, MetaInfo: "Fake data", Custom: custom}
	return false
}

func (fi FakeIter) Err() error {
	return nil
}

func (fi FakeIter) Close() {
}

func NewClientTestConnections(file string, status int, logger *simple.Logger) connectors.Clients {

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

	redisClient := &MemoryCache{m: make(map[string]string)}
	mgo := NewFakeSession()

	return &FakeConnections{Http: httpClient, Redis: redisClient, DB: mgo, Name: "FakeConnections", l: logger}
}

func assertEqual(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Fatalf("%s != %s", a, b)
	}
}

// implementation

// fake redis Get
func (r *FakeConnections) Get(key string) (string, error) {
	r.Redis.lck.RLock()
	defer r.Redis.lck.RUnlock()
	val, ok := r.Redis.m[key]
	if !ok {
		return "", errors.New("Not found")
	}
	return val, nil
}

// fake redis Set
func (r *FakeConnections) Set(key string, value string, expr time.Duration) (string, error) {
	r.Redis.lck.Lock()
	defer r.Redis.lck.Unlock()
	r.Redis.m[key] = value
	return value, nil
}

// fake redis Close
func (r *FakeConnections) Del(key string) error {
	r.Redis.lck.Lock()
	defer r.Redis.lck.Unlock()
	delete(r.Redis.m, key)
	return nil
}

func (r *FakeConnections) Close() error {
	r.Redis = nil
	return nil
}

func (r *FakeConnections) Do(req *http.Request) (*http.Response, error) {
	return r.Http.Do(req)
}

func (r *FakeConnections) DBInsert(body []byte) error {
	return nil
}

func (r *FakeConnections) DBGet(id string) (schema.SchemaInterface, error) {
	custom := schema.CustomDetail{Name: "test", Surname: "test", Email: "test@test"}
	d := schema.SchemaInterface{ID: bson.ObjectIdHex("5cc042307ccc69ada893144c"), LastUpdate: 1323434, MetaInfo: "nada", Custom: custom}
	return d, nil
}

func (r *FakeConnections) DBUpdate(body []byte) (schema.SchemaInterface, error) {
	custom := schema.CustomDetail{Name: "test", Surname: "test", Email: "test@test"}
	d := schema.SchemaInterface{ID: bson.ObjectIdHex("5cc042307ccc69ada893144c"), LastUpdate: 1323434, MetaInfo: "nada", Custom: custom}
	return d, nil
}

func (r *FakeConnections) DBDelete(id string) error {
	return nil
}

func (r *FakeConnections) DBList(lr *schema.ListRange) ([]schema.SchemaInterface, error) {
	var p []schema.SchemaInterface
	custom := schema.CustomDetail{Name: "test", Surname: "test", Email: "test@test"}
	d := schema.SchemaInterface{ID: bson.ObjectIdHex("5cc042307ccc69ada893144c"), LastUpdate: 1323434, MetaInfo: "nada", Custom: custom}
	p = append(p, d)
	return p, nil
}

func (r *FakeConnections) Error(msg string, val ...interface{}) {
	r.l.Error(fmt.Sprintf(msg, val...))
}

func (r *FakeConnections) Info(msg string, val ...interface{}) {
	r.l.Info(fmt.Sprintf(msg, val...))
}

func (r *FakeConnections) Debug(msg string, val ...interface{}) {
	r.l.Debug(fmt.Sprintf(msg, val...))
}

func (r *FakeConnections) Trace(msg string, val ...interface{}) {
	r.l.Trace(fmt.Sprintf(msg, val...))
}

func TestHandlers(t *testing.T) {

	logger := &simple.Logger{Level: "info"}

	t.Run("IsAlive : should pass", func(t *testing.T) {
		var STATUS int = 200
		// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
		rr := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v2/sys/info/isalive", nil)
		NewClientTestConnections("../../tests/payload-example.json", STATUS, logger)
		handler := http.HandlerFunc(IsAlive)
		handler.ServeHTTP(rr, req)

		body, e := ioutil.ReadAll(rr.Body)
		if e != nil {
			t.Fatalf("Should not fail : found error %v", e)
		}
		logger.Trace(fmt.Sprintf("Response %s", string(body)))
		// ignore errors here
		if rr.Code != STATUS {
			t.Errorf(fmt.Sprintf("Handler %s returned with incorrect status code - got (%d) wanted (%d)", "IsAlive", rr.Code, STATUS))
		}
	})

	t.Run("DBInsert : should pass", func(t *testing.T) {
		var STATUS int = 201
		// insert a good peices of data
		custom := schema.CustomDetail{Name: "test", Surname: "test", Email: "test@test"}
		d := schema.SchemaInterface{LastUpdate: time.Now().UnixNano(), MetaInfo: "nada", Custom: custom}
		b, _ := json.Marshal(d)
		rr := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/object", bytes.NewBuffer(b))
		conn := NewClientTestConnections("../../tests/payload-example.json", STATUS, logger)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			MiddlewareHandler(w, r, conn, "DBInsert")
		})

		handler.ServeHTTP(rr, req)
		body, _ := ioutil.ReadAll(rr.Body)
		logger.Info(fmt.Sprintf("Response %s", string(body)))
		if rr.Code != STATUS {
			t.Errorf(fmt.Sprintf("Handler %s returned with no error - got (%d) wanted (%d)", "DBInsert", rr.Code, STATUS))
		}
	})

	t.Run("DBUpdate : should pass", func(t *testing.T) {
		var STATUS int = 200
		// insert a good peices of data
		custom := schema.CustomDetail{Name: "test", Surname: "test", Email: "test@test"}
		d := schema.SchemaInterface{LastUpdate: time.Now().UnixNano(), MetaInfo: "nada", Custom: custom}
		b, _ := json.Marshal(d)
		rr := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/api/v1/object", bytes.NewBuffer(b))
		conn := NewClientTestConnections("../../tests/payload-example.json", STATUS, logger)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			MiddlewareHandler(w, r, conn, "DBUpdate")
		})

		handler.ServeHTTP(rr, req)
		body, _ := ioutil.ReadAll(rr.Body)
		logger.Info(fmt.Sprintf("Response %s", string(body)))
		if rr.Code != STATUS {
			t.Errorf(fmt.Sprintf("Handler %s returned with no error - got (%d) wanted (%d)", "DBUpdate", rr.Code, STATUS))
		}
	})

	t.Run("DBGet : should pass", func(t *testing.T) {
		var STATUS int = 200
		// insert a good peices of data
		custom := schema.CustomDetail{Name: "test", Surname: "test", Email: "test@test"}
		d := schema.SchemaInterface{LastUpdate: time.Now().UnixNano(), MetaInfo: "nada", Custom: custom}
		b, _ := json.Marshal(d)
		rr := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/api/v1/object", bytes.NewBuffer(b))
		conn := NewClientTestConnections("../../tests/payload-example.json", STATUS, logger)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			MiddlewareHandler(w, r, conn, "DBGet")
		})

		handler.ServeHTTP(rr, req)
		body, _ := ioutil.ReadAll(rr.Body)
		logger.Info(fmt.Sprintf("Response %s", string(body)))
		if rr.Code != STATUS {
			t.Errorf(fmt.Sprintf("Handler %s returned with no error - got (%d) wanted (%d)", "DBGet", rr.Code, STATUS))
		}
	})

	t.Run("DBDelete : should pass", func(t *testing.T) {
		var STATUS int = 200
		// insert a good peices of data
		custom := schema.CustomDetail{Name: "test", Surname: "test", Email: "test@test"}
		d := schema.SchemaInterface{LastUpdate: time.Now().UnixNano(), MetaInfo: "nada", Custom: custom}
		b, _ := json.Marshal(d)
		rr := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/api/v1/object", bytes.NewBuffer(b))
		conn := NewClientTestConnections("../../tests/payload-example.json", STATUS, logger)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			MiddlewareHandler(w, r, conn, "DBDelete")
		})

		handler.ServeHTTP(rr, req)
		body, _ := ioutil.ReadAll(rr.Body)
		logger.Info(fmt.Sprintf("Response %s", string(body)))
		if rr.Code != STATUS {
			t.Errorf(fmt.Sprintf("Handler %s returned with no error - got (%d) wanted (%d)", "DBDelete", rr.Code, STATUS))
		}
	})

	t.Run("DBList : should pass", func(t *testing.T) {
		var STATUS int = 200
		// insert a good peices of data
		rr := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/object", nil)
		conn := NewClientTestConnections("../../tests/payload-example.json", STATUS, logger)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			MiddlewareHandler(w, r, conn, "DBList")
		})

		handler.ServeHTTP(rr, req)
		body, _ := ioutil.ReadAll(rr.Body)
		logger.Info(fmt.Sprintf("Response %s", string(body)))
		if rr.Code != STATUS {
			t.Errorf(fmt.Sprintf("Handler %s returned with no error - got (%d) wanted (%d)", "DBDelete", rr.Code, STATUS))
		}
	})

	/*

		func TestUpdate(t *testing.T) {
			// this test should fail
			// read our config (we assume passes)
			var b []byte
			_, err := DbUpdate(b)

			if err == nil {
				t.Errorf("DBInsert should fail with unexpected end of json file")
			}

			// insert a good peices of data
			profile := Profile{Name: "test", Surname: "test", Email: "test@test", Phone: "2113213", Address: "test road", Unit: "unit 1", City: "test city", State: "test state", Zip: "test zip", Country: "this country"}
			d := SchemaInterface{Last: time.Now(), MetaInfo: "nada", Schema: profile}
			b, err = json.Marshal(d)

			// fail with auth
			config.MongoDB.Password = "crap"
			_, err = DbUpdate(b)

			if err == nil {
				t.Errorf("DBInsert should fail with auth error")
			}
		}

		func TestGet(t *testing.T) {
			// this test should fail
			// read our config (we assume passes)
			config.MongoDB.Password = "crap"
			_, err := DbGet("dsds")

			if err == nil {
				t.Errorf("DBGet should fail with auth error")
			}

			config.MongoDB.Password = "orders"
			_, err = DbGet("5b910262bbe2a38cbc99fa4c")

			if err == nil {
				t.Errorf("DBGet should fail with not found")
			}
		}

		func TestDelete(t *testing.T) {
			// this test should fail
			// read our config (we assume passes)
			config.MongoDB.Password = "crap"
			err := DbDelete("dsds")

			if err == nil {
				t.Errorf("DBDelete should fail with auth error")
			}

			config.MongoDB.Password = "orders"
			err = DbDelete("5b910262bbe2a38cbc99fa4c")

			if err == nil {
				t.Errorf("DBDelete should fail with not found")
			}
		}

		func TestList(t *testing.T) {
			// this test should fail
			// read our config (we assume passes)
			config.MongoDB.Password = "crap"
			//var payload []SchemaInterface

			lr := ListRange{From: 0, To: 20, Search: ""}
			_, err := DbList(lr)

			if err == nil {
				t.Errorf("DBList should fail with auth error")
			}

			config.MongoDB.Password = "profile"
			_, e := DbList(lr)

			if e != nil {
				t.Errorf("DBList failed %v ", e)
			}
	*/
}
