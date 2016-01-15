package majordomo_worker

import (
	"testing"

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
	s.heartbeatInMillis = 500
	s.reconnectInMillis = 50
	s.pollInterval = 250
	s.heartbeatLiveness = 10

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
	broker.performReceive <- struct{}{}
	go worker.Receive()

	broker.shutdown <- struct{}{}
	worker.Shutdown()
}

func TestWorkerShutdownTestSuite(t *testing.T) {
	suite.Run(t, new(WorkerShutdownTestSuite))
}
