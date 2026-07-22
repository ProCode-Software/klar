package main

import (
	"os"
	"runtime"
	"runtime/pprof"
)

type prof struct {
	file *os.File
}

func startProf() {
	v := os.Getenv("KLAR_PROFILE")
	var path string
	switch v {
	case "":
		return
	case "1":
		path = "klar.prof"
	default:
		path = v
	}
	file, err := os.Create(path) //nolint:gosec // G703 - internal env var only
	if err != nil {
		panic(err)
	}
	profiler.file = file
	runtime.SetCPUProfileRate(10_000)
	if err = pprof.StartCPUProfile(file); err != nil {
		panic(err)
	}
}

func stopProf() {
	if profiler.file == nil {
		return
	}
	pprof.StopCPUProfile()
	profiler.file.Close()
}
