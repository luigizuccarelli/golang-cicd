package connectors

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"gitea-cicd.apps.aws2-dev.ocp.14west.io/cicd/golang-mongodbinterface/pkg/schema"
	"github.com/globalsign/mgo/bson"
	"github.com/imdario/mergo"
)

const (
	ID              string = "id"
	DBINSERT        string = "DBInsert : "
	DBUPDATE        string = "DBUpdate : "
	DBDELETE        string = "DBDelete : "
	DBLIST          string = "DBList : "
	DBGET           string = "DBGet : "
	DBCOUNT         string = "DBCount : "
	DBSCHEMA        string = "customer"
	DBSESSION       string = "Failed to clone session"
	CONTENTTYPE     string = "Content-Type"
	APPLICATIONJSON string = "application/json"
	OK              string = "OK"
	FROM            string = "FROM"
	TO              string = "TO"
	SEARCH          string = "SEARCH"
	DATABASE        string = ""
)

func (r *Connections) Do(req *http.Request) (*http.Response, error) {
	return r.Http.Do(req)
}

func (r *Connections) Error(msg string, val ...interface{}) {
	r.l.Error(fmt.Sprintf(msg, val...))
}

func (r *Connections) Info(msg string, val ...interface{}) {
	r.l.Info(fmt.Sprintf(msg, val...))
}

func (r *Connections) Debug(msg string, val ...interface{}) {
	r.l.Debug(fmt.Sprintf(msg, val...))
}

func (r *Connections) Trace(msg string, val ...interface{}) {
	r.l.Trace(fmt.Sprintf(msg, val...))
}

// database crudl implementation

// Insert
func (r *Connections) DBInsert(body []byte) error {
	var data *schema.SchemaInterface
	s := r.DB.Clone()
	defer s.Close()
	c := s.DB(os.Getenv("MONGODB_DATABASENAME")).C(DBSCHEMA)
	e := json.Unmarshal(body, &data)
	if e != nil {
		r.Error(DBINSERT+" %v\n", e)
		return e
	}
	// append time to the schema
	data.LastUpdate = time.Now().UnixNano()
	// collection
	err := c.Insert(&data)
	if err != nil {
		r.Error(DBINSERT+" %v\n", err)
		return err
	}
	// all good
	return nil
}

// Update
func (r *Connections) DBUpdate(body []byte) (schema.SchemaInterface, error) {
	var data, existing schema.SchemaInterface
	s := r.DB.Clone()
	defer s.Close()
	c := s.DB(os.Getenv("MONGODB_DATABASENAME")).C(DBSCHEMA)
	e := json.Unmarshal(body, &data)
	if e != nil {
		r.Error(DBUPDATE+" %v\n", e)
		return data, e
	}
	r.Debug(DBUPDATE+" %v\n", data)
	f := bson.IsObjectIdHex(data.ID.Hex())
	if f == false {
		return data, errors.New("bson ObjectId not valid")
	}
	// first find the collection with the given ID
	err := c.FindId(data.ID).One(&existing)
	if err != nil {
		return data, err
	}
	r.Debug(DBUPDATE+": from database : %v ", existing)
	data.LastUpdate = time.Now().UnixNano()
	// now merge the 2 structs
	em := mergo.Merge(&existing, data, mergo.WithOverride)
	if em != nil {
		return data, em
	}
	// update the merged structs
	query := bson.M{"_id": bson.ObjectIdHex(data.ID.Hex())}
	r.Debug(DBUPDATE+" : merged data : %v ", existing)
	e = c.Update(query, existing)
	if e != nil {
		r.Error(DBUPDATE+" %v\n", e)
		return data, e
	}
	// all good
	return data, nil
}

// DBGet gets the schema/data from the database
func (r *Connections) DBGet(id string) (schema.SchemaInterface, error) {

	var data schema.SchemaInterface
	s := r.DB.Clone()
	defer s.Close()
	c := s.DB(os.Getenv("MONGODB_DATABASENAME")).C(DBSCHEMA)
	// check the bson id
	f := bson.IsObjectIdHex(id)
	if f == false {
		return data, errors.New("bson ObjectId not valid")
	}
	// first find the collection with the given ID
	query := bson.M{"_id": bson.ObjectIdHex(id)}
	e := c.Find(query).One(&data)
	r.Trace("Get : data : %v ", data)
	if e != nil {
		r.Error(DBGET+" %v\n", e)
		return data, e
	}
	// all good
	return data, nil
}

// DBDelete deletes schema/data from the database
func (r *Connections) DBDelete(id string) error {
	s := r.DB.Clone()
	defer s.Close()
	c := s.DB(os.Getenv("MONGODB_DATABASENAME")).C(DBSCHEMA)
	// check the bson id
	f := bson.IsObjectIdHex(id)
	if f == false {
		return errors.New("bson ObjectId not valid")
	}
	// first find the collection with the given ID
	query := bson.M{"_id": bson.ObjectIdHex(id)}
	e := c.Remove(query)
	if e != nil {
		r.Error(DBDELETE+" %v\n", e)
		return e
	}
	// all good
	return nil
}

// DBbList lists a range of data from the database
func (r *Connections) DBList(lr *schema.ListRange) ([]schema.SchemaInterface, error) {

	var data schema.SchemaInterface
	var payload []schema.SchemaInterface

	s := r.DB.Clone()
	defer s.Close()
	c := s.DB(os.Getenv("MONGODB_DATABASENAME")).C(DBSCHEMA)

	// first find the collection with the given ID
	iter := c.Find(nil).Sort("_id").Skip(lr.From).Limit(lr.To).Iter()

	for iter.Next(&data) {
		r.Trace("Data : %v ", data)
		payload = append(payload, data)
	}
	if iter.Err() != nil {
		r.Error(DBLIST+" %v\n", iter.Err())
		iter.Close()
		return payload, iter.Err()
	}
	iter.Close()
	// all good
	return payload, nil
}
