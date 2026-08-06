// Command solace is a single CLI for deploying and operating Solace PubSub+
// Event Brokers on Kubernetes (via the EventBroker Operator), Docker, or Podman.
// It presents the same lifecycle verbs on every platform. Unsupported — not a
// Solace product.
package main

import (
	"fmt"
	"os"

	"solace/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
