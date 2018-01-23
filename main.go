package main

import (
	"os"
	"os/signal"
	"syscall"
)

func main() {
	abort := make(chan struct{})
	go func() {
		os.Stdin.Read(make([]byte, 1)) // read a single byte
		abort <- struct{}{}
	}()

	t, err := NewT()
	if err != nil {
		return
	}

	t.Start()

	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2)

		for {
			select {
			case <-sigs:
				t.Close()
				os.Exit(0)
			}
		}

	}()

	<-abort
}
