package main

import (
	"amnesiabox/internal/config"
	"amnesiabox/internal/server"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Read config and parameters
	config, err := config.GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Handle ctrl + c
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM)
	go exitListener(exit)

	// Start the server
	stopChan, err := server.StartServer(config)
	if err != nil {
		log.Fatal(err)
	}
	defer server.StopServer()

	<-stopChan
}

func exitListener(exit chan os.Signal) {
	<-exit
	server.StopServer()
	log.Println("exiting...")
	os.Exit(0)
}
