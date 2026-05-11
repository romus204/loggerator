package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/romus204/loggerator/internal/config"
	"github.com/romus204/loggerator/internal/kube"
	"github.com/romus204/loggerator/internal/mattermost"
	"github.com/romus204/loggerator/internal/notifier"
	"github.com/romus204/loggerator/internal/telegram"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	config, err := config.NewConfig(parseConfigPath())
	if err != nil {
		log.Fatal(err)
	}

	var targets []notifier.Notifier
	for _, name := range config.Notifiers {
		switch name {
		case "telegram":
			targets = append(targets, telegram.NewBot(ctx, config.Telegram))
		case "mattermost":
			targets = append(targets, mattermost.NewClient(ctx, config.Mattermost))
		default:
			log.Fatalf("unknown notifier %q: expected \"telegram\" or \"mattermost\"", name)
		}
	}
	if len(targets) == 0 {
		log.Fatal("no notifiers configured")
	}
	n := notifier.NewMultiNotifier(targets...)

	kubeClient := kube.NewCubeClient(ctx, config.Kube, n)

	wg := sync.WaitGroup{}

	kubeClient.Subscribe(&wg)
	n.StartSendWorker(&wg)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	cancel()
	hardExit()

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

func hardExit() {
	go func() {
		time.Sleep(30 * time.Second)
		os.Exit(1)
	}()
}
