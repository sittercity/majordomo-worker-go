//+build !test

package majordomo_worker

import (
	"git.sittercity.com/core-services/majordomo-worker-go.git/Godeps/_workspace/src/github.com/pebbe/zmq4"
)

func NewWorker(logger Logger, config WorkerConfig) (Worker, error) {
	context, _ := zmq4.NewContext()
	return newWorker(context, logger, config)
}
