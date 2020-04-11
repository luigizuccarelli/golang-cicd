package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/microlib/simple"
	"gopkg.in/robfig/cron.v2"
)

var (
	logger *simple.Logger
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

	// create lightweight go threads
	cr := cron.New()
	cr.AddFunc(os.Getenv("CRON"),
		func() {
			execProjects(logger)
		})
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)

	go func() {
		<-c
		cleanup(cr)
		os.Exit(1)
	}()

	cr.Start()

	for {
		logger.Debug(fmt.Sprintf("NOP sleeping for %s seconds", os.Getenv("SLEEP")))
		s, _ := strconv.Atoi(os.Getenv("SLEEP"))
		time.Sleep(time.Duration(s) * time.Second)
	}
}

func execProjects(logger *simple.Logger) {
	var list ProjectList

	data, _ := ioutil.ReadFile("projects.json")
	err := json.Unmarshal([]byte(data), &list)
	if err != nil {
		logger.Error(fmt.Sprintf("Converting projects.json  %v", err))
		return
	}
	// create lightweight go threads
	for i, _ := range list.Projects {
		if !list.Projects[i].Skip {
			go executePipeline(list.Projects[i], logger)
		} else {
			logger.Warn(fmt.Sprintf("Skipping : Project : %s ", list.Projects[i].Name))
		}
	}
}

// utilities

func executePipeline(project ProjectDetail, logger *simple.Logger) {
	var pipeline *Pipeline

	logger.Info(fmt.Sprintf("Scanning : Project : %s - %s", project.Name, project.Path))
	_, errStat := os.Stat(project.Workdir + "/" + project.Path)
	if os.IsNotExist(errStat) {
		args := []string{
			"-c",
			"git clone " + project.Scm,
		}
		res, e := execOS(project.Workdir, args, false)
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
	res, e := execOS(project.Workdir+"/"+project.Path, args, false)
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

	hashLocal, e := execOS(project.Workdir+"/"+project.Path, args, true)
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
	hashRemote, e := execOS(project.Workdir+"/"+project.Path, args, true)
	if e != nil {
		logger.Error(fmt.Sprintf("Std err : %s", hashRemote))
		logger.Error(fmt.Sprintf("Command : "+strings.Join(args, " ")+" %v", e))
		return
	}
	logger.Info(fmt.Sprintf("Result : remote hash %s", hashRemote))

	if (hashLocal != hashRemote) || project.Force == true {

		if project.Force {
			logger.Info("Force : project force flag == true")
		}
		// check out lates from master
		args = []string{
			"-c",
			"git pull origin",
		}
		res, e := execOS(project.Workdir+"/"+project.Path, args, false)
		if e != nil {
			logger.Error(fmt.Sprintf("Std err : %s", hashRemote))
			logger.Error(fmt.Sprintf("Command : "+strings.Join(args, " ")+" %v", e))
			return
		}
		time.Sleep(2 * time.Second)
		logger.Info(fmt.Sprintf("Result : git pull origin %s", res))

		file, _ := ioutil.ReadFile(project.Workdir + "/" + project.Path + "/cicd.json")
		err := json.Unmarshal([]byte(file), &pipeline)
		if err != nil {
			logger.Error(fmt.Sprintf("Converting cicd.json %v", err))
			return
		}
		logger.Trace(fmt.Sprintf("Schema : %v", pipeline))
		repoPath := project.Path
		logger.Debug(fmt.Sprintf("Path : %s", repoPath))
		time.Sleep(1 * time.Second)

		// we can now start the actual pipeline
		logger.Info("[Start Pipeline]\n")
		removeContents("console/" + repoPath)
		for x, _ := range pipeline.Stages {
			if !pipeline.Stages[x].Skip {
				outLog := fmt.Sprintf("Executing : pipeline stage [%d] : %s", pipeline.Stages[x].Id, pipeline.Stages[x].Name)
				logger.Info(outLog)
				time.Sleep(time.Duration(pipeline.Stages[x].Wait) * time.Second)
				if pipeline.Stages[x].Name == "Deploy" {
					logger.Info(fmt.Sprintf("Envars : pipeline stage [%s] : %s", pipeline.Stages[x].Name, pipeline.Stages[x].Envars))
					for k, _ := range pipeline.Stages[x].Envars {
						os.Setenv(pipeline.Stages[x].Envars[k].Name, pipeline.Stages[x].Envars[k].Value)
					}
				}
				res, e := execCommand(pipeline.Workdir+"/"+repoPath, pipeline.Stages[x].Exec, pipeline.Stages[x].Commands, false)
				if e != nil {
					logger.Error(fmt.Sprintf("Std err : %s", res))
					logger.Error(fmt.Sprintf("Command : "+strings.Join(pipeline.Stages[x].Commands, " ")+" %v", e))
					consoleLog(repoPath+"/"+strconv.Itoa(pipeline.Stages[x].Id), outLog+"\n"+res)
					break
				}
				logger.Info(fmt.Sprintf("Result : %s", res))
				consoleLog(repoPath+"/"+strconv.Itoa(pipeline.Stages[x].Id), outLog+"\n"+res)
			} else {
				logger.Warn(fmt.Sprintf("Skipping : pipeline stage [%d] : %s", pipeline.Stages[x].Id, pipeline.Stages[x].Name))
			}
		}
		logger.Info("[End Pipeline]")
	} else {
		logger.Info("Hashes are equal")
	}
}

func cleanup(c *cron.Cron) {
	logger.Warn("Cleanup resources")
	logger.Info("Terminating")
	c.Stop()
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
	return err
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
