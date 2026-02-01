package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/alecthomas/kong"
	"github.com/hpcloud/tail"

	"github.com/quyxishi/refract"
	"github.com/quyxishi/refract/internal"
	"github.com/quyxishi/refract/internal/block/ipset"
	"github.com/quyxishi/refract/internal/serial"
)

func main() {
	var cli CLI

	parser := kong.Must(&cli,
		kong.Name("refract"),
		kong.Description("A policy enforcement service that provides real-time session concurrency control for Xray-core"),
		kong.Vars{
			"version": refract.Version(),
		},
	)

	_, err := parser.Parse(os.Args[1:])
	parser.FatalIfErrorf(err)

	// *

	logFile, err := os.OpenFile(cli.BlockLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 600)
	if err != nil {
		parser.FatalIfErrorf(fmt.Errorf("unable to open block log file due: %v", err))
	}
	defer logFile.Close()

	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)

	// *

	// Hardcode ipset strategy for this moment (todo!)
	strategy := ipset.IpsetBlockStrategy{
		Timeout:         uint32(cli.Timeout.Seconds()),
		DestinationPort: cli.DestinationPort,
	}

	if err = strategy.Init(); err != nil {
		log.Fatalf("failed to initialize strategy:%s due: %v", strategy.Name(), err)
	}

	events := make(chan serial.LogEntry, 1000)
	handler := internal.LogsHandler{
		Events:   events,
		Window:   cli.Window,
		Timeout:  cli.Timeout,
		Strategy: &strategy,
	}

	// Start the processor in background
	go handler.Serve()

	t, err := tail.TailFile(cli.AccessLog, tail.Config{
		Follow:   true,
		ReOpen:   true, // Handle log rotation
		Poll:     true, // Fallback if inotify fails
		Location: &tail.SeekInfo{Offset: 0, Whence: io.SeekEnd},
		Logger:   tail.DiscardingLogger,
	})
	if err != nil {
		log.Fatalf("failed to tail access.log due: %v\n", err)
	}

	for line := range t.Lines {
		if line.Err != nil {
			continue
		}

		entry, ok := serial.ParseLine(line.Text)
		if ok {
			events <- entry
		}
	}
}
