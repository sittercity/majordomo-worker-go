package majordomo_worker

import (
	"testing"

	"github.com/pebbe/zmq4"
	"github.com/stretchr/testify/suite"
)

type WorkerTestSuite struct {
	suite.Suite

	ctx *zmq4.Context

	brokerAddress, serviceName           string
	heartbeatInMillis, reconnectInMillis int

	defaultAction WorkerAction
}

func (s *WorkerTestSuite) SetupTest() {
	var err error
	s.ctx, err = zmq4.NewContext()
	if err != nil {
		panic(err)
	}

	s.brokerAddress = "inproc://test-worker"
	s.serviceName = "test-service"
	s.heartbeatInMillis = 25
	s.reconnectInMillis = 25

	s.defaultAction = defaultWorkerAction{}
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
		action,
	)
}

func (s *WorkerTestSuite) Test_Receive_DoesNothingExplicitWithHeartbeat() {
	broker := createBroker()
	go broker.run(s.ctx, s.brokerAddress)

	worker := s.createWorker(s.heartbeatInMillis, s.reconnectInMillis, s.defaultAction)
	broker.performReceive <- struct{}{}

	sendWorkerMessage(broker, MD_HEARTBEAT)
	go worker.Receive()

	// We can ignore the first message for this test, it's the initial READY
	<-broker.receivedFromWorker

	broker.performReceive <- struct{}{}
	workerMsg2 := <-broker.receivedFromWorker
	if s.Equal([]byte(MD_READY), workerMsg2[3], "Expected second READY after heartbeat") {
		s.Equal([]byte(""), workerMsg2[1])
		s.Equal([]byte(MD_WORKER), workerMsg2[2])
		s.Equal([]byte(MD_READY), workerMsg2[3])
		s.Equal([]byte(s.serviceName), workerMsg2[4])
	}

	broker.shutdown <- struct{}{}
	worker.Close()
}

func (s *WorkerTestSuite) Test_Receive_IgnoresInvalidMessages() {
	broker := createBroker()
	go broker.run(s.ctx, s.brokerAddress)

	// Set high heartbeat/reconnect so we don't get HEARTBEAT/READY commands
	worker := s.createWorker(10000, 10000, s.defaultAction)

	// Invalid message, this should trigger no response from the worker
	data := [][]byte{[]byte(nil)}
	broker.sendToWorker <- data
	go worker.Receive()

	// We can ignore the first message for this test, it's the initial READY
	broker.performReceive <- struct{}{}
	<-broker.receivedFromWorker

	// This disconnect should trigger a READY response
	sendWorkerMessage(broker, MD_DISCONNECT)

	broker.performReceive <- struct{}{}
	workerMsg2 := <-broker.receivedFromWorker
	if s.Equal([]byte(MD_READY), workerMsg2[3], "Expected second READY after the disconnect") {
		s.Equal([]byte(""), workerMsg2[1])
		s.Equal([]byte(MD_WORKER), workerMsg2[2])
		s.Equal([]byte(MD_READY), workerMsg2[3])
		s.Equal([]byte(s.serviceName), workerMsg2[4])
	}

	broker.shutdown <- struct{}{}
	worker.Close()
}

func (s *WorkerTestSuite) Test_Receive_SendsHeartbeatIfThresholdHit() {
	broker := createBroker()
	go broker.run(s.ctx, s.brokerAddress)

	// Give a high reconnect so we definitely trigger the heartbeat here
	worker := s.createWorker(1, 10000, s.defaultAction)
	go worker.Receive()

	// We can ignore the initial READY
	broker.performReceive <- struct{}{}
	<-broker.receivedFromWorker

	broker.performReceive <- struct{}{}
	workerMsg := <-broker.receivedFromWorker
	if s.Equal([]byte(MD_HEARTBEAT), workerMsg[3], "Expected HEARTBEAT from worker") {
		s.Equal([]byte(""), workerMsg[1])
		s.Equal([]byte(MD_WORKER), workerMsg[2])
		s.Equal([]byte(MD_HEARTBEAT), workerMsg[3])
	}

	broker.shutdown <- struct{}{}
	worker.Close()
}

func (s *WorkerTestSuite) Test_Receive_CallsActionIfRequest() {
	actionCalled := false
	expectedStr := "data"

	broker := createBroker()
	go broker.run(s.ctx, s.brokerAddress)

	workerAction := funcWorkerAction{
		call: func(args []string) []string {
			actionCalled = true
			s.Equal(expectedStr, args[0])
			return args
		},
	}

	worker := s.createWorker(2000, 2000, workerAction)
	broker.performReceive <- struct{}{}
	<-broker.receivedFromWorker // Get the initial READY and discard it

	sendWorkerMessage(broker, MD_REQUEST, []byte(s.serviceName), nil, []byte(expectedStr))

	go worker.Receive()

	var workerMsg [][]byte

	// Listen and discard all READY commands, we don't care about them for this test
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
	worker.Close()
}

func TestWorkerTestSuite(t *testing.T) {
	suite.Run(t, new(WorkerTestSuite))
}
