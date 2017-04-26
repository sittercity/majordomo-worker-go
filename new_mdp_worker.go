//+build !test

package majordomo_worker

import (
	"github.com/pebbe/zmq4"
)

func NewWorker(logger Logger, config WorkerConfig) (Worker, error) {
	context, _ := zmq4.NewContext()
	return newWorker(context, logger, config)
}
