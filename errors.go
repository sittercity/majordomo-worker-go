package majordomo_worker

type GracefulShutdown string

func (e GracefulShutdown) Error() string {
	return "Shutdown signal detected, exiting ..."
}

func (GracefulShutdown) GracefulShutdown() bool { return true }
