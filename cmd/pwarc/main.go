package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/rusni-pyzda/pwitter"
)

func run() error {
	ctx := context.Background()
	auth, err := pwitter.AnonymousAuth(ctx)
	if err != nil {
		return fmt.Errorf("construction anonymous authorizer: %w", err)
	}

	client := &pwitter.Client{Authorizer: auth}

	switch flag.Arg(0) {
	case "tweet":
		t, err := client.TweetDetail(ctx, flag.Arg(1))
		if err != nil {
			return fmt.Errorf("fetching tweet: %w", err)
		}
		fmt.Printf("%s\n", t.RawJSON)
	default:
		return fmt.Errorf("unknown command")
	}

	return nil
}

func main() {
	flag.Parse()
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
	}
}
