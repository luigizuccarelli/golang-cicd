package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/microlib/simple"
)

// checkEnvars - private function, iterates through each item and checks the required field
func checkEnvar(item string, logger *simple.Logger) error {
	name := strings.Split(item, ",")[0]
	required, _ := strconv.ParseBool(strings.Split(item, ",")[1])
	if os.Getenv(name) == "" {
		if required {
			logger.Error(fmt.Sprintf("%s envar is mandatory please set it", name))
			return errors.New(fmt.Sprintf("%s envar is mandatory please set it", name))
		} else {
			logger.Error(fmt.Sprintf("%s envar is empty please set it", name))
		}
	}
	return nil
}

// ValidateEnvars : public call that groups all envar validations
// These envars are set via the openshift template
// Each microservice will obviously have a diffefrent envars so change where needed
func ValidateEnvars(logger *simple.Logger) error {
	items := []string{
		"LOG_LEVEL,false",
	}
	for x, _ := range items {
		if err := checkEnvar(items[x], logger); err != nil {
			return err
		}
	}
	return nil
}
