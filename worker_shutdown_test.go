package majordomo_worker

import (
	"testing"
	"time"

	"git.sittercity.com/core-services/majordomo-worker-go.git/Godeps/_workspace/src/github.com/pebbe/zmq4"
	"git.sittercity.com/core-services/majordomo-worker-go.git/Godeps/_workspace/src/github.com/stretchr/testify/suite"
)

type WorkerShutdownTestSuite struct {
	suite.Suite

	ctx *zmq4.Context

	brokerAddress, serviceName           string
	heartbeatInMillis, reconnectInMillis int
	pollInterval, heartbeatLiveness      int

	defaultAction WorkerAction
	logger        *testLogger
}

func (s *WorkerShutdownTestSuite) SetupTest() {
	var err error
	s.ctx, err = zmq4.NewContext()
	if err != nil {
		panic(err)
	}

	s.brokerAddress = "inproc://test-worker"
	s.serviceName = "test-service"
	s.heartbeatInMillis = 50
	s.reconnectInMillis = 50
	s.pollInterval = 10
	s.heartbeatLiveness = 5

	s.defaultAction = defaultWorkerAction{}
	s.logger = new(testLogger)
}

func (suite *WorkerShutdownTestSuite) TearDownTest() {
	suite.ctx.Term()
}

func (s *WorkerShutdownTestSuite) createWorker(heartbeat, reconnect int, action WorkerAction) *mdWorker {
	return createWorker(
		s.ctx,
		s.brokerAddress,
		s.serviceName,
		heartbeat,
		reconnect,
		s.pollInterval,
		s.heartbeatLiveness,
		action,
		s.logger,
	)
}

func (s *WorkerShutdownTestSuite) Test_Shutdown() {
	broker := createBroker()
	go broker.run(s.ctx, s.brokerAddress)

	worker := s.createWorker(1000, 1000, s.defaultAction)
	go worker.Receive()

	broker.shutdown <- struct{}{}
	worker.Shutdown()
}

func (s *WorkerShutdownTestSuite) Test_Shutdown_LogsShutdown() {
	broker := createBroker()
	go broker.run(s.ctx, s.brokerAddress)

	// Set the heartbeat high so we don't get those messages
	worker := s.createWorker(100000, s.reconnectInMillis, s.defaultAction)
	go worker.Receive()

	s.logger.reset()

	broker.shutdown <- struct{}{}
	worker.Shutdown()

	time.Sleep(100) // Give some time for the graceful shutdown

	if s.NotEmpty(s.logger.debugs) {
		s.Equal(
			map[string]interface{}{"message": "Worker attempting graceful shutdown..."},
			s.logger.debugs[0],
		)
		s.Equal(
			map[string]interface{}{"message": "Worker socket and context closed successfully"},
			s.logger.debugs[1],
		)
	}
}

func TestWorkerShutdownTestSuite(t *testing.T) {
	suite.Run(t, new(WorkerShutdownTestSuite))
}
