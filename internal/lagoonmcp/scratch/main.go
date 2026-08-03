// scratch: temporary manual test for assertCertGood - do not commit
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/uselagoon/lagoon-cli/internal/lagoonmcp"
)

func main() {
	host := flag.String("host", "", "hostname to check (required)")
	timeout := flag.Duration("timeout", 10*time.Second, "dial timeout")
	flag.Parse()

	if *host == "" {
		fmt.Fprintln(os.Stderr, "usage: go run ./internal/lagoonmcp/scratch -host <hostname> [-port 443] [-timeout 10s]")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	result, err := lagoonmcp.CheckRoute(ctx, *host)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FAIL: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("%+v\n", result)
}
