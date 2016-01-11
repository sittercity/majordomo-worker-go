//
//  Report 0MQ version.
//

package main

import (
	zmq "git.sittercity.com/core-services/majordomo-worker-go.git/Godeps/_workspace/src/github.com/pebbe/zmq4"

	"fmt"
)

func main() {
	major, minor, patch := zmq.Version()
	fmt.Printf("Current 0MQ version is %d.%d.%d\n", major, minor, patch)
}
