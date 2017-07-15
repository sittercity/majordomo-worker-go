package majordomo_worker

import (
	"fmt"
	"testing"
	"time"

	"github.com/pebbe/zmq4"
	"github.com/stretchr/testify/suite"
)

type WorkerConnectTestSuite struct {
	suite.Suite

	ctx *zmq4.Context

	brokerAddress, serviceName           string
	heartbeatInMillis, reconnectInMillis int
	pollInterval, heartbeatLiveness      int

	defaultAction WorkerAction
	logger        *testLogger
}

func (s *WorkerConnectTestSuite) SetupTest() {
	var err error
	s.ctx, err = zmq4.NewContext()
	if err != nil {
		panic(err)
	}

	s.brokerAddress = "inproc://test-worker"
	s.serviceName = "test-service"
	s.heartbeatInMillis = 500
	s.reconnectInMillis = 50
	s.pollInterval = 10
	s.heartbeatLiveness = 10

	s.defaultAction = defaultWorkerAction{}
	s.logger = new(testLogger)
}

func (s *WorkerConnectTestSuite) TearDownTest() {
	s.ctx.Term()
}

func (s *WorkerConnectTestSuite) createWorker(heartbeat, reconnect int, action WorkerAction) *mdWorker {
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

func (s *WorkerConnectTestSuite) Test_Create_ContactsBroker() {
	broker := createBroker()
	go broker.run(s.ctx, s.brokerAddress)

	worker := s.createWorker(s.heartbeatInMillis, s.reconnectInMillis, s.defaultAction)
	broker.performReceive <- struct{}{}
	go worker.Receive()

	workerMsg := <-broker.receivedFromWorker
	if s.Equal([]byte(MD_READY), workerMsg[3], "Expected READY") {
		s.Equal([]byte(""), workerMsg[1])
		s.Equal([]byte(MD_WORKER), workerMsg[2])
		s.Equal([]byte(MD_READY), workerMsg[3])
		s.Equal([]byte(s.serviceName), workerMsg[4])
	}

	broker.shutdown <- struct{}{}
	worker.cleanup()
}

func (s *WorkerConnectTestSuite) Test_Create_LogsConnectionAndReady() {
	broker := createBroker()
	go broker.run(s.ctx, s.brokerAddress)

	// Set heartbeat high so we don't get those messages
	worker := s.createWorker(1000, s.reconnectInMillis, s.defaultAction)
	broker.performReceive <- struct{}{}
	go worker.Receive()

	if s.NotEmpty(s.logger.debugs) {
		s.Equal(
			map[string]interface{}{"message": fmt.Sprintf("Attempting connection to broker at '%s'", s.brokerAddress)},
			s.logger.debugs[0],
		)
		s.Equal(
			map[string]interface{}{"message": fmt.Sprintf("Sent command '%s' to broker with message '%q'", MD_READY, [][]byte{})},
			s.logger.debugs[1],
		)
		s.Equal(
			map[string]interface{}{"message": fmt.Sprintf("Connected successfully to broker at '%s'", s.brokerAddress)},
			s.logger.debugs[2],
		)
	}

	broker.shutdown <- struct{}{}
	worker.cleanup()
}

func (s *WorkerConnectTestSuite) Test_Create_ReturnErrorIfConnectionFails() {
	config := WorkerConfig{
		BrokerAddress:        "bad://some-bad-address",
		ServiceName:          s.serviceName,
		HeartbeatInMillis:    time.Duration(1) * time.Millisecond,
		ReconnectInMillis:    time.Duration(1) * time.Millisecond,
		PollingInterval:      time.Duration(1) * time.Millisecond,
		MaxHeartbeatLiveness: 1,
		Action:               s.defaultAction,
	}

	worker, err := newWorker(s.ctx, s.logger, config)
	s.Error(err)

	worker.cleanup()
}

func (s *WorkerConnectTestSuite) Test_Create_HandlesMultipleBrokerAddresses() {
	config := WorkerConfig{
		BrokerAddress:        "inproc://test-worker,inproc://test-worker",
		ServiceName:          s.serviceName,
		HeartbeatInMillis:    time.Duration(1) * time.Millisecond,
		ReconnectInMillis:    time.Duration(1) * time.Millisecond,
		PollingInterval:      time.Duration(1) * time.Millisecond,
		MaxHeartbeatLiveness: 1,
		Action:               s.defaultAction,
	}

	worker, err := newWorker(s.ctx, s.logger, config)
	s.NoError(err)
	s.Equal(2, len(worker.sockets))

	worker.cleanup()
}

func TestWorkerConnectTestSuite(t *testing.T) {
	suite.Run(t, new(WorkerConnectTestSuite))
}
