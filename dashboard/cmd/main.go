package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/alireza787b/mavlink-anywhere/dashboard/internal/api"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	listen := flag.String("listen", "127.0.0.1:9070", "Address and port to listen on")
	configPath := flag.String("config", "/etc/mavlink-router/main.conf", "Path to mavlink-router config file")
	envPath := flag.String("env", "/etc/default/mavlink-router", "Path to mavlink-router env file")
	version := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *version {
		fmt.Printf("mavlink-anywhere dashboard %s (built %s)\n", Version, BuildTime)
		os.Exit(0)
	}

	srv := api.NewServer(*configPath, *envPath, Version)

	go func() {
		log.Printf("mavlink-anywhere dashboard %s starting on http://%s", Version, *listen)
		if err := http.ListenAndServe(*listen, srv.Router()); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down dashboard...")
}
