package main

import (
	"context"
	"fmt"
	"github.com/pete911/hcr/internal/flag"
	"github.com/pete911/hcr/internal/hcr"
	"github.com/pete911/hcr/internal/logger"
	"go.uber.org/zap/zapcore"
	"os"
)

func main() {
	log, err := logger.NewZapLogger(zapcore.InfoLevel)
	if err != nil {
		fmt.Printf("new zap logger: %v", err)
		os.Exit(1)
	}

	config, err := flag.ParseFlags()
	if err != nil {
		return
	}
	log.Info(config.String())

	if err := hcr.NewReleaser(log, config).Release(context.TODO()); err != nil {
		log.Fatal(fmt.Sprintf("release: %v", err))
	}
}
