package notifier

import "sync"

type Notifier interface {
	Send(msg string, container string)
	StartSendWorker(wg *sync.WaitGroup)
}
