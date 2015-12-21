package majordomo_worker

import (
	"testing"

	"git.sittercity.com/core-services/majordomo-worker-go.git/Godeps/_workspace/src/github.com/stretchr/testify/suite"
)

type WorkerConnectTestSuite struct {
	WorkerTestSuite
}

func (suite *WorkerConnectTestSuite) SetupTest() {
	suite.WorkerTestSuite.SetupTest()
}

func (suite *WorkerConnectTestSuite) TearDownTest() {
	suite.WorkerTestSuite.TearDownTest()
}

func (s *WorkerConnectTestSuite) Test_Create_ContactsBroker() {
	broker := createBroker()
	go broker.run(s.ctx, s.brokerAddress)

	worker := s.createWorker(s.heartbeatInMillis, s.reconnectInMillis, s.defaultAction)
	broker.performReceive <- struct{}{}

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

func (s *WorkerConnectTestSuite) Test_Receive_ReconnectsIfDisconnnectReceived() {
	broker := createBroker()
	go broker.run(s.ctx, s.brokerAddress)

	worker := s.createWorker(s.heartbeatInMillis, s.reconnectInMillis, s.defaultAction)
	broker.performReceive <- struct{}{}

	sendWorkerMessage(broker, MD_DISCONNECT)

	go worker.Receive()

	workerMsg1 := <-broker.receivedFromWorker
	if s.Equal([]byte(MD_READY), workerMsg1[3], "Expected first READY") {
		s.Equal([]byte(""), workerMsg1[1])
		s.Equal([]byte(MD_WORKER), workerMsg1[2])
		s.Equal([]byte(MD_READY), workerMsg1[3])
		s.Equal([]byte(s.serviceName), workerMsg1[4])
	}

	broker.performReceive <- struct{}{}
	workerMsg2 := <-broker.receivedFromWorker
	if s.Equal([]byte(MD_READY), workerMsg2[3], "Expected second READY") {
		s.Equal([]byte(""), workerMsg2[1])
		s.Equal([]byte(MD_WORKER), workerMsg2[2])
		s.Equal([]byte(MD_READY), workerMsg2[3])
		s.Equal([]byte(s.serviceName), workerMsg2[4])
	}

	broker.shutdown <- struct{}{}
	worker.cleanup()
}

func (s *WorkerConnectTestSuite) Test_Receive_ReconnectsIfNoBrokerMessageReceived() {
	broker := createBroker()
	go broker.run(s.ctx, s.brokerAddress)

	worker := s.createWorker(s.heartbeatInMillis, s.reconnectInMillis, s.defaultAction)
	broker.performReceive <- struct{}{}

	go worker.Receive()

	// We can ignore the first message for this test, it's the initial READY
	<-broker.receivedFromWorker

	broker.performReceive <- struct{}{}
	workerMsg2 := <-broker.receivedFromWorker
	if s.Equal([]byte(MD_READY), workerMsg2[3], "Expected second READY after timeout") {
		s.Equal([]byte(""), workerMsg2[1])
		s.Equal([]byte(MD_WORKER), workerMsg2[2])
		s.Equal([]byte(MD_READY), workerMsg2[3])
		s.Equal([]byte(s.serviceName), workerMsg2[4])
	}

	broker.shutdown <- struct{}{}
	worker.cleanup()
}

func TestWorkerConnectTestSuite(t *testing.T) {
	suite.Run(t, new(WorkerConnectTestSuite))
}
