package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/mux"
	"github.com/microlib/simple"
)

var (
	logger  *simple.Logger
	counter uint64
)

func startHttpServer(port string, logger *simple.Logger) *http.Server {
	srv := &http.Server{Addr: ":" + port}

	r := mux.NewRouter()
	r.HandleFunc("/api/v1/json", func(w http.ResponseWriter, req *http.Request) {
		JsonHandler(w, req, logger)
	}).Methods("GET", "POST")

	r.HandleFunc("/api/v1/force/{id}/{flag}", func(w http.ResponseWriter, req *http.Request) {
		ForcePipelineHandler(w, req, logger)
	}).Methods("POST")

	r.HandleFunc("/api/v1/pipeline/{repo}/{stage}/{status}", func(w http.ResponseWriter, req *http.Request) {
		PipelineStatusHandler(w, req, logger)
	}).Methods("POST")

	r.HandleFunc("/api/v2/sys/info/isalive", IsAlive).Methods("GET")

	sh := http.StripPrefix("/api/v2/web/", http.FileServer(http.Dir("./simple-kb-html/")))
	r.PathPrefix("/api/v2/web/").Handler(sh)

	http.Handle("/", r)

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			logger.Error("Httpserver: ListenAndServe() error: " + err.Error())
		}
	}()

	return srv
}

func main() {

	if os.Getenv("LOG_LEVEL") != "" {
		logger = &simple.Logger{Level: os.Getenv("LOG_LEVEL")}
	} else {
		logger = &simple.Logger{Level: "info"}
	}

	var port string = "9000"
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}

	srv := startHttpServer(port, logger)
	logger.Info("Starting server on port " + srv.Addr)
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	exit_chan := make(chan int)

	go func() {
		for {
			s := <-c
			switch s {
			case syscall.SIGHUP:
				exit_chan <- 0
			case syscall.SIGINT:
				exit_chan <- 0
			case syscall.SIGTERM:
				exit_chan <- 0
			case syscall.SIGQUIT:
				exit_chan <- 0
			default:
				exit_chan <- 1
			}
		}
	}()

	code := <-exit_chan

	if err := srv.Shutdown(nil); err != nil {
		panic(err)
	}
	logger.Info("Server shutdown successfully")
	os.Exit(code)
}
