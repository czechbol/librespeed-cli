/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"os"

	"github.com/czechbol/librespeedtest/cmd"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func main() {
	log.Out = os.Stdout
	log.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	if err := (&cmd.CLIOptions{}).CobraCommand(log).Execute(); err != nil {
		log.Error(err)
	}
}
