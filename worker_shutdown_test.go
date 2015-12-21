package majordomo_worker

import (
	"testing"

	"git.sittercity.com/core-services/majordomo-worker-go.git/Godeps/_workspace/src/github.com/stretchr/testify/suite"
)

type WorkerShutdownTestSuite struct {
	WorkerTestSuite
}

func (suite *WorkerShutdownTestSuite) SetupTest() {
	suite.WorkerTestSuite.SetupTest()
}

func (suite *WorkerShutdownTestSuite) TearDownTest() {
	suite.WorkerTestSuite.TearDownTest()
}

func (s *WorkerShutdownTestSuite) Test_Shutdown() {
	broker := createBroker()
	go broker.run(s.ctx, s.brokerAddress)

	worker := s.createWorker(1000, 1000, s.defaultAction)
	go worker.Receive()

	broker.shutdown <- struct{}{}
	worker.Shutdown()
}

func TestWorkerShutdownTestSuite(t *testing.T) {
	suite.Run(t, new(WorkerShutdownTestSuite))
}
