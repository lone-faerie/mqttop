package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"

	"github.com/lone-faerie/mqttop/cmd"
)

func main() {
	runtime.MemProfileRate = 1
	go func() {
		log.Println("starting pprof")
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	if err := cmd.Execute(); err != nil {
		cmd.Error(err)
		if exit, ok := err.(*cmd.ExitError); ok {
			os.Exit(exit.Code)
		}

//		cmd.Error(err)
		cmd.Usage()
	}
}
