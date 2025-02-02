package cmd

import (
	"sync"

	"github.com/creasty/defaults"
)

var defaultsOnce sync.Once

func loadDefaults() {
	if err := defaults.Set(&localRepo); err != nil {
		panic(err)
	}
}
