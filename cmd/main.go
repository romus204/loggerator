package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/romus204/loggerator/internal/config"
	"github.com/romus204/loggerator/internal/kube"
	"github.com/romus204/loggerator/internal/telegram"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	config, err := config.NewConfig(parseConfigPath())
	if err != nil {
		log.Fatal(err)
	}

	bot := telegram.NewBot(config.Telegram)
	kubeClient := kube.NewCubeClient(ctx, config.Kube, bot)

	wg := sync.WaitGroup{}

	kubeClient.Subscribe(&wg)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	cancel()

	wg.Wait()
}

func parseConfigPath() string {
	var configPath string
	flag.StringVar(&configPath, "config", "", "path to config file")
	flag.Parse()

	if configPath == "" {
		log.Fatal("config path is empty")
	}

	return configPath
}
