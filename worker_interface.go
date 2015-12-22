package majordomo_worker

type WorkerAction interface {
	Call([][]byte) [][]byte
}

type Worker interface {
	Shutdown()
	Receive() ([][]byte, error)
}
