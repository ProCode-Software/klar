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

	// For native FFI running on KlarVM, which is architecture-specific.
	// TODO: if supported by go plugins, add architectures:
	//	arm (arm32), riscv64
	KlarLinux_x86
	KlarLinux_ARM64
	KlarLinux_i386

	KlarMacOS_x86
	KlarMacOS_ARM64

	KlarBSD_x86
	KlarBSD_ARM64
	KlarBSD_i386
)

var (
	TargetList = map[string]Target{
		"js": JavaScript, "klar": KlarVM,
	}
	NativePlatforms = map[string]Platform{
		"linux_x86":   KlarLinux_x86,
		"linux_arm64": KlarLinux_ARM64,
		"linux_i386":  KlarLinux_i386,

		"macos_x86":   KlarMacOS_x86,
		"macos_arm64": KlarMacOS_ARM64,

		"freebsd_x86":   KlarBSD_x86,
		"freebsd_arm64": KlarBSD_ARM64,
		"freebsd_i386":  KlarBSD_i386,
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
	case "freebsd/386":
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
		p = KlarMacOS_ARM64
	default:
		return Double{KlarVM, PlatformUnknown},
			errors.New("current distribution '" + goos + "' not supported yet")
	}
	return Double{KlarVM, p}, nil
}
