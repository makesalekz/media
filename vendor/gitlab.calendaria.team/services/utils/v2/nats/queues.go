package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"gitlab.calendaria.team/services/utils/v1/config"
)

// ------------------------------ Naming -----------------------------------------
type naming struct {
	Subject      string `json:"subject"`
	ConsumerName string `json:"consumer_name"`
}

// getNames function uses to generate standard general name
func getNames(appName, queueName string) naming {
	return naming{
		Subject:      fmt.Sprintf("%s.%s", appName, queueName),
		ConsumerName: fmt.Sprintf("%s_%s", appName, queueName), // consumer name can't contain ., *, >, /, \
	}
}

// ------------------------------ Constants -----------------------------------------
const (
	maxAgeDuration     = 3 * 24 * time.Hour
	maxMessagesInQueue = 1000000
	maxMessageSize     = 5 << 20 // one message maximum size 5mb
)

// ------------------------------ Queue -----------------------------------------
type IQueue interface {
	// Pub publishes the provided data to the queue.
	// It marshals the data to JSON and sends it to the JetStream server.
	Pub(data any)
}

type queueKey struct{}

type Queue struct {
	js   jetstream.JetStream
	log  *log.Helper
	name string
}

// newQueue creates a new Queue instance.
func newQueue(js jetstream.JetStream, log *log.Helper, name string) *Queue {
	return &Queue{
		js:   js,
		log:  log,
		name: name,
	}
}

func (q *Queue) Pub(data any) {
	// marshal data
	msg, err := json.Marshal(data) // JetStream messages need to be []byte
	if err != nil {
		q.log.Errorf("failed to marshal data for %s: %v", q.name, err)
		return
	}

	// initialize context
	ctx := context.WithValue(context.Background(), queueKey{}, q)

	// publish message
	_, err = q.js.Publish(ctx, q.name, msg)
	if err != nil {
		q.log.Errorf("js.Publish for %s: %v", q.name, err)
		return
	}
}

// ------------------------------ Queue Manager -----------------------------------------
type IQueueManager interface {
	// GetLocal retrieves or initializes a local queue by concat appName (service name) and queue name.
	// If the queue already exists, it's returned; otherwise, a new one is created.
	GetLocal(name string) IQueue

	// GetRemote retrieves or initializes a queue based on the given subject.
	// If the queue already exists, it's returned; otherwise, a new one is created.
	GetRemote(subj string) IQueue

	// AddConsumer adds a new consumer to the specified queue.
	// It creates or updates a JetStream stream and consumer,
	// then starts consuming messages and handling them with the provided handler.
	AddConsumer(queueName string, handler func(ctx context.Context, m jetstream.Msg) bool)

	// AddRemoteConsumer adds a new consumer to a remote queue.
	// It creates or updates a JetStream stream and consumer,
	// then starts consuming messages and handling them with the provided handler.
	AddRemoteConsumer(appName, queueName string, handler func(ctx context.Context, m jetstream.Msg) bool)
}

type QueueManager struct {
	js      jetstream.JetStream
	log     *log.Helper
	appName string
	ql      sync.RWMutex
	queues  map[string]*Queue
}

// NewQueueManager initializes a new QueueManager instance,
// establishing a JetStream connection and configuring the manager.
func NewQueueManager(c *config.Config, nc *nats.Conn, logger log.Logger) IQueueManager {
	js, err := jetstream.New(nc)
	if err != nil {
		log.NewHelper(logger).Errorf("failed to initialize JetStream: %v", err)
		return nil
	}

	return &QueueManager{
		js:      js,
		log:     log.NewHelper(logger),
		appName: c.GetAppName(),
		queues:  make(map[string]*Queue),
	}
}

// getQueue retrieves a queue from the manager's queue map.
func (qm *QueueManager) getQueue(subj string) *Queue {
	qm.ql.RLock()
	defer qm.ql.RUnlock()
	return qm.queues[subj]
}

// initQueue retrieves or initializes a queue.
// If the queue doesn't exist, it's created and added to the map.
func (qm *QueueManager) initQueue(subj string) *Queue {
	qm.ql.Lock()
	defer qm.ql.Unlock()

	queue, ok := qm.queues[subj]
	if !ok {
		queue = newQueue(qm.js, qm.log, subj)
		qm.queues[subj] = queue
	}

	return queue
}

func (qm *QueueManager) GetLocal(queueName string) IQueue {
	names := getNames(qm.appName, queueName)

	return qm.GetRemote(names.Subject)
}

func (qm *QueueManager) GetRemote(subject string) IQueue {
	queue := qm.getQueue(subject)
	if queue == nil {
		queue = qm.initQueue(subject)
	}

	return queue
}

func (qm *QueueManager) AddConsumer(queueName string, handler func(ctx context.Context, m jetstream.Msg) bool) {
	qm.AddRemoteConsumer(qm.appName, queueName, handler)
}

func (qm *QueueManager) AddRemoteConsumer(appName, queueName string, handler func(ctx context.Context, m jetstream.Msg) bool) {
	// define names
	names := getNames(appName, queueName)

	// initial variables
	queue := qm.GetRemote(names.Subject)
	ctx := context.WithValue(context.Background(), queueKey{}, queue)
	messageDelays := []time.Duration{5 * time.Second, 15 * time.Second, 30 * time.Minute, 1 * time.Hour, 2 * time.Hour}

	// add stream with service name
	stream, err := qm.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:              appName,
		Description:       fmt.Sprintf("Stream for \"%s\" service", appName),
		Subjects:          []string{fmt.Sprintf("%s.*", appName)},
		MaxAge:            maxAgeDuration,
		MaxMsgsPerSubject: maxMessagesInQueue,
		MaxMsgSize:        maxMessageSize,
	})
	if err != nil {
		qm.log.Errorf("failed to add stream: %v", err)
		return
	}

	// add consumer to stream with subject
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Name:          names.ConsumerName,
		Durable:       names.ConsumerName,
		Description:   fmt.Sprintf("Consumer for queue \"%s\"", names.Subject),
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckAllPolicy,
		ReplayPolicy:  jetstream.ReplayInstantPolicy,
		AckWait:       30 * time.Second,
		MaxDeliver:    len(messageDelays) + 1,
		BackOff:       messageDelays,
		FilterSubject: names.Subject,
	})
	if err != nil {
		qm.log.Errorf("failed to add consumer: %v", err)
		return
	}

	// actions on consume message
	_, err = consumer.Consume(func(msg jetstream.Msg) {
		// check handler for success reply
		if handler(ctx, msg) {
			err2 := msg.Ack()
			if err2 != nil {
				qm.log.Errorf("failed to ack msg: %v", err2)
			}
		}
	})
	if err != nil {
		qm.log.Errorf("failed to consume: %v", err)
		return
	}
}
