package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/microlib/simple"
)

var (
	logger *simple.Logger
	mu     sync.Mutex
)

func main() {

	if os.Getenv("LOG_LEVEL") == "" {
		logger = &simple.Logger{Level: "trace"}
	} else {
		logger = &simple.Logger{Level: os.Getenv("LOG_LEVEL")}
	}

	err := ValidateEnvars(logger)
	if err != nil {
		os.Exit(1)
	}

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	http.HandleFunc("/api/v1/websocket/streamdata", func(w http.ResponseWriter, r *http.Request) {
		upgrader.CheckOrigin = func(r *http.Request) bool { return true }
		conn, _ := upgrader.Upgrade(w, r, nil)
		defer conn.Close()
		logger.Trace(fmt.Sprintf("Websocket connection %v", conn))
		execProjects(conn, logger)
	})

	logger.Info("Starting websocket on 8080")
	http.ListenAndServe(":8080", nil)

}

func execProjects(conn *websocket.Conn, logger *simple.Logger) {
	var project ProjectDetail

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			logger.Error(fmt.Sprintf("Reading websocket message  %v", err))
			return
		} else {
			if string(message) == "poll" {
				data, _ := ioutil.ReadFile("project.json")
				err := json.Unmarshal([]byte(data), &project)
				if err != nil {
					logger.Error(fmt.Sprintf("Converting project.json  %v", err))
				}
				logger.Info(fmt.Sprintf("Read project file : %s ", string(data)))
				// create lightweight go threads
				for i, _ := range project.Repositories {
					if !project.Repositories[i].Skip {
						go executePipeline(conn, project.Repositories[i], logger)
					} else {
						logger.Warn(fmt.Sprintf("Skipping : Project : %s ", project.Repositories[i].Name))
					}
				}
			} else {
				logger.Info("Simulate test from FE")
				str := "1000-:clear"
				send(conn, str, logger)

				time.Sleep(2 * time.Second)
				str = "1000-1:pending"
				send(conn, str, logger)

				time.Sleep(5 * time.Second)
				str = "1000-1:success"
				send(conn, str, logger)

				time.Sleep(1 * time.Second)
				str = "1000-2:pending"
				send(conn, str, logger)

				time.Sleep(5 * time.Second)
				str = "1000-2:success"
				send(conn, str, logger)

				time.Sleep(1 * time.Second)
				str = "1000-3:skipping"
				send(conn, str, logger)

				time.Sleep(1 * time.Second)
				str = "1000-4:pending"
				send(conn, str, logger)

				time.Sleep(5 * time.Second)
				str = "1000-4:success"
				send(conn, str, logger)

				time.Sleep(1 * time.Second)
				str = "1000-5:pending"
				send(conn, str, logger)

				time.Sleep(5 * time.Second)
				str = "1000-5:error"
				send(conn, str, logger)
			}
		}
	}
}

// utilities

func executePipeline(conn *websocket.Conn, repo Repository, logger *simple.Logger) {
	var pipeline *Pipeline

	workDirPath := repo.WorkDir + "/" + repo.Path
	logger.Info(fmt.Sprintf("Scanning : Project : %s - %s", repo.Name, repo.Path))
	_, errStat := os.Stat(workDirPath)
	if os.IsNotExist(errStat) {
		args := []string{
			"-c",
			"git clone " + repo.Scm,
		}
		res, e := execOS(repo.WorkDir, args, false)
		if e != nil {
			logger.Error(fmt.Sprintf("Std err : %s", res))
			logger.Error(fmt.Sprintf("Command : "+strings.Join(args, " ")+" %v", e))
			return
		}
		logger.Info("Git : clone completed")
	} else {
		logger.Info("Repo : already cloned")
	}

	// we first fetch from master
	args := []string{
		"-c",
		"git fetch origin",
	}
	res, e := execOS(workDirPath, args, false)
	if e != nil {
		logger.Error(fmt.Sprintf("Std err : %s", res))
		logger.Error(fmt.Sprintf("Command : "+strings.Join(args, " ")+" %v", e))
		return
	}
	logger.Info("Completed : git fetch")

	// check local HEAD hash
	args = []string{
		"-c",
		"git rev-parse --short HEAD",
	}

	hashLocal, e := execOS(workDirPath, args, true)
	if e != nil {
		logger.Error(fmt.Sprintf("Std err : %s", hashLocal))
		logger.Error(fmt.Sprintf("Command : "+strings.Join(args, " ")+"%s %v", e))
		return
	}
	logger.Info(fmt.Sprintf("Result : local hash %s", hashLocal))

	// check remote HEAD hash
	args = []string{
		"-c",
		"git rev-parse --short origin/master",
	}
	hashRemote, e := execOS(workDirPath, args, true)
	if e != nil {
		logger.Error(fmt.Sprintf("Std err : %s", hashRemote))
		logger.Error(fmt.Sprintf("Command : "+strings.Join(args, " ")+" %v", e))
		return
	}
	logger.Info(fmt.Sprintf("Result : remote hash %s", hashRemote))

	if (hashLocal != hashRemote) || repo.Force == true {

		if repo.Force {
			logger.Info("Force : repo force flag == true")
		}
		// check out lates from master
		args = []string{
			"-c",
			"git pull origin",
		}
		res, e := execOS(workDirPath, args, false)
		if e != nil {
			logger.Error(fmt.Sprintf("Std err : %s", hashRemote))
			logger.Error(fmt.Sprintf("Command : "+strings.Join(args, " ")+" %v", e))
			return
		}
		time.Sleep(2 * time.Second)
		logger.Info(fmt.Sprintf("Result : git pull origin %s", res))

		file, _ := ioutil.ReadFile(workDirPath + "/cicd.json")
		err := json.Unmarshal([]byte(file), &pipeline)
		if err != nil {
			logger.Error(fmt.Sprintf("Converting cicd.json %v", err))
			return
		}
		logger.Trace(fmt.Sprintf("Schema : %v", pipeline))
		logger.Debug(fmt.Sprintf("Path : %s", repo.Path))

		// we can now start the actual pipeline
		logger.Info("[Start Pipeline]\n")
		removeContents("console/" + repo.Path)
		str := pipeline.Id + "-" + ":clear"
		send(conn, str, logger)
		time.Sleep(2 * time.Second)
		for x, _ := range pipeline.Stages {
			if !pipeline.Stages[x].Skip {
				outLog := fmt.Sprintf("Executing : pipeline stage [%d] : %s", pipeline.Stages[x].Id, pipeline.Stages[x].Name)
				str := pipeline.Id + "-" + strconv.Itoa(pipeline.Stages[x].Id) + ":pending"
				se := send(conn, str, logger)
				if se != nil {
					logger.Error(fmt.Sprintf("Websocket send : %s", se))
				}
				logger.Info(outLog)
				time.Sleep(time.Duration(pipeline.Stages[x].Wait) * 1 * time.Second)
				if pipeline.Stages[x].Name == "Deploy" {
					logger.Info(fmt.Sprintf("Envars : pipeline stage [%s] : %s", pipeline.Stages[x].Name, pipeline.Stages[x].Envars))
					for k, _ := range pipeline.Stages[x].Envars {
						os.Setenv(pipeline.Stages[x].Envars[k].Name, pipeline.Stages[x].Envars[k].Value)
					}
				}
				res, e := execCommand(workDirPath, pipeline.Stages[x].Exec, pipeline.Stages[x].Commands, false)
				if e != nil {
					logger.Error(fmt.Sprintf("Std err : %s", res))
					logger.Error(fmt.Sprintf("Command : "+strings.Join(pipeline.Stages[x].Commands, " ")+" %v", e))
					consoleLog(repo.Path+"/"+strings.ToLower(pipeline.Stages[x].Name), outLog+"\n"+res)
					str := pipeline.Id + "-" + strconv.Itoa(pipeline.Stages[x].Id) + ":error"
					se := send(conn, str, logger)
					if se != nil {
						logger.Error(fmt.Sprintf("Websocket send : %s", se))
					}
					break
				}
				logger.Info(fmt.Sprintf("Result : %s", res))
				consoleLog(repo.Path+"/"+strings.ToLower(pipeline.Stages[x].Name), outLog+"\n"+res)
				str = pipeline.Id + "-" + strconv.Itoa(pipeline.Stages[x].Id) + ":success"
				se = send(conn, str, logger)
				if se != nil {
					logger.Error(fmt.Sprintf("Websocket send : %s", se))
				}
			} else {
				logger.Warn(fmt.Sprintf("Skipping : pipeline stage [%d] : %s", pipeline.Stages[x].Id, pipeline.Stages[x].Name))
				str := pipeline.Id + "-" + strconv.Itoa(pipeline.Stages[x].Id) + ":skipping"
				send(conn, str, logger)
				time.Sleep(2 * time.Second)
			}
		}
		logger.Info("[End Pipeline]")
	} else {
		logger.Info("Hashes are equal")
	}
}

// mutex on websocket wrtite
func send(conn *websocket.Conn, str string, logger *simple.Logger) error {
	mu.Lock()
	defer mu.Unlock()
	logger.Trace(fmt.Sprintf("Sending websocket message %s ", str))
	return conn.WriteMessage(1, []byte(str))
}

func execCommand(path string, c string, params []string, trim bool) (string, error) {
	var stdout, stderr bytes.Buffer
	var out string = ""
	cmd := exec.Command(c, params...)
	cmd.Dir = path
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	outStr, errStr := string(stdout.Bytes()), string(stderr.Bytes())
	if err != nil {
		return errStr, err
	}
	if trim {
		out = outStr[:len(outStr)-1]
	} else {
		out = outStr
	}
	return out, nil
}

func execOS(path string, params []string, trim bool) (string, error) {
	var stdout, stderr bytes.Buffer
	var out string = ""
	cmd := exec.Command("/bin/bash", params...)
	cmd.Dir = path
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	outStr, errStr := string(stdout.Bytes()), string(stderr.Bytes())
	if err != nil {
		return errStr, err
	}
	if trim {
		out = outStr[:len(outStr)-1]
	} else {
		out = outStr
	}
	return out, nil
}

func consoleLog(path string, data string) error {
	os.MkdirAll("console/"+path, os.ModePerm)
	err := ioutil.WriteFile("console/"+path+"/out.txt", []byte(data), 0755)
	if err != nil {
		logger.Error(fmt.Sprintf("Writing log file %v", err))
		return err
	}
	return nil
}

func removeContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}
