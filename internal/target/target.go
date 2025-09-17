package target

import (
	"errors"
	"runtime"
)

type (
	Target   int
	Platform int

	// A Double contains both a [Target] and a [Platform].
	Double struct {
		Target   Target
		Platform Platform
	}
)

const (
	TargetUnknown Target = iota
	JavaScript
	KlarVM
)

const (
	PlatformUnknown Platform = iota

	// JavaScript runtimes
	JSBrowser
	JSNode
	JSDeno
	JSBun
)

var (
	TargetList = map[string]Target{
		"js": JavaScript, "klar": KlarVM,
	}
	PlatformList = map[string]Platform{
		"browser": JSBrowser,
		"node":    JSNode,
		"bun":     JSBun,
		"deno":    JSDeno,
	}
)

func FromCurrent() (Double, error) {
	return FromGoDouble(runtime.GOOS + "/" + runtime.GOARCH)
}

func FromGoDouble(goos string) (Double, error) {
	var p Platform
	switch goos {
	/* case "freebsd/386":
		p = KlarBSD_i386
	case "freebsd/amd64":
		p = KlarBSD_ARM64
	case "freebsd/arm64":
		p = KlarBSD_ARM64
	case "linux/386":
		p = KlarLinux_i386
	case "linux/amd64":
		p = KlarLinux_x86
	case "linux/arm64":
		p = KlarLinux_ARM64
	case "darwin/amd64":
		p = KlarMacOS_x86
	case "darwin/arm64":
		p = KlarMacOS_ARM64 */
	default:
		return Double{KlarVM, PlatformUnknown},
			errors.New("current distribution '" + goos + "' not supported yet")
	}
	return Double{KlarVM, p}, nil
}

func (t Target) String() string {
	return map[Target]string{
		JavaScript:    "js",
		KlarVM:        "klar",
		TargetUnknown: "unknown",
	}[t]
}

func (p Platform) String() string {
	return map[Platform]string{
		JSNode:    "node",
		JSDeno:    "deno",
		JSBun:     "bun",
		JSBrowser: "browser",

		PlatformUnknown: "unknown",
	}[p]
}
