package session

import (
	"github.com/rs/zerolog/log"
	"time"
)

func timeTrack(start time.Time, name string) {
	// elapsed := time.Since(start)
	log.Printf("%s took %s", name, time.Since(start))
}
