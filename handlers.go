package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/microlib/simple"
)

const (
	CONTENTTYPE     string = "Content-Type"
	APPLICATIONJSON string = "application/json"
)

var (
	payload ProjectDetail
)

func JsonHandler(w http.ResponseWriter, r *http.Request, logger *simple.Logger) {
	var response Response

	addHeaders(w, r)

	pipelines, err := buildSchema(logger)
	if err != nil {
		response = Response{Name: os.Getenv("NAME"), StatusCode: "500", Status: "KO", Message: "Error buildSchema", Payload: pipelines}
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		response = Response{Name: os.Getenv("NAME"), StatusCode: "200", Status: "OK", Message: "Payload buildSchema succeeded", Payload: pipelines}
		w.WriteHeader(http.StatusOK)
	}

	b, _ := json.MarshalIndent(response, "", "	")
	logger.Debug(fmt.Sprintf("JsonHandler response : %s", string(b)))
	fmt.Fprintf(w, string(b))
}

func ForcePipelineHandler(w http.ResponseWriter, r *http.Request, logger *simple.Logger) {
	var response Response
	var project ProjectDetail
	vars := mux.Vars(r)

	addHeaders(w, r)

	id, _ := strconv.Atoi(vars["id"])
	flag, _ := strconv.ParseBool(vars["flag"])

	file, err := ioutil.ReadFile("project.json")
	if err != nil {
		logger.Error(fmt.Sprintf("Reading project.json %v", err))
		response = Response{Name: os.Getenv("NAME"), StatusCode: "500", Status: "KO", Message: "Error reading project.json file ", Payload: []Pipeline{}}
		w.WriteHeader(http.StatusInternalServerError)
		b, _ := json.MarshalIndent(response, "", "	")
		fmt.Fprintf(w, string(b))
		return
	}
	err = json.Unmarshal(file, &project)
	if err != nil {
		logger.Error(fmt.Sprintf("Unmarshalling project.json %v", err))
		response = Response{Name: os.Getenv("NAME"), StatusCode: "500", Status: "KO", Message: "Error unmarshalling project.json file", Payload: []Pipeline{}}
		w.WriteHeader(http.StatusInternalServerError)
		b, _ := json.MarshalIndent(response, "", "	")
		fmt.Fprintf(w, string(b))
		return
	} else {
		project.Repositories[id].Force = flag
		data, _ := json.MarshalIndent(project, "", "  ")
		ioutil.WriteFile("project.json", data, 0755)
		response = Response{Name: os.Getenv("NAME"), StatusCode: "200", Status: "OK", Message: fmt.Sprintf("Repository %d force flag set to %t ", id, flag), Payload: []Pipeline{}}
		w.WriteHeader(http.StatusOK)
	}

	b, _ := json.MarshalIndent(response, "", "	")
	logger.Debug(fmt.Sprintf("JsonHandler response : %s", string(b)))
	fmt.Fprintf(w, string(b))
}

func PipelineStatusHandler(w http.ResponseWriter, r *http.Request, logger *simple.Logger) {
	var response Response
	var stage StageDetail
	vars := mux.Vars(r)

	addHeaders(w, r)

	repo := vars["repo"]
	name := vars["name"]
	status := vars["status"]
	body, _ := ioutil.ReadAll(r.Body)
	stage = StageDetail{Name: name, Status: status, Log: string(body)}
	response = Response{
		Name:       os.Getenv("NAME"),
		StatusCode: "200",
		Status:     "OK",
		Message:    fmt.Sprintf("Pipeline for stage %s status %s ", name, status),
		MetaInfo:   repo,
		Stage:      stage,
		Payload:    []Pipeline{},
	}
	w.WriteHeader(http.StatusOK)

	b, _ := json.MarshalIndent(response, "", "	")
	logger.Debug(fmt.Sprintf("JsonHandler response : %s", string(b)))
	fmt.Fprintf(w, string(b))
}

func buildSchema(logger *simple.Logger) ([]Pipeline, error) {
	var pipeline Pipeline
	var pipelines []Pipeline

	file, err := ioutil.ReadFile("project.json")
	if err != nil {
		logger.Error(fmt.Sprintf("Reading project.json %v", err))
		return pipelines, err
	}
	err = json.Unmarshal(file, &payload)
	logger.Debug(fmt.Sprintf("Payload %v", payload))
	if err != nil {
		logger.Error(fmt.Sprintf("Unmarshalling project.json %v", err))
		return pipelines, err
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient := &http.Client{Transport: tr}

	for x, _ := range payload.Repositories {
		logger.Debug(fmt.Sprintf("Payload %v", payload))
		req, _ := http.NewRequest("GET", payload.Repositories[x].RawUrl, nil)
		req.Header.Set("X-Api-Key", os.Getenv("APIKEY"))
		req.Header.Set("Content-Type", "application/json")
		resp, err := httpClient.Do(req)
		if err != nil {
			logger.Error(fmt.Sprintf("Http request %v", err))
			continue
		}
		defer resp.Body.Close()
		body, e := ioutil.ReadAll(resp.Body)
		if e != nil {
			logger.Error(fmt.Sprintf("Could not read cicd.json file %v", e))
			continue
		}
		err = json.Unmarshal(body, &pipeline)
		if err != nil {
			logger.Error(fmt.Sprintf("Unmarshalling project.json %v", err))
			continue
		}
		pipelines = append(pipelines, pipeline)
	}
	return pipelines, nil
}

func IsAlive(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "{ \"version\" : \""+os.Getenv("VERSION")+"\" , \"name\": \""+os.Getenv("NAME")+"\" }")
}

// headers (with cors) utility
func addHeaders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(CONTENTTYPE, APPLICATIONJSON)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}
