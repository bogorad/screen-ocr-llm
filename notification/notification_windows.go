//go:build windows

package notification

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32                 = syscall.NewLazyDLL("user32.dll")
	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	procMessageBox         = user32.NewProc("MessageBoxW")
	procCreateWindowEx     = user32.NewProc("CreateWindowExW")
	procDefWindowProc      = user32.NewProc("DefWindowProcW")
	procDestroyWindow      = user32.NewProc("DestroyWindow")
	procShowWindow         = user32.NewProc("ShowWindow")
	procSetWindowPos       = user32.NewProc("SetWindowPos")
	procGetSystemMetrics   = user32.NewProc("GetSystemMetrics")
	procSetTimer           = user32.NewProc("SetTimer")
	procKillTimer          = user32.NewProc("KillTimer")
	procRegisterClassEx    = user32.NewProc("RegisterClassExW")
	procUpdateWindow       = user32.NewProc("UpdateWindow")
	procGetMessage         = user32.NewProc("GetMessageW")
	procDispatchMessage    = user32.NewProc("DispatchMessageW")
	procTranslateMessage   = user32.NewProc("TranslateMessage")
	procPostQuitMessage    = user32.NewProc("PostQuitMessage")
	procBeginPaint         = user32.NewProc("BeginPaint")
	procEndPaint           = user32.NewProc("EndPaint")
	procDrawText           = user32.NewProc("DrawTextW")
	procLoadCursor         = user32.NewProc("LoadCursorW")
	procInvalidateRect     = user32.NewProc("InvalidateRect")
	procPostMessage        = user32.NewProc("PostMessageW")
	procPostThreadMessage  = user32.NewProc("PostThreadMessageW")
	procGetCurrentThreadId = kernel32.NewProc("GetCurrentThreadId")
)

const (
	WS_POPUP            = 0x80000000
	WS_VISIBLE          = 0x10000000
	WS_BORDER           = 0x00800000
	WS_EX_NOACTIVATE    = 0x08000000
	WS_EX_TOOLWINDOW    = 0x00000080
	WS_EX_CLIENTEDGE    = 0x00000200
	WM_DESTROY          = 0x0002
	WM_PAINT            = 0x000F
	WM_TIMER            = 0x0113
	WM_CLOSE            = 0x0010
	WM_LBUTTONDOWN      = 0x0201
	WM_RBUTTONDOWN      = 0x0204
	WM_NCLBUTTONDOWN    = 0x00A1
	WM_NCRBUTTONDOWN    = 0x00A4
	WM_USER             = 0x0400
	WM_UPDATE_TEXT      = WM_USER + 1
	WM_EXIT_LOOP        = WM_USER + 2
	SW_SHOW             = 5
	SWP_NOACTIVATE      = 0x0010
	SWP_NOMOVE          = 0x0002
	SWP_NOSIZE          = 0x0001
	HWND_TOPMOST        = ^uintptr(0)
	SM_CXSCREEN         = 0
	SM_CYSCREEN         = 1
	DT_CENTER           = 0x00000001
	DT_VCENTER          = 0x00000004
	DT_WORDBREAK        = 0x00000010
	COLOR_WINDOW        = 5
	IDC_ARROW           = 32512
	TIMER_CLOSE         = 1
	TIMER_COUNTDOWN     = 2
)

type WNDCLASSEX struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     syscall.Handle
	HIcon         syscall.Handle
	HCursor       syscall.Handle
	HbrBackground syscall.Handle
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       syscall.Handle
}

type MSG struct {
	Hwnd    syscall.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

type POINT struct {
	X, Y int32
}

type PAINTSTRUCT struct {
	Hdc         syscall.Handle
	FErase      int32
	RcPaint     RECT
	FRestore    int32
	FIncUpdate  int32
	RgbReserved [32]byte
}

type RECT struct {
	Left, Top, Right, Bottom int32
}

var (
	popupText string

	// Single popup thread management
	popupQueue chan string
	popupOnce  sync.Once
	popupMutex sync.Mutex
	windowClassRegistered bool

	// Current popup state
	currentPopupHwnd syscall.Handle
	currentPopupMutex sync.Mutex
	isCountdownMode bool
	countdownRemaining int
)
// ShowBlockingError displays a modal, blocking error dialog and returns after user dismisses it.
func ShowBlockingError(title, message string) {
	titlePtr, _ := syscall.UTF16PtrFromString(title)
	msgPtr, _ := syscall.UTF16PtrFromString(message)
	const MB_OK = 0x00000000
	const MB_ICONERROR = 0x00000010
	const MB_SYSTEMMODAL = 0x00001000
	procMessageBox.Call(0, uintptr(unsafe.Pointer(msgPtr)), uintptr(unsafe.Pointer(titlePtr)), MB_OK|MB_ICONERROR|MB_SYSTEMMODAL)
}


// initPopupThread initializes the single popup thread
func initPopupThread() {
	popupOnce.Do(func() {
		popupQueue = make(chan string, 10)
		log.Printf("Popup: Starting single popup thread")

		go func() {

			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Popup thread panic: %v", r)
				}
			}()

			// Register window class once for the entire thread
			if err := registerPopupWindowClass(); err != nil {
				log.Printf("Popup: Failed to register window class: %v", err)
				return
			}

			log.Printf("Popup: Single thread ready, processing popup queue")

			// Process popup requests sequentially
			for text := range popupQueue {
				log.Printf("Popup: Processing popup request")
				if err := createAndShowPopup(text); err != nil {
					log.Printf("Popup: Failed to show popup: %v", err)
				}
			}
		}()
	})
}

// showWindowsPopup queues a popup to be shown by the single popup thread
func showWindowsPopup(text string) error {
	initPopupThread()

	select {
	case popupQueue <- text:
		log.Printf("Popup: Queued popup request")
		return nil
	default:
		log.Printf("Popup: Queue full, dropping popup request")
		return nil // Don't block or error - just drop it
	}
}



func loadCursor() syscall.Handle {
	cursor, _, _ := procLoadCursor.Call(0, IDC_ARROW)
	return syscall.Handle(cursor)
}

func wndProc(hwnd syscall.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_PAINT:
		var ps PAINTSTRUCT
		hdc, _, _ := procBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))

		// Draw text (left-aligned, top-aligned, with word wrap)
		rect := RECT{Left: 10, Top: 10, Right: 390, Bottom: 90}
		textPtr, _ := syscall.UTF16PtrFromString(popupText)
		procDrawText.Call(
			hdc,
			uintptr(unsafe.Pointer(textPtr)),
			uintptr(^uint32(0)), // -1 as uintptr
			uintptr(unsafe.Pointer(&rect)),
			DT_WORDBREAK, // Left-aligned, top-aligned, word wrap only
		)

		procEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
		return 0

	case WM_TIMER:
		timerID := wParam
		if timerID == TIMER_COUNTDOWN {
			// Countdown timer - update text and check if done
			currentPopupMutex.Lock()
			if isCountdownMode && countdownRemaining > 0 {
				countdownRemaining--
				if countdownRemaining > 0 {
					popupText = fmt.Sprintf("OCR in progress...\n%d seconds remaining", countdownRemaining)
					currentPopupMutex.Unlock()
					// Force repaint
					procInvalidateRect.Call(uintptr(hwnd), 0, 1)
				} else {
					// Timeout reached - close popup
					log.Printf("Popup: Countdown reached zero, closing")
					isCountdownMode = false
					currentPopupMutex.Unlock()
					procKillTimer.Call(uintptr(hwnd), TIMER_COUNTDOWN)
					procDestroyWindow.Call(uintptr(hwnd))
				}
			} else {
				currentPopupMutex.Unlock()
			}
			return 0
		} else if timerID == TIMER_CLOSE {
			// Close timer expired
			log.Printf("Popup: Close timer expired, closing window")
			procKillTimer.Call(uintptr(hwnd), TIMER_CLOSE)
			procKillTimer.Call(uintptr(hwnd), TIMER_COUNTDOWN)
			procDestroyWindow.Call(uintptr(hwnd))
			return 0
		}

	case WM_UPDATE_TEXT:
		// Custom message to update text
		currentPopupMutex.Lock()
		// Stop countdown mode and switch to result display
		if isCountdownMode {
			isCountdownMode = false
			procKillTimer.Call(uintptr(hwnd), TIMER_COUNTDOWN)
			// Set 3-second close timer
			procSetTimer.Call(uintptr(hwnd), TIMER_CLOSE, 3000, 0)
			log.Printf("Popup: Switched to result mode, showing for 3 seconds")
		}
		currentPopupMutex.Unlock()
		// Force repaint with new text
		procInvalidateRect.Call(uintptr(hwnd), 0, 1)
		return 0

	case WM_LBUTTONDOWN, WM_RBUTTONDOWN, WM_NCLBUTTONDOWN, WM_NCRBUTTONDOWN:
		// Close immediately on any click
		log.Printf("Popup: Click detected, closing window")
		procKillTimer.Call(uintptr(hwnd), TIMER_CLOSE)
		procKillTimer.Call(uintptr(hwnd), TIMER_COUNTDOWN)
		procDestroyWindow.Call(uintptr(hwnd))
		return 0

	case WM_DESTROY:
		log.Printf("Popup: WM_DESTROY received for hwnd=%d", hwnd)
		currentPopupMutex.Lock()
		currentPopupHwnd = 0
		isCountdownMode = false
		currentPopupMutex.Unlock()
		// Post custom exit message to thread (not window) to exit message loop
		threadID, _, _ := procGetCurrentThreadId.Call()
		log.Printf("Popup: Posting WM_EXIT_LOOP to thread %d", threadID)
		ret, _, err := procPostThreadMessage.Call(threadID, WM_EXIT_LOOP, 0, 0)
		log.Printf("Popup: PostThreadMessage result=%d, err=%v", ret, err)
		return 0

	case WM_CLOSE:
		procKillTimer.Call(uintptr(hwnd), TIMER_CLOSE)
		procKillTimer.Call(uintptr(hwnd), TIMER_COUNTDOWN)
		procDestroyWindow.Call(uintptr(hwnd))
		return 0
	}

	ret, _, _ := procDefWindowProc.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return ret
}

// registerPopupWindowClass registers the window class once
func registerPopupWindowClass() error {
	popupMutex.Lock()
	defer popupMutex.Unlock()

	if windowClassRegistered {
		return nil
	}

	className, _ := syscall.UTF16PtrFromString("OCRNotificationClass")

	// Register window class
	wc := WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEX{})),
		LpfnWndProc:   syscall.NewCallback(wndProc),
		HInstance:     0,
		HCursor:       loadCursor(),
		HbrBackground: syscall.Handle(COLOR_WINDOW + 1),
		LpszClassName: className,
	}

	atom, _, _ := procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 {
		return syscall.GetLastError()
	}

	windowClassRegistered = true
	log.Printf("Popup: Window class registered successfully")
	return nil
}

// createAndShowPopup creates and shows a single popup window
func createAndShowPopup(text string) error {
	log.Printf("Popup: Creating popup window")
	popupText = text

	className, _ := syscall.UTF16PtrFromString("OCRNotificationClass")
	windowName, _ := syscall.UTF16PtrFromString("OCR Result")

	// Get screen dimensions
	screenHeight, _, _ := procGetSystemMetrics.Call(SM_CYSCREEN)

	// Position in lower-left corner (400x100 pixels as requested)
	x := int32(20)
	y := int32(screenHeight) - 120 // 100px height + 20px margin
	width := int32(400)
	height := int32(100)

	log.Printf("Popup: Creating window at position (%d, %d) with size %dx%d", x, y, width, height)

	// Create window (no-activate toolwindow so clicks won't steal focus; we'll close on click)
	hwnd, _, _ := procCreateWindowEx.Call(
		WS_EX_NOACTIVATE|WS_EX_TOOLWINDOW|WS_EX_CLIENTEDGE,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowName)),
		WS_POPUP|WS_VISIBLE,
		uintptr(x),
		uintptr(y),
		uintptr(width),
		uintptr(height),
		0,
		0,
		0,
		0,
	)

	log.Printf("Popup: CreateWindowEx returned hwnd: %d", hwnd)

	if hwnd == 0 {
		log.Printf("Popup: Failed to create popup window")
		return nil // Don't return error to avoid breaking OCR
	}
	log.Printf("Popup: Window created successfully, hwnd: %d", hwnd)

	// Set window to be topmost but not steal focus
	procSetWindowPos.Call(
		hwnd,
		HWND_TOPMOST,
		0, 0, 0, 0,
		SWP_NOACTIVATE|SWP_NOMOVE|SWP_NOSIZE,
	)

	// Show window
	procShowWindow.Call(hwnd, SW_SHOW)
	procUpdateWindow.Call(hwnd)

	// Store hwnd for updates
	currentPopupMutex.Lock()
	currentPopupHwnd = syscall.Handle(hwnd)
	inCountdownMode := isCountdownMode
	currentPopupMutex.Unlock()

	// Set appropriate timer based on mode
	if inCountdownMode {
		// Countdown mode - start 1-second timer immediately to ensure reliable ticking
		timerResult, _, _ := procSetTimer.Call(hwnd, TIMER_COUNTDOWN, 1000, 0)
		log.Printf("Popup: Countdown mode, 1s timer started, result: %d", timerResult)
	} else {
		// Normal mode - set 3-second close timer
		timerResult, _, _ := procSetTimer.Call(hwnd, TIMER_CLOSE, 3000, 0)
		log.Printf("Popup: Set 3-second close timer, result: %d", timerResult)
	}

	// Message loop: run until WM_QUIT or WM_EXIT_LOOP
	var msg MSG
	for {
		ret, _, _ := procGetMessage.Call(
			uintptr(unsafe.Pointer(&msg)),
			0,
			0,
			0,
		)
		if ret == 0 { // WM_QUIT
			log.Printf("Popup: Message loop received WM_QUIT, exiting")
			break
		}
		if msg.Message == WM_EXIT_LOOP {
			log.Printf("Popup: Message loop received WM_EXIT_LOOP, exiting")
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
	}

	log.Printf("Popup: Message loop exited, flushing remaining messages")

	// Flush any remaining messages from the queue to prevent them from affecting next popup
	procPeekMessage := user32.NewProc("PeekMessageW")
	var flushMsg MSG
	for {
		ret, _, _ := procPeekMessage.Call(
			uintptr(unsafe.Pointer(&flushMsg)),
			0,
			0,
			0,
			1, // PM_REMOVE
		)
		if ret == 0 {
			break // No more messages
		}
		log.Printf("Popup: Flushed message 0x%x from queue", flushMsg.Message)
	}

	log.Printf("Popup: Message queue flushed")
	return nil
}

// StartCountdownPopup creates and shows a countdown popup
func StartCountdownPopup(timeoutSeconds int) error {
	initPopupThread()

	currentPopupMutex.Lock()
	// Close any existing popup
	if currentPopupHwnd != 0 {
		log.Printf("Popup: Closing existing popup (hwnd=%d) before starting countdown", currentPopupHwnd)
		procDestroyWindow.Call(uintptr(currentPopupHwnd))
		currentPopupHwnd = 0
	}
	isCountdownMode = true
	countdownRemaining = timeoutSeconds
	initialText := fmt.Sprintf("OCR in progress...\n%d seconds remaining", timeoutSeconds)
	currentPopupMutex.Unlock()

	log.Printf("Popup: Starting countdown popup with %d seconds", timeoutSeconds)

	// Queue the popup creation
	select {
	case popupQueue <- initialText:
		// Start countdown timer after popup is created
		go func() {
			time.Sleep(100 * time.Millisecond) // Wait for popup to be created
			currentPopupMutex.Lock()
			hwnd := currentPopupHwnd
			currentPopupMutex.Unlock()
			if hwnd != 0 {
				// Set 1-second countdown timer
				procSetTimer.Call(uintptr(hwnd), TIMER_COUNTDOWN, 1000, 0)
				log.Printf("Popup: Countdown timer started")
			}
		}()
		return nil
	default:
		log.Printf("Popup: Queue full, dropping countdown popup request")
		return nil
	}
}

// UpdatePopupText updates the text of the current popup
func UpdatePopupText(text string) error {
	currentPopupMutex.Lock()
	hwnd := currentPopupHwnd
	popupText = text
	currentPopupMutex.Unlock()

	if hwnd == 0 {
		log.Printf("Popup: No active popup to update")
		return nil
	}

	log.Printf("Popup: Updating popup text to %d characters", len(text))
	// Send custom message to update text
	procPostMessage.Call(uintptr(hwnd), WM_UPDATE_TEXT, 0, 0)
	return nil
}

// ClosePopup closes the current popup if any
func ClosePopup() error {
	currentPopupMutex.Lock()
	hwnd := currentPopupHwnd
	currentPopupMutex.Unlock()

	if hwnd == 0 {
		return nil
	}

	log.Printf("Popup: Closing popup")
	procDestroyWindow.Call(uintptr(hwnd))
	return nil
}
