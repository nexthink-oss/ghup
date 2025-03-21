package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nexthink-oss/ghup/cmd"
)

var (
	version string = "snapshot"
	commit  string = "unknown"
	date    string = "unknown"
)

func main() {
	cmd := cmd.New()
	cmd.Version = fmt.Sprintf("%s-%s (built %s)", version, commit, date)
	if err := cmd.ExecuteContext(context.Background()); err != nil {
		os.Exit(1)
	}
}
