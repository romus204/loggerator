package kube

import (
	"context"
	"errors"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/romus204/loggerator/internal/config"
	"github.com/romus204/loggerator/internal/telegram"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Kube struct {
	kubeClient *kubernetes.Clientset
	cfg        config.Kube
	bot        *telegram.Bot
	ctx        context.Context
}

type PodContainer struct {
	pod       string
	container []string
}

func NewCubeClient(ctx context.Context, cfg config.Kube, bot *telegram.Bot) *Kube {
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", cfg.KubeConfig)
	if err != nil {
		log.Fatalf("error to create kube config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		log.Fatalf("error to create kube config: %v", err)
	}

	return &Kube{kubeClient: clientset, cfg: cfg, bot: bot, ctx: ctx}
}

func (k *Kube) Subscribe(wg *sync.WaitGroup) {
	pods := k.getPodsWithFilter()
	for _, t := range pods {
		for _, p := range t.container {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-k.ctx.Done():
						return
					default:
						err := k.streamPodLogs(k.cfg.Namespace, t.pod, p)
						if err != nil {
							if errors.Is(err, context.Canceled) {
								return
							}
							log.Printf("error to stream logs: %v", err)
							time.Sleep(5 * time.Second)
						}
					}
				}
			}()

		}

	}
}

func (k *Kube) streamPodLogs(namespace, podName, containerName string) error {
	sinceTime := v1.NewTime(time.Now())

	req := k.kubeClient.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container: containerName,
		Follow:    true,
		SinceTime: &sinceTime,
	})

	stream, err := req.Stream(k.ctx)
	if err != nil {
		return err
	}
	defer stream.Close()

	buf := make([]byte, 4096)
	for {
		select {
		case <-k.ctx.Done():
			return nil
		default:
			n, err := stream.Read(buf)
			if err != nil {
				return err
			}

			if n > 0 {
				logs := string(buf[:n])
				filteredLogs := k.filterLogs(logs, k.cfg.Filter)
				if filteredLogs != "" {
					lines := strings.Split(filteredLogs, "\n")
					for _, line := range lines {
						if line == "" {
							continue
						}
						k.bot.Send(line)
					}
				}
			}

		}
	}
}

func (k *Kube) filterLogs(logs, keyword string) string {
	var filteredLines []string
	lines := strings.Split(logs, "\n")

	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), strings.ToLower(keyword)) {
			filteredLines = append(filteredLines, line)
		}
	}

	return strings.Join(filteredLines, "\n")
}

// return pods with regex filter of config
func (k *Kube) getPodsWithFilter() []PodContainer {
	res := []PodContainer{}
	pods, err := k.kubeClient.CoreV1().Pods(k.cfg.Namespace).List(k.ctx, v1.ListOptions{})
	if err != nil {
		log.Fatalf("Failed to get pods: %v", err)
	}

	for _, cp := range k.cfg.Target {
		regex, err := regexp.Compile(cp.Pod)
		if err != nil {
			log.Fatalf("Failed to compile regex: %v", err)
		}

		for _, pod := range pods.Items {
			if regex.MatchString(pod.Name) {
				containers := make([]string, 0)
				containers = append(containers, cp.Container...)
				res = append(res, PodContainer{pod: pod.Name, container: containers})
			}
		}

	}
	return res
}
