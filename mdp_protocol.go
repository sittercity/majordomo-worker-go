package majordomo_worker

const (
	MD_WORKER = "MDPW01"

	MD_READY      = "\x01"
	MD_REQUEST    = "\x02"
	MD_REPLY      = "\x03"
	MD_HEARTBEAT  = "\x04"
	MD_DISCONNECT = "\x05"
)
