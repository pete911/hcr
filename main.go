package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pete911/hcr/internal/flag"
	"github.com/pete911/hcr/internal/hcr"
	"github.com/pete911/hcr/internal/logger"
	"go.uber.org/zap/zapcore"
	"os"
)

var Version = "dev"

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
	if config.Version {
		fmt.Println(Version)
		os.Exit(0)
	}

	log.Info(config.String())
	releaser, err := hcr.NewReleaser(log, config)
	if err != nil {
		log.Fatal(fmt.Sprintf("new releaser: %v", err))
	}

	chs, err := releaser.Release(context.TODO())
	if err != nil {
		log.Fatal(fmt.Sprintf("release: %v", err))
	}

	// print released charts
	var out []map[string]string
	for _, ch := range chs {
		out = append(out, map[string]string{"chart": ch.Name(), "version": ch.Metadata.Version, "tag": releaser.GetReleaseTag(ch)})
	}
	b, err := json.Marshal(out)
	if err != nil {
		log.Error(fmt.Sprintf("marshal released charts info: %v", err))
		return
	}
	if b != nil {
		fmt.Println(string(b))
	}
}
