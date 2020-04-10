package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"gitea-cicd.apps.aws2-dev.ocp.14west.io/cicd/golang-mongodbinterface/pkg/connectors"
	"gitea-cicd.apps.aws2-dev.ocp.14west.io/cicd/golang-mongodbinterface/pkg/schema"
	"github.com/gorilla/mux"
)

var (
	ID              string = "id"
	CONTENTTYPE     string = "Content-Type"
	APPLICATIONJSON string = "application/json"
	FROM            string = "from"
	TO              string = "to"
	SEARCH          string = "search"
)

// IsAlive - liveliness and readiness probe check
func IsAlive(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "{\"isalive\": true , \"version\": \""+os.Getenv("VERSION")+"\"}\n")
}

// Send CustomAlert - via slack
//func SendAlert(msg []byte, conn connectors.Clients) error {
//	req, err := http.NewRequest("POST", os.Getenv("SLACK_URL"), bytes.NewBuffer(msg))
//	conn.Debug("SendAlert URL info", os.Getenv("SLACK_URL"))
//	resp, err := conn.Do(req)
//	if err != nil || resp.StatusCode != 200 {
//		conn.Error("SendAlert Failed ", err)
//		return err
//	}
//	conn.Info("SendAlert sent successfully", nil)
//	return nil
//}

// MiddlewareHandler a http response and request wrapper
func MiddlewareHandler(w http.ResponseWriter, r *http.Request, conn connectors.Clients, crudl string) {

	var response *schema.Response
	var payload []schema.SchemaInterface

	w.Header().Set(CONTENTTYPE, APPLICATIONJSON)
	//w.WriteHeader(http.StatusInternalServerError)

	switch {
	case crudl == "DBInsert":
		body, err := ioutil.ReadAll(r.Body)
		_, err = handleError(conn, crudl, payload, err)
		if err == nil {
			err = conn.DBInsert(body)
			payload = append(payload, schema.SchemaInterface{LastUpdate: time.Now().Unix(), MetaInfo: "Database Insert"})
			response, err = handleError(conn, crudl, payload, err)
			if err == nil {
				w.WriteHeader(http.StatusCreated)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
		}
	case crudl == "DBUpdate":
		body, err := ioutil.ReadAll(r.Body)
		_, err = handleError(conn, crudl, payload, err)
		if err == nil {
			p, e := conn.DBUpdate(body)
			p.LastUpdate = time.Now().Unix()
			p.MetaInfo = "Database Update"
			payload = append(payload, p)
			response, err = handleError(conn, crudl, payload, e)
			if err == nil {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
		}
	case crudl == "DBDelete":
		vars := mux.Vars(r)
		err := conn.DBDelete(vars[ID])
		payload = append(payload, schema.SchemaInterface{LastUpdate: time.Now().Unix(), MetaInfo: "Database Delete"})
		response, err = handleError(conn, crudl, payload, err)
		if err == nil {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	case crudl == "DBGet":
		vars := mux.Vars(r)
		p, err := conn.DBGet(vars[ID])
		payload = append(payload, p)
		response, err = handleError(conn, crudl, payload, err)
		if err == nil {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	case crudl == "DBList":
		vars := mux.Vars(r)
		from, _ := strconv.Atoi(vars[FROM])
		to, _ := strconv.Atoi(vars[TO])
		lr := &schema.ListRange{From: from, To: to, Search: vars[SEARCH]}
		p, err := conn.DBList(lr)
		response, err = handleError(conn, crudl, p, err)
		if err == nil {
			w.WriteHeader(http.StatusOK)
		}
	}
	b, _ := json.MarshalIndent(response, "", "	")
	fmt.Fprintf(w, string(b))
}

// utility functions

// handleError - private
func handleError(conn connectors.Clients, crudl string, p []schema.SchemaInterface, err error) (*schema.Response, error) {
	if err != nil {
		conn.Error("MW call  %v "+crudl+"\n", err)
		response := &schema.Response{StatusCode: "500", Status: "KO", Message: fmt.Sprintf("MW call %s %v\n", crudl, err), Payload: p}
		return response, err
	}
	conn.Info("MW call  %s succesfull\n", crudl)
	conn.Trace("MW call  %s details %v\n", crudl, p)
	response := &schema.Response{StatusCode: "200", Status: "OK", Message: fmt.Sprintf("MW call  %s successfull \n", crudl), Payload: p}
	return response, nil
}
