ipackage main

import (
	"context"
	"log"
	"time"

	"probability-engine/internal/engine"
)

func main() {
	log.Println("probability-engine starting")

	_ = engine.New(engine.Options{})

	ctx := context.Background()
	_ = ctx
	_ = time.Now()

	log.Println("probability-engine ready")
}
