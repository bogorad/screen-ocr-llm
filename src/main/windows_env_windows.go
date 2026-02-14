//go:build windows

package main

import (
	"log"
	"syscall"
)

// enableDPIAwareness attempts to set per-monitor DPI awareness on Windows to fix scaling issues.
func enableDPIAwareness() {
	shcore := syscall.NewLazyDLL("Shcore.dll")
	setProcessDpiAwareness := shcore.NewProc("SetProcessDpiAwareness")
	const processPerMonitorDPIAware = 2
	if err := setProcessDpiAwareness.Find(); err == nil {
		ret, _, _ := setProcessDpiAwareness.Call(uintptr(processPerMonitorDPIAware))
		if ret == 0 {
			log.Printf("DPI: Successfully set per-monitor DPI awareness")
		} else {
			log.Printf("DPI: Failed to set per-monitor DPI awareness, error code: %d", ret)
		}
		return
	}

	log.Printf("DPI: Shcore.SetProcessDpiAwareness not available, trying fallback")
	user32 := syscall.NewLazyDLL("user32.dll")
	setProcessDPIAware := user32.NewProc("SetProcessDPIAware")
	if err := setProcessDPIAware.Find(); err == nil {
		ret, _, _ := setProcessDPIAware.Call()
		if ret != 0 {
			log.Printf("DPI: Successfully set system DPI awareness (fallback)")
		} else {
			log.Printf("DPI: Failed to set system DPI awareness (fallback)")
		}
	} else {
		log.Printf("DPI: SetProcessDPIAware not available, no DPI awareness set")
	}
}

func logMonitorConfiguration() {
	user32 := syscall.NewLazyDLL("user32.dll")
	getSystemMetrics := user32.NewProc("GetSystemMetrics")

	// Get monitor count.
	smCMonitors := 80 // SM_CMONITORS
	ret, _, _ := getSystemMetrics.Call(uintptr(smCMonitors))
	monitorCount := int(ret)
	log.Printf("MONITOR: Detected %d monitors", monitorCount)

	// Virtual screen metrics.
	smXVirtualScreen := 76  // SM_XVIRTUALSCREEN
	smYVirtualScreen := 77  // SM_YVIRTUALSCREEN
	smCXVirtualScreen := 78 // SM_CXVIRTUALSCREEN
	smCYVirtualScreen := 79 // SM_CYVIRTUALSCREEN

	vx, _, _ := getSystemMetrics.Call(uintptr(smXVirtualScreen))
	vy, _, _ := getSystemMetrics.Call(uintptr(smYVirtualScreen))
	vw, _, _ := getSystemMetrics.Call(uintptr(smCXVirtualScreen))
	vh, _, _ := getSystemMetrics.Call(uintptr(smCYVirtualScreen))

	log.Printf("MONITOR: Virtual screen - x:%d y:%d w:%d h:%d", vx, vy, vw, vh)

	// Primary screen metrics.
	smCXScreen := 0 // SM_CXSCREEN
	smCYScreen := 1 // SM_CYSCREEN
	pw, _, _ := getSystemMetrics.Call(uintptr(smCXScreen))
	ph, _, _ := getSystemMetrics.Call(uintptr(smCYScreen))
	log.Printf("MONITOR: Primary screen - w:%d h:%d", pw, ph)
}
