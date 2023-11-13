package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/nats-io/nats.go"
	"gitlab.calendaria.team/services/media/internal/data"
)

type QueueManager struct {
	nc      *data.NatsClient
	log     *log.Helper
	appName string
	queues  map[string]*Queue
}

func NewQueueManager(c *data.Config, nc *data.NatsClient, logger log.Logger) *QueueManager {
	return &QueueManager{
		nc:      nc,
		log:     log.NewHelper(logger),
		appName: c.GetAppName(),
		queues:  make(map[string]*Queue),
	}
}

type queueKey struct{}

func (qm *QueueManager) GetLocal(name string) *Queue {
	subj := qm.appName + "/" + name
	queue, ok := qm.queues[subj]
	if ok {
		return queue
	}

	queue = newQueue(qm.nc, qm.log, subj)

	qm.queues[subj] = queue

	return queue
}

func (qm *QueueManager) AddConsumer(name string, handler func(ctx context.Context, m *nats.Msg) bool) {
	subj := qm.appName + "/" + name
	queue := qm.GetLocal(name)

	ctx := context.WithValue(context.Background(), queueKey{}, queue)
	_, err := qm.nc.QueueSubscribe(subj, "workers", func(m *nats.Msg) {
		if !handler(ctx, m) {
			m.Nak()
		}
	})
	if err != nil {
		qm.log.Errorf("nc.QueueSubscribe: %s", err.Error())
	}
}

func (qm *QueueManager) GetRemote(subj string) *Queue {
	queue, ok := qm.queues[subj]
	if ok {
		return queue
	}

	queue = newQueue(qm.nc, qm.log, subj)

	qm.queues[subj] = queue

	return queue
}

type Queue struct {
	nc   *data.NatsClient
	log  *log.Helper
	name string
}

func newQueue(nc *data.NatsClient, log *log.Helper, name string) *Queue {
	return &Queue{
		nc:   nc,
		log:  log,
		name: name,
	}
}

func (q *Queue) Pub(data any) {
	err := q.nc.Publish(q.name, data)
	if err != nil {
		q.log.Warnf("nc.Publish for %s: %s", q.name, err.Error())
	}
}

type Notification struct {
	UsersIds []int64
	Title    string
	Body     string
	Image    string
	Data     map[string]string
}
