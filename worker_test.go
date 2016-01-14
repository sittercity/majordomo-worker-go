package majordomo_worker

import (
	"testing"
	"time"

	"git.sittercity.com/core-services/majordomo-worker-go.git/Godeps/_workspace/src/github.com/pebbe/zmq4"
	"git.sittercity.com/core-services/majordomo-worker-go.git/Godeps/_workspace/src/github.com/stretchr/testify/suite"
)

type WorkerTestSuite struct {
	suite.Suite

	ctx *zmq4.Context

	brokerAddress, serviceName           string
	heartbeatInMillis, reconnectInMillis int
	pollInterval, heartbeatLiveness      int

	defaultAction WorkerAction
	logger        *testLogger
}

func (s *WorkerTestSuite) SetupTest() {
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

func (s *WorkerTestSuite) TearDownTest() {
	s.ctx.Term()
}

func (s *WorkerTestSuite) createWorker(heartbeat, reconnect int, action WorkerAction) *mdWorker {
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

func (s *WorkerTestSuite) Test_Receive_DoesNothingExplicitWithHeartbeat() {
	broker := createBroker()
	go broker.run(s.ctx, s.brokerAddress)

	worker := s.createWorker(1000, s.reconnectInMillis, s.defaultAction)

	sendWorkerMessage(broker, MD_HEARTBEAT)
	go worker.Receive()

	time.Sleep(250) // Need to wait a little bit to make sure the poller gets the heartbeat

	// We can ignore the first message for this test, it's the initial READY
	broker.performReceive <- struct{}{}
	<-broker.receivedFromWorker

	sendWorkerMessage(broker, MD_DISCONNECT)

	broker.performReceive <- struct{}{}
	workerMsg := <-broker.receivedFromWorker
	if s.Equal([]byte(MD_READY), workerMsg[3], "Expected second READY after heartbeat") {
		s.Equal([]byte(""), workerMsg[1])
		s.Equal([]byte(MD_WORKER), workerMsg[2])
		s.Equal([]byte(MD_READY), workerMsg[3])
		s.Equal([]byte(s.serviceName), workerMsg[4])
	}

	broker.shutdown <- struct{}{}
	worker.cleanup()
}

func (s *WorkerTestSuite) Test_Receive_IgnoresInvalidMessages() {
	broker := createBroker()
	go broker.run(s.ctx, s.brokerAddress)

	worker := s.createWorker(s.heartbeatInMillis, s.reconnectInMillis, s.defaultAction)
	broker.performReceive <- struct{}{}
	go worker.Receive()

	// Invalid message, this should trigger no response from the worker
	data := [][]byte{[]byte(nil)}
	broker.sendToWorker <- data

	// This disconnect should trigger a READY response
	sendWorkerMessage(broker, MD_DISCONNECT)
	broker.performReceive <- struct{}{}

	// We can ignore the first message for this test, it's the initial READY
	<-broker.receivedFromWorker

	workerMsg := readUntilNonHeartbeat(broker)
	if s.Equal([]byte(MD_READY), workerMsg[3], "Expected second READY after the disconnect") {
		s.Equal([]byte(""), workerMsg[1])
		s.Equal([]byte(MD_WORKER), workerMsg[2])
		s.Equal([]byte(MD_READY), workerMsg[3])
		s.Equal([]byte(s.serviceName), workerMsg[4])
	}

	broker.shutdown <- struct{}{}
	worker.cleanup()
}

func (s *WorkerTestSuite) Test_Receive_IgnoresInvalidCommand() {
	broker := createBroker()
	go broker.run(s.ctx, s.brokerAddress)

	worker := s.createWorker(s.heartbeatInMillis, s.reconnectInMillis, s.defaultAction)
	broker.performReceive <- struct{}{}
	go worker.Receive()

	sendWorkerMessage(broker, "\x06")

	// This disconnect should trigger a READY response
	sendWorkerMessage(broker, MD_DISCONNECT)
	broker.performReceive <- struct{}{}

	// We can ignore the first message for this test, it's the initial READY
	<-broker.receivedFromWorker

	workerMsg := readUntilNonHeartbeat(broker)
	if s.Equal([]byte(MD_READY), workerMsg[3], "Expected second READY after the disconnect") {
		s.Equal([]byte(""), workerMsg[1])
		s.Equal([]byte(MD_WORKER), workerMsg[2])
		s.Equal([]byte(MD_READY), workerMsg[3])
		s.Equal([]byte(s.serviceName), workerMsg[4])
	}

	broker.shutdown <- struct{}{}
	worker.cleanup()
}

func (s *WorkerTestSuite) Test_Receive_SendsHeartbeatIfThresholdHit() {
	broker := createBroker()
	go broker.run(s.ctx, s.brokerAddress)

	// Give a low heartbeat values so we definitely trigger it
	worker := s.createWorker(1, s.reconnectInMillis, s.defaultAction)
	broker.performReceive <- struct{}{}
	go worker.Receive()

	// We can ignore the initial READY
	<-broker.receivedFromWorker

	broker.performReceive <- struct{}{}
	workerMsg := <-broker.receivedFromWorker
	if s.Equal([]byte(MD_HEARTBEAT), workerMsg[3], "Expected HEARTBEAT from worker") {
		s.Equal([]byte(""), workerMsg[1])
		s.Equal([]byte(MD_WORKER), workerMsg[2])
		s.Equal([]byte(MD_HEARTBEAT), workerMsg[3])
	}

	s.Equal(4, len(workerMsg))

	broker.shutdown <- struct{}{}
	worker.cleanup()
}

func (s *WorkerTestSuite) Test_Receive_CallsActionIfRequest() {
	actionCalled := false
	expectedStr := "data"

	broker := createBroker()
	go broker.run(s.ctx, s.brokerAddress)

	workerAction := funcWorkerAction{
		call: func(args [][]byte) [][]byte {
			actionCalled = true
			s.Equal([]byte(expectedStr), args[0])
			return args
		},
	}

	worker := s.createWorker(s.heartbeatInMillis, s.reconnectInMillis, workerAction)
	broker.performReceive <- struct{}{}
	<-broker.receivedFromWorker // Get the initial READY and discard it
	go worker.Receive()

	sendWorkerMessage(broker, MD_REQUEST, []byte(s.serviceName), nil, []byte(expectedStr))

	workerMsg := readUntilNonHeartbeat(broker)
	if s.Equal([]byte(MD_REPLY), workerMsg[3], "Expected REPLY from worker") {
		s.Equal([]byte(""), workerMsg[1])
		s.Equal([]byte(MD_WORKER), workerMsg[2])
		s.Equal([]byte(MD_REPLY), workerMsg[3])
		s.Equal([]byte(s.serviceName), workerMsg[4])
		s.Equal([]byte(""), workerMsg[5])
		s.Equal([]byte(expectedStr), workerMsg[6])
	}

	s.True(actionCalled)

	broker.shutdown <- struct{}{}
	worker.cleanup()
}

func TestWorkerTestSuite(t *testing.T) {
	suite.Run(t, new(WorkerTestSuite))
}
