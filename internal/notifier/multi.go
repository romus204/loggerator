package notifier

import "sync"

type MultiNotifier struct {
	notifiers []Notifier
}

func NewMultiNotifier(notifiers ...Notifier) *MultiNotifier {
	return &MultiNotifier{notifiers: notifiers}
}

func (m *MultiNotifier) Send(msg string, container string) {
	for _, n := range m.notifiers {
		n.Send(msg, container)
	}
}

func (m *MultiNotifier) StartSendWorker(wg *sync.WaitGroup) {
	for _, n := range m.notifiers {
		n.StartSendWorker(wg)
	}
}
