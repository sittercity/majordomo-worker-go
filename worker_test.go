package majordomo_worker

import (
	"fmt"
	"math/rand"
	"testing"

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

	s.brokerAddress = "inproc://test-worker" + fmt.Sprintf("%d", rand.Int())
	s.serviceName = "test-service"
	s.heartbeatInMillis = 500
	s.reconnectInMillis = 50
	s.pollInterval = 500
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

	// Set the specific durations so we don't run into race conditions for this specific test
	worker := s.createWorker(10000, s.reconnectInMillis, s.defaultAction)
	broker.performReceive <- struct{}{}
	go worker.Receive()

	sendWorkerMessage(broker, MD_HEARTBEAT)
	broker.performReceive <- struct{}{}

	// We can ignore the first message for this test, it's the initial READY
	<-broker.receivedFromWorker
	workerMsg2 := <-broker.receivedFromWorker
	if s.Equal([]byte(MD_READY), workerMsg2[3], "Expected second READY after heartbeat") {
		s.Equal([]byte(""), workerMsg2[1])
		s.Equal([]byte(MD_WORKER), workerMsg2[2])
		s.Equal([]byte(MD_READY), workerMsg2[3])
		s.Equal([]byte(s.serviceName), workerMsg2[4])
	}

	broker.shutdown <- struct{}{}
	worker.cleanup()
}

func (s *WorkerTestSuite) Test_Receive_IgnoresInvalidMessages() {
	broker := createBroker()
	go broker.run(s.ctx, s.brokerAddress)

	// Set high heartbeat/reconnect so we don't get HEARTBEAT/READY commands
	worker := s.createWorker(10000, 10000, s.defaultAction)
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
	workerMsg2 := <-broker.receivedFromWorker
	if s.Equal([]byte(MD_READY), workerMsg2[3], "Expected second READY after the disconnect") {
		s.Equal([]byte(""), workerMsg2[1])
		s.Equal([]byte(MD_WORKER), workerMsg2[2])
		s.Equal([]byte(MD_READY), workerMsg2[3])
		s.Equal([]byte(s.serviceName), workerMsg2[4])
	}

	broker.shutdown <- struct{}{}
	worker.cleanup()
}

func (s *WorkerTestSuite) Test_Receive_IgnoresInvalidCommand() {
	broker := createBroker()
	go broker.run(s.ctx, s.brokerAddress)

	// Set high heartbeat so we don't get HEARTBEAT/READY commands
	worker := s.createWorker(10000, 100, s.defaultAction)
	broker.performReceive <- struct{}{}
	go worker.Receive()

	sendWorkerMessage(broker, "\x06")

	// This disconnect should trigger a READY response
	sendWorkerMessage(broker, MD_DISCONNECT)
	broker.performReceive <- struct{}{}

	// We can ignore the first message for this test, it's the initial READY
	<-broker.receivedFromWorker
	workerMsg2 := <-broker.receivedFromWorker
	if s.Equal([]byte(MD_READY), workerMsg2[3], "Expected second READY after the disconnect") {
		s.Equal([]byte(""), workerMsg2[1])
		s.Equal([]byte(MD_WORKER), workerMsg2[2])
		s.Equal([]byte(MD_READY), workerMsg2[3])
		s.Equal([]byte(s.serviceName), workerMsg2[4])
	}

	broker.shutdown <- struct{}{}
	worker.cleanup()
}

func (s *WorkerTestSuite) Test_Receive_SendsHeartbeatIfThresholdHit() {
	broker := createBroker()
	go broker.run(s.ctx, s.brokerAddress)

	// Give a low heartbeat values so we definitely trigger it
	worker := s.createWorker(1, 10000, s.defaultAction)
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

	worker := s.createWorker(10000, s.reconnectInMillis, workerAction)
	broker.performReceive <- struct{}{}
	<-broker.receivedFromWorker // Get the initial READY and discard it
	go worker.Receive()

	sendWorkerMessage(broker, MD_REQUEST, []byte(s.serviceName), nil, []byte(expectedStr))

	// Listen and discard all READY commands, we don't care about them for this test
	var workerMsg [][]byte
	for {
		broker.performReceive <- struct{}{}
		workerMsg = <-broker.receivedFromWorker

		if string(workerMsg[3]) != MD_READY {
			break
		}
	}

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
