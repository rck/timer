package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	var (
		flagBell        = flag.Bool("b", false, "Send bell")
		flagNotifySend  = flag.String("n", "", "Send desktop notification with given title (needs 'notify-send')")
		flagGranularity = flag.Duration("g", 5*time.Second, "Granularity to sleep between checks")
	)
	flag.Parse()

	var sleep time.Duration
	for _, f := range flag.Args() {
		d, err := time.ParseDuration(f)
		if err != nil {
			log.Fatal(err)
		}
		sleep += d
	}
	end := time.Now().Add(sleep)

	signal.Ignore(syscall.SIGTTIN)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGUSR1)
	// syscall.SIGUSR1: write current info
	// syscall.SIGTTIN: ignore, we would get it on `timer ... &` for the Scanln() over and over again
	//                  see man 2 read on EIO

	stdin := make(chan struct{}, 1)
	go func() {
		for {
			if _, err := fmt.Scanln(); err != nil {
				// most likely backgrounded; just give up
				return
			}
			stdin <- struct{}{}
		}
	}()

	ticker := time.Tick(*flagGranularity)

	for {
		select {
		case sig := <-sigs:
			if sig == syscall.SIGUSR1 {
				status := status(end)
				notifyStderr(status)
				maybeNotifyDesktop(*flagNotifySend, status)
			}
		case <-stdin:
			notifyStderr(status(end))
		case <-ticker:
			if time.Now().After(end) {
				if *flagBell {
					fmt.Print("\a")
				}
				maybeNotifyDesktop(*flagNotifySend, fmt.Sprintf("%v", sleep))
				return
			}
		}
	}
}

func status(end time.Time) string {
	return fmt.Sprintf("%s left, sleeping till %s\n", end.Sub(time.Now()).Round(time.Second).String(), end.Format("15:04:05"))
}

func notifyStderr(status string) {
	fmt.Fprintln(os.Stderr, status)
}
func notifyDesktop(summary, status string) {
	exec.Command("notify-send", summary, status).Run()
}
func maybeNotifyDesktop(summary, status string) {
	if summary != "" {
		notifyDesktop(summary, status)
	}
}
