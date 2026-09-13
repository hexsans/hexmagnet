package dht

import (
	"github.com/hexsans/hexmagnet/internal/queue"
	"github.com/hexsans/hexmagnet/internal/worker"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// QueueConsumerParams carries the dependencies shared by every DHT queue
// consumer. Consumer packages embed it in their own fx.In params struct and
// add the handler and producer dependencies they need.
type QueueConsumerParams struct {
	fx.In

	ConsumerMaker queue.ConsumerMaker
	Runtime       *queue.Runtime
	Logger        *zap.SugaredLogger
}

// NewWorker builds the shared consumer worker for a DHT topic. logName is the
// logger name suffix (e.g. "message_queue.triage").
func (p QueueConsumerParams) NewWorker(
	name, topic, groupID, logName string,
	handler queue.MessageHandler,
) worker.Worker {
	return NewConsumerWorker(name, p.ConsumerMaker, p.Runtime, topic, groupID, handler, p.Logger.Named(logName))
}
