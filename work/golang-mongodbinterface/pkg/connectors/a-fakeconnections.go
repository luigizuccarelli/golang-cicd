// +build ignore test

package connectors

import (
	"errors"
	"net/http"

	"gitea-cicd.apps.aws2-dev.ocp.14west.io/cicd/golang-mongodbinterface/pkg/schema"
	"github.com/globalsign/mgo/bson"
	"github.com/microlib/simple"
)

// This file is used for live connections and is included in the build but excluded for testing
// It uises the header directive // +build !test
// Used with the -tags=test flag when testing

type FakeRedis struct {
}

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
	Err  bool
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
	var s **schema.SchemaInterface
	for _, x := range docs {
		s = x.(**schema.SchemaInterface)
	}
	data := **s
	if data.MetaInfo == "ERROR" {
		return errors.New("Forced Error")
	}
	return nil
}

// Remove fake.
func (fc FakeCollection) Remove(selector interface{}) error {
	return nil
}

// Update fake.
func (fc FakeCollection) Update(selector interface{}, update interface{}) error {
	var s schema.SchemaInterface
	s = update.(schema.SchemaInterface)
	if s.MetaInfo == "ERROR" {
		return errors.New("Forced Error")
	}
	return nil
}

// EnsurIndex fake.
func (fc FakeCollection) EnsureIndex(index interface{}) error {
	return nil
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

type Connections struct {
	Http  *http.Client
	Redis *FakeRedis
	l     *simple.Logger
	DB    SessionInterface
	Name  string
}
