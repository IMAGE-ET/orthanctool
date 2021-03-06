package main

import (
	"context"
	"flag"
	"sync"
	"time"

	"github.com/google/subcommands"
	"github.com/levinalex/orthanctool/api"
)

type changesCommand struct {
	cmdArgs             []string
	orthanc             apiFlag
	allChanges          bool
	filter              string
	pollIntervalSeconds int
	sweepSeconds        int
}

func ChangesCommand() *changesCommand {
	return &changesCommand{}
}

func (c changesCommand) Name() string { return "changes" }
func (c changesCommand) Usage() string {
	return c.Name() + ` --orthanc <url> [--all] [--poll] [--sweep=<seconds>] [command...]:
	Iterates over changes in Orthanc.
	Outputs each change as JSON. 
	If command is given, it will be run for each change and JSON will be passed to it via stdin.` + "\n\n"
}
func (c changesCommand) Synopsis() string { return "yield change entries" }

func (c *changesCommand) SetFlags(f *flag.FlagSet) {
	f.Var(&c.orthanc, "orthanc", "Orthanc URL")
	f.IntVar(&c.pollIntervalSeconds, "poll", 60, "poll interval in seconds. Set to 0 to disable polling)")
	f.BoolVar(&c.allChanges, "all", true, "yield past changes")
	f.StringVar(&c.filter, "filter", "", "only output changes of this type")
	f.IntVar(&c.sweepSeconds, "sweep", 0, "yield all existing instances every N seconds. 0 to disable (default). Implies -all")
}

func (c *changesCommand) run(ctx context.Context) error {
	wg := sync.WaitGroup{}
	ctx, cancel := context.WithCancel(ctx)
	errors := make(chan error, 0)
	returnError := readFirstError(errors, func() { cancel() })

	_, lastIndex, err := c.orthanc.Api.LastChange(ctx)
	if err != nil {
		errors <- err
	}

	onChange := func(cng api.ChangeResult) {
		if c.filter == "" || c.filter == cng.ChangeType {
			cmdAction(c.cmdArgs, cng)
		}
	}

	if c.pollIntervalSeconds > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			pollInterval := time.Duration(c.pollIntervalSeconds) * time.Second
			errors <- api.ChangeWatch{
				StartIndex:   lastIndex,
				PollInterval: pollInterval,
			}.Run(ctx, c.orthanc.Api, onChange)
		}()
	}

	if c.allChanges || c.sweepSeconds > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			errors <- api.ChangeWatch{
				StartIndex: 0,
				StopIndex:  lastIndex,
			}.Run(ctx, c.orthanc.Api, onChange)

			if c.sweepSeconds > 0 {
				for {
					time.Sleep(time.Duration(c.sweepSeconds) * time.Second)

					errors <- api.ChangeWatch{
						StartIndex: 0,
						StopAtEnd:  true,
					}.Run(ctx, c.orthanc.Api, onChange)

					if ctx.Err() != nil {
						break
					}
				}
			}
		}()
	}

	wg.Wait()

	close(errors)
	return <-returnError
}

func (c *changesCommand) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	c.cmdArgs = f.Args()[0:]
	err := c.run(ctx)

	if err != nil {
		return fail(err)
	}
	return subcommands.ExitSuccess
}
