// +build !test

package connectors

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/globalsign/mgo"
	"github.com/go-redis/redis"
	"github.com/microlib/simple"
)

// This file is used for live connections and is included in the build but excluded for testing
// It uises the header directive // +build !test
// Used with the -tags=test flag when testing

type Connections struct {
	DB    *mgo.Session
	l     *simple.Logger
	Http  *http.Client
	Redis *redis.Client
	Name  string
}

func NewClientConnections(logger *simple.Logger) Clients {
	var mongoDBDialInfo *mgo.DialInfo
	// set up http object
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpClient := &http.Client{Transport: tr}

	if os.Getenv("MONGODB_REPLICASET") == "" {
		// mongodb connection
		mongoDBDialInfo = &mgo.DialInfo{
			Addrs:    []string{os.Getenv("MONGODB_HOST") + ":" + os.Getenv("MONGODB_PORT")},
			Timeout:  40 * time.Second,
			Database: os.Getenv("MONGODB_DATABASE"),
			Username: os.Getenv("MONGODB_USER"),
			Password: os.Getenv("MONGODB_PASSWORD"),
		}
	} else {
		mongoDBDialInfo = &mgo.DialInfo{
			Addrs:          []string{os.Getenv("MONGODB_HOST")},
			Timeout:        40 * time.Second,
			Database:       os.Getenv("MONGODB_DATABASE"),
			Username:       os.Getenv("MONGODB_USER"),
			Password:       os.Getenv("MONGODB_PASSWORD"),
			ReplicaSetName: os.Getenv("MONGODB_REPLICASET"),
		}
	}

	// database setup and init
	ss, e := mgo.DialWithInfo(mongoDBDialInfo)
	if e != nil {
		logger.Error(fmt.Sprintf("Mongodb init %v\n", e.Error()))
		return nil
	}
	ss.SetMode(mgo.Monotonic, true)
	logger.Trace(fmt.Sprintf("Mongodb dialinfo %v\n", mongoDBDialInfo))
	logger.Trace(fmt.Sprintf("Mongodb connection successful %v\n", ss))

	// connect to redis
	redisClient := redis.NewClient(&redis.Options{
		Addr:         os.Getenv("REDIS_HOST") + ":" + os.Getenv("REDIS_PORT"),
		DialTimeout:  10 * time.Second,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		PoolSize:     10,
		PoolTimeout:  30 * time.Second,
		Password:     os.Getenv("REDIS_PASSWORD"),
		DB:           0,
	})

	return &Connections{Http: httpClient, Redis: redisClient, DB: ss, Name: "LiveConnectors", l: logger}
}

func (r *Connections) Get(key string) (string, error) {
	val, err := r.Redis.Get(key).Result()
	return val, err
}

func (r *Connections) Set(key string, value string, expr time.Duration) (string, error) {
	val, err := r.Redis.Set(key, value, expr).Result()
	return val, err
}

func (r *Connections) Close() error {
	r.Redis.Close()
	return nil
}
