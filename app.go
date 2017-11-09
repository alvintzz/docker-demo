package main

import (
	"log"
	"flag"
	"net/http"
	"context"
	"os"
	"os/signal"
	"time"
	"syscall"

	"github.com/alvintzz/docker-demo/thisapp"
)

func PongHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World !!"))
}

func main() {
	var environment = flag.String("env", "development", "Set Environment of apps. Default is development.")
	flag.Parse()

	thisapp := thisapp.NewAppModule(*environment)

	//Create a mux for routing incoming requests
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", PongHandler)

	thisapp.InitHandlers(mux)

	//Create server to serve all incoming request with registered mux
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	//Register stop signal and channel that wait for the signal. If it get the signal,
	//then it will be inserted into the channel
	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)
	signal.Notify(stop, syscall.SIGTERM)
	signal.Notify(stop, syscall.SIGINT)

	//Run the API using go func while the channel waiting for signal
	go func() {
		log.Printf("Listening to port %s", ":8080")
		log.Fatal(server.ListenAndServe())
	}()
	<-stop

	//The apps will go here only if there STOP signal like ctrl+c or log.Fatal
	log.Print("Shutting down server...")

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	server.Shutdown(ctx)

	log.Print("Server gracefully stopped")
}
