//go:build windows

package gui

import (
	_ "embed"
	"fmt"
	"image"
	"image/draw"
	"log"
	"os"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"screen-ocr-llm/src/screenshot"

	"github.com/lxn/win"
)

//go:embed lasso.cur
var embeddedLassoCursorData []byte

// Global state for the simple overlay
var (
	simpleOverlayHwnd          win.HWND
	simpleIsSelecting          bool
	simpleSelectionMode        selectionMode
	simpleLastModeToggle       time.Time
	simpleSpaceWasDown         bool
	simpleEscapeWasDown        bool
	simpleStartX, simpleStartY int32
	simpleEndX, simpleEndY     int32
	simpleLassoPoints          []screenshot.Point
	simpleScreenWidth          int32
	simpleScreenHeight         int32
	simpleVirtualScreenX       int32
	simpleVirtualScreenY       int32
	simpleCrossCursor          win.HCURSOR
	simpleHandCursor           win.HCURSOR
	simpleLassoCursorInit      bool
	simpleSelectionResult      chan screenshot.Region
)

type selectionMode int

const (
	modeRect selectionMode = iota
	modeLasso
)

const (
	minSelectionSpan         = 5
	lassoMinPoints           = 8
	lassoCloseDistance       = 14
	lassoMinPointSeparation2 = 4
	lassoMinArea             = 100
	overlayKeyPollTimerID    = 1
	overlayKeyPollIntervalMs = 25
	overlayToggleDebounce    = 300 * time.Millisecond
)

var (
	user32DLL                    = syscall.NewLazyDLL("user32.dll")
	procAllowSetForegroundWindow = user32DLL.NewProc("AllowSetForegroundWindow")
	procGetAsyncKeyState         = user32DLL.NewProc("GetAsyncKeyState")
)

// Global variables for screen capture
var (
	screenImage   *image.RGBA
	screenHDC     win.HDC
	screenHBitmap win.HBITMAP
)

// StartInteractiveRegionSelection creates a working overlay with screen background
func StartInteractiveRegionSelection() (screenshot.Region, error) {
	return StartInteractiveRegionSelectionWithMode("rectangle")
}

// StartInteractiveRegionSelectionWithMode creates a working overlay with a configured initial mode.
func StartInteractiveRegionSelectionWithMode(defaultMode string) (screenshot.Region, error) {
	log.Printf("Starting WORKING Windows region selection...")

	// Get screen dimensions
	simpleScreenWidth = win.GetSystemMetrics(win.SM_CXSCREEN)
	simpleScreenHeight = win.GetSystemMetrics(win.SM_CYSCREEN)
	// Use VIRTUAL SCREEN metrics to cover all monitors
	vx := win.GetSystemMetrics(win.SM_XVIRTUALSCREEN)
	vy := win.GetSystemMetrics(win.SM_YVIRTUALSCREEN)
	vw := win.GetSystemMetrics(win.SM_CXVIRTUALSCREEN)
	vh := win.GetSystemMetrics(win.SM_CYVIRTUALSCREEN)
	log.Printf("Virtual screen: x=%d y=%d w=%d h=%d", vx, vy, vw, vh)

	// Store virtual screen offset for coordinate calculation
	simpleVirtualScreenX = vx
	simpleVirtualScreenY = vy

	log.Printf("Screen dimensions: %dx%d", simpleScreenWidth, simpleScreenHeight)

	// Capture the screen first (use full virtual screen size)
	var err error
	screenImage, err = captureScreen(int(vw), int(vh))
	if err != nil {
		return screenshot.Region{}, fmt.Errorf("failed to capture screen: %v", err)
	}
	log.Printf("Screen captured successfully")

	// Load cross cursor
	simpleCrossCursor = win.LoadCursor(0, win.MAKEINTRESOURCE(win.IDC_CROSS))
	if simpleCrossCursor == 0 {
		log.Printf("OVERLAY: Failed to load cross cursor")
	}
	if !simpleLassoCursorInit {
		simpleHandCursor = loadEmbeddedLassoCursor()
		if simpleHandCursor == 0 {
			simpleHandCursor = win.LoadCursor(0, win.MAKEINTRESOURCE(win.IDC_HAND))
			if simpleHandCursor == 0 {
				log.Printf("OVERLAY: Failed to load lasso cursor and hand fallback")
			}
		}
		simpleLassoCursorInit = true
	}

	// Initialize selection state
	simpleSelectionResult = make(chan screenshot.Region, 1)
	simpleIsSelecting = false
	simpleSelectionMode = parseSelectionMode(defaultMode)
	simpleLastModeToggle = time.Time{}
	simpleSpaceWasDown = false
	simpleEscapeWasDown = false
	simpleLassoPoints = nil
	log.Printf("OVERLAY: Initial selection mode: %s", selectionModeString(simpleSelectionMode))

	// Register window class with unique name to avoid conflicts
	classNameStr := fmt.Sprintf("WorkingOverlay_%d", time.Now().UnixNano())
	className := syscall.StringToUTF16Ptr(classNameStr)
	wndClass := win.WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(win.WNDCLASSEX{})),
		Style:         win.CS_HREDRAW | win.CS_VREDRAW,
		LpfnWndProc:   syscall.NewCallback(workingWndProc),
		HInstance:     win.GetModuleHandle(nil),
		HCursor:       simpleCrossCursor,
		HbrBackground: 0, // No background brush - we'll paint ourselves
		LpszClassName: className,
	}

	atom := win.RegisterClassEx(&wndClass)
	if atom == 0 {
		log.Printf("OVERLAY: Failed to register window class")
		return screenshot.Region{}, fmt.Errorf("failed to register window class")
	}
	log.Printf("OVERLAY: Window class registered successfully, atom: %d", atom)
	defer win.UnregisterClass(className)

	// Create fullscreen window covering the entire virtual screen
	simpleOverlayHwnd = win.CreateWindowEx(
		win.WS_EX_TOPMOST,
		className,
		syscall.StringToUTF16Ptr("Select Region - Drag to select, SPACE toggles lasso, ESC cancels"),
		win.WS_POPUP|win.WS_VISIBLE,
		vx, vy, vw, vh,
		0, 0, win.GetModuleHandle(nil), nil,
	)

	if simpleOverlayHwnd == 0 {
		log.Printf("OVERLAY: Failed to create overlay window")
		return screenshot.Region{}, fmt.Errorf("failed to create overlay window")
	}

	log.Printf("OVERLAY: Window created successfully, hwnd: %v, position: (%d,%d) size: (%d,%d)", simpleOverlayHwnd, vx, vy, vw, vh)

	// Show window and bring to front
	log.Printf("OVERLAY: Calling ShowWindow")
	win.ShowWindow(simpleOverlayHwnd, win.SW_SHOW)
	log.Printf("OVERLAY: Calling AllowSetForegroundWindow")
	pid := os.Getpid()
	procAllowSetForegroundWindow.Call(uintptr(pid))
	log.Printf("OVERLAY: Calling SetForegroundWindow")
	ret := win.SetForegroundWindow(simpleOverlayHwnd)
	log.Printf("OVERLAY: SetForegroundWindow returned: %v", ret)
	log.Printf("OVERLAY: Calling BringWindowToTop")
	bringRet := win.BringWindowToTop(simpleOverlayHwnd)
	log.Printf("OVERLAY: BringWindowToTop returned: %v", bringRet)
	log.Printf("OVERLAY: Calling SetFocus")
	focusRet := win.SetFocus(simpleOverlayHwnd)
	log.Printf("OVERLAY: SetFocus returned: %v", focusRet)
	log.Printf("OVERLAY: Calling UpdateWindow")
	win.UpdateWindow(simpleOverlayHwnd)

	if timerID := win.SetTimer(simpleOverlayHwnd, overlayKeyPollTimerID, overlayKeyPollIntervalMs, 0); timerID == 0 {
		log.Printf("OVERLAY: Failed to start keyboard poll timer")
	}

	log.Printf("Window shown, starting message loop...")

	// Message loop
	var msg win.MSG
	for {
		ret := win.GetMessage(&msg, 0, 0, 0)
		if ret == 0 { // WM_QUIT
			log.Printf("WM_QUIT received")
			break
		}
		if ret == -1 { // Error
			log.Printf("GetMessage error")
			break
		}

		win.TranslateMessage(&msg)
		win.DispatchMessage(&msg)

		// Check if selection is done
		select {
		case region := <-simpleSelectionResult:
			win.DestroyWindow(simpleOverlayHwnd)
			log.Printf("Selection completed: %+v", region)
			return region, nil
		default:
		}
	}

	win.DestroyWindow(simpleOverlayHwnd)
	return screenshot.Region{}, fmt.Errorf("selection cancelled")
}

// captureScreen captures the entire screen as an RGBA image
func captureScreen(width, height int) (*image.RGBA, error) {
	log.Printf("OVERLAY: Starting screen capture for overlay background, expected size: %dx%d", width, height)
	// Use the project's screenshot package to capture the screen
	img, err := screenshot.Capture()
	if err != nil {
		log.Printf("OVERLAY: Screen capture failed: %v", err)
		return nil, err
	}

	actualW := img.Bounds().Dx()
	actualH := img.Bounds().Dy()
	log.Printf("OVERLAY: Screen captured successfully, actual size: %dx%d", actualW, actualH)

	// The image is already RGBA, but let's ensure it matches our expected size
	if actualW != width || actualH != height {
		log.Printf("OVERLAY: Size mismatch, resizing from %dx%d to %dx%d", actualW, actualH, width, height)
		// Resize if needed
		rgba := image.NewRGBA(image.Rect(0, 0, width, height))
		draw.Draw(rgba, rgba.Bounds(), img, image.Point{}, draw.Src)
		return rgba, nil
	}

	return img, nil
}

// workingWndProc handles window messages for the working overlay
func workingWndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	// Log all messages for debugging
	if msg != win.WM_PAINT && msg != win.WM_NCHITTEST && msg != win.WM_SETCURSOR && msg != win.WM_TIMER {
		log.Printf("Window message: 0x%x (wParam=%d, lParam=%d)", msg, wParam, lParam)
	}

	// Special logging for mouse events
	if msg == win.WM_LBUTTONDOWN || msg == win.WM_LBUTTONUP || msg == win.WM_RBUTTONDOWN {
		log.Printf("MOUSE EVENT: 0x%x at (%d, %d)", msg, win.LOWORD(uint32(lParam)), win.HIWORD(uint32(lParam)))
	}

	switch msg {
	case win.WM_LBUTTONDOWN:
		x := int32(win.LOWORD(uint32(lParam)))
		y := int32(win.HIWORD(uint32(lParam)))
		log.Printf("Mouse down at (%d, %d), mode=%s", x, y, selectionModeString(simpleSelectionMode))

		win.SetCapture(hwnd)
		simpleIsSelecting = true
		if simpleSelectionMode == modeLasso {
			simpleLassoPoints = []screenshot.Point{{X: int(x), Y: int(y)}}
			simpleStartX = x
			simpleStartY = y
			simpleEndX = x
			simpleEndY = y
		} else {
			simpleStartX = x
			simpleStartY = y
			simpleEndX = x
			simpleEndY = y
		}

		// Force immediate repaint
		win.InvalidateRect(hwnd, nil, false)
		win.UpdateWindow(hwnd)
		return 0

	case win.WM_MOUSEMOVE:
		if simpleIsSelecting {
			x := int32(win.LOWORD(uint32(lParam)))
			y := int32(win.HIWORD(uint32(lParam)))
			if simpleSelectionMode == modeLasso {
				simpleEndX = x
				simpleEndY = y
				newPoint := screenshot.Point{X: int(x), Y: int(y)}
				if len(simpleLassoPoints) == 0 {
					simpleLassoPoints = append(simpleLassoPoints, newPoint)
				} else {
					lastPoint := simpleLassoPoints[len(simpleLassoPoints)-1]
					if pointDistanceSquared(lastPoint, newPoint) >= lassoMinPointSeparation2 {
						simpleLassoPoints = append(simpleLassoPoints, newPoint)
					}
				}
			} else {
				simpleEndX = x
				simpleEndY = y
			}

			// Force immediate repaint to show selection
			win.InvalidateRect(hwnd, nil, false)
			win.UpdateWindow(hwnd)
		}
		return 0

	case win.WM_LBUTTONUP:
		if simpleIsSelecting {
			win.ReleaseCapture()
			x := int32(win.LOWORD(uint32(lParam)))
			y := int32(win.HIWORD(uint32(lParam)))
			simpleEndX = x
			simpleEndY = y

			if simpleSelectionMode == modeLasso {
				simpleIsSelecting = false
				upPoint := screenshot.Point{X: int(x), Y: int(y)}
				if len(simpleLassoPoints) == 0 {
					simpleLassoPoints = append(simpleLassoPoints, upPoint)
				} else {
					lastPoint := simpleLassoPoints[len(simpleLassoPoints)-1]
					if pointDistanceSquared(lastPoint, upPoint) >= lassoMinPointSeparation2 {
						simpleLassoPoints = append(simpleLassoPoints, upPoint)
					}
				}

				if !lassoHasValidClosure(simpleLassoPoints) {
					log.Printf("Lasso not closed on mouse-up; retry by dragging and ending near start")
					simpleLassoPoints = nil
					win.InvalidateRect(hwnd, nil, false)
					win.UpdateWindow(hwnd)
					return 0
				}

				left, top, right, bottom := polygonBounds(simpleLassoPoints)
				width := right - left
				height := bottom - top
				area := polygonArea(simpleLassoPoints)
				if width <= minSelectionSpan || height <= minSelectionSpan || area < lassoMinArea {
					log.Printf("Lasso selection too small: width=%d height=%d area=%d", width, height, area)
					simpleLassoPoints = nil
					win.InvalidateRect(hwnd, nil, false)
					win.UpdateWindow(hwnd)
					return 0
				}

				polygon := make([]screenshot.Point, len(simpleLassoPoints))
				for i, p := range simpleLassoPoints {
					polygon[i] = screenshot.Point{
						X: p.X + int(simpleVirtualScreenX),
						Y: p.Y + int(simpleVirtualScreenY),
					}
				}

				region := screenshot.Region{
					X:       int(left) + int(simpleVirtualScreenX),
					Y:       int(top) + int(simpleVirtualScreenY),
					Width:   int(width),
					Height:  int(height),
					Polygon: polygon,
				}
				log.Printf("Final lasso region with virtual screen offset: X=%d Y=%d W=%d H=%d points=%d", region.X, region.Y, region.Width, region.Height, len(region.Polygon))
				simpleSelectionResult <- region
				return 0
			}

			simpleIsSelecting = false

			// Calculate region
			left := simpleMin(simpleStartX, simpleEndX)
			top := simpleMin(simpleStartY, simpleEndY)
			width := simpleAbs(simpleEndX - simpleStartX)
			height := simpleAbs(simpleEndY - simpleStartY)

			log.Printf("Mouse up at (%d, %d), selection: %d,%d,%d,%d", x, y, left, top, width, height)

			if width > minSelectionSpan && height > minSelectionSpan {
				region := screenshot.Region{
					X:      int(left) + int(simpleVirtualScreenX),
					Y:      int(top) + int(simpleVirtualScreenY),
					Width:  int(width),
					Height: int(height),
				}
				log.Printf("Final region with virtual screen offset: X=%d Y=%d W=%d H=%d", region.X, region.Y, region.Width, region.Height)
				simpleSelectionResult <- region
			} else {
				log.Printf("Selection too small, ignoring")
			}
		}
		return 0

	case win.WM_PAINT:
		var ps win.PAINTSTRUCT
		hdc := win.BeginPaint(hwnd, &ps)

		log.Printf("WM_PAINT called, isSelecting=%v", simpleIsSelecting)

		// Draw the captured screen as background
		if screenImage != nil {
			drawScreenBackground(hdc)
		}

		drawSelectionHints(hdc)

		if simpleSelectionMode == modeLasso {
			if simpleIsSelecting && len(simpleLassoPoints) > 1 {
				drawLassoPolyline(hdc, simpleLassoPoints)
			}
		} else if simpleIsSelecting {
			log.Printf("Drawing selection rectangle: (%d,%d) to (%d,%d)", simpleStartX, simpleStartY, simpleEndX, simpleEndY)
			drawSelectionRectangle(hdc, simpleStartX, simpleStartY, simpleEndX, simpleEndY)
		}

		win.EndPaint(hwnd, &ps)
		return 0

	case win.WM_SETCURSOR:
		setModeCursor()
		return 1 // Indicate we handled it

	case win.WM_ACTIVATE:
		log.Printf("WM_ACTIVATE received, wParam: %d", wParam)
		return 0

	case win.WM_TIMER:
		if wParam == overlayKeyPollTimerID {
			handlePolledKeys(hwnd)
			return 0
		}
		return 0

	case win.WM_KEYDOWN:
		switch wParam {
		case win.VK_ESCAPE:
			simpleEscapeWasDown = true
			cancelSelection()
		case win.VK_SPACE:
			simpleSpaceWasDown = true
			toggleSelectionMode(hwnd)
		}
		return 0

	case win.WM_KEYUP, win.WM_SYSKEYUP:
		switch wParam {
		case win.VK_SPACE:
			simpleSpaceWasDown = false
		case win.VK_ESCAPE:
			simpleEscapeWasDown = false
		}
		return 0

	case win.WM_NCHITTEST:
		// Force all points to be client area so the window receives mouse events
		return uintptr(win.HTCLIENT)

	case win.WM_DESTROY:
		log.Printf("WM_DESTROY received")
		win.KillTimer(hwnd, overlayKeyPollTimerID)
		// Do NOT PostQuitMessage here. In the success path we return from
		// StartInteractiveRegionSelection() as soon as we have the region,
		// and posting WM_QUIT here would leave a leftover WM_QUIT in the
		// thread queue that the next invocation would consume immediately,
		// causing an instant "selection cancelled" on the second hotkey.
		return 0
	}

	return win.DefWindowProc(hwnd, msg, wParam, lParam)
}

func selectionModeString(mode selectionMode) string {
	if mode == modeLasso {
		return "lasso"
	}
	return "rect"
}

func parseSelectionMode(value string) selectionMode {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "lasso":
		return modeLasso
	default:
		return modeRect
	}
}

func loadEmbeddedLassoCursor() win.HCURSOR {
	if len(embeddedLassoCursorData) == 0 {
		log.Printf("OVERLAY: Embedded lasso cursor data is empty")
		return 0
	}

	tempFile, err := os.CreateTemp("", "screen-ocr-lasso-*.cur")
	if err != nil {
		log.Printf("OVERLAY: Failed to create temp cursor file: %v", err)
		return 0
	}
	tempPath := tempFile.Name()

	if _, err := tempFile.Write(embeddedLassoCursorData); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
		log.Printf("OVERLAY: Failed to write temp cursor file: %v", err)
		return 0
	}
	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempPath)
		log.Printf("OVERLAY: Failed to close temp cursor file: %v", err)
		return 0
	}

	handle := win.LoadImage(
		0,
		syscall.StringToUTF16Ptr(tempPath),
		win.IMAGE_CURSOR,
		0,
		0,
		win.LR_LOADFROMFILE|win.LR_DEFAULTSIZE,
	)
	_ = os.Remove(tempPath)
	if handle == 0 {
		log.Printf("OVERLAY: Failed to load embedded lasso cursor")
		return 0
	}

	log.Printf("OVERLAY: Loaded embedded lasso cursor")
	return win.HCURSOR(handle)
}

func getAsyncKeyState(vk int32) (bool, bool) {
	state, _, _ := procGetAsyncKeyState.Call(uintptr(vk))
	s := uint16(state)
	isDown := s&0x8000 != 0
	wasPressedSinceLastPoll := s&0x0001 != 0
	return isDown, wasPressedSinceLastPoll
}

func handlePolledKeys(hwnd win.HWND) {
	spaceDown, spacePressed := getAsyncKeyState(win.VK_SPACE)
	if !simpleSpaceWasDown && (spaceDown || spacePressed) {
		log.Printf("Space detected via async polling")
		toggleSelectionMode(hwnd)
	}
	simpleSpaceWasDown = spaceDown

	escapeDown, escapePressed := getAsyncKeyState(win.VK_ESCAPE)
	if !simpleEscapeWasDown && (escapeDown || escapePressed) {
		log.Printf("Escape detected via async polling")
		cancelSelection()
	}
	simpleEscapeWasDown = escapeDown
}

func toggleSelectionMode(hwnd win.HWND) {
	now := time.Now()
	if !simpleLastModeToggle.IsZero() && now.Sub(simpleLastModeToggle) < overlayToggleDebounce {
		log.Printf("Ignoring mode toggle inside debounce window")
		return
	}
	simpleLastModeToggle = now

	if simpleIsSelecting {
		win.ReleaseCapture()
		simpleIsSelecting = false
	}
	simpleLassoPoints = nil
	if simpleSelectionMode == modeRect {
		simpleSelectionMode = modeLasso
	} else {
		simpleSelectionMode = modeRect
	}
	log.Printf("Selection mode changed to %s", selectionModeString(simpleSelectionMode))
	setModeCursor()
	win.InvalidateRect(hwnd, nil, false)
	win.UpdateWindow(hwnd)
}

func setModeCursor() {
	if simpleSelectionMode == modeLasso {
		if simpleHandCursor != 0 {
			win.SetCursor(simpleHandCursor)
			return
		}
	}
	if simpleCrossCursor != 0 {
		win.SetCursor(simpleCrossCursor)
	}
}

func cancelSelection() {
	log.Printf("Escape pressed, cancelling selection")
	win.PostQuitMessage(0)
}

func pointDistanceSquared(a, b screenshot.Point) int {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return dx*dx + dy*dy
}

func lassoHasValidClosure(points []screenshot.Point) bool {
	if len(points) < lassoMinPoints {
		return false
	}
	start := points[0]
	end := points[len(points)-1]
	return pointDistanceSquared(start, end) <= lassoCloseDistance*lassoCloseDistance
}

func polygonBounds(points []screenshot.Point) (int32, int32, int32, int32) {
	if len(points) == 0 {
		return 0, 0, 0, 0
	}

	minX, maxX := points[0].X, points[0].X
	minY, maxY := points[0].Y, points[0].Y
	for _, p := range points[1:] {
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}

	return int32(minX), int32(minY), int32(maxX), int32(maxY)
}

func polygonArea(points []screenshot.Point) int {
	if len(points) < 3 {
		return 0
	}

	var area2 int64
	for i := 0; i < len(points); i++ {
		j := (i + 1) % len(points)
		area2 += int64(points[i].X*points[j].Y - points[j].X*points[i].Y)
	}
	if area2 < 0 {
		area2 = -area2
	}
	return int(area2 / 2)
}

func drawSelectionRectangle(hdc win.HDC, startX, startY, endX, endY int32) {
	gdi32 := syscall.NewLazyDLL("gdi32.dll")
	createPen := gdi32.NewProc("CreatePen")
	rectangle := gdi32.NewProc("Rectangle")

	redPen, _, _ := createPen.Call(0, 3, 0x0000FF)
	oldPen := win.SelectObject(hdc, win.HGDIOBJ(redPen))
	oldBrush := win.SelectObject(hdc, win.GetStockObject(win.NULL_BRUSH))

	left := simpleMin(startX, endX)
	top := simpleMin(startY, endY)
	right := simpleMax(startX, endX)
	bottom := simpleMax(startY, endY)
	rectangle.Call(uintptr(hdc), uintptr(left), uintptr(top), uintptr(right), uintptr(bottom))

	win.SelectObject(hdc, oldPen)
	win.SelectObject(hdc, oldBrush)
	win.DeleteObject(win.HGDIOBJ(redPen))
}

func drawLassoPolyline(hdc win.HDC, points []screenshot.Point) {
	if len(points) < 2 {
		return
	}

	gdi32 := syscall.NewLazyDLL("gdi32.dll")
	createPen := gdi32.NewProc("CreatePen")
	polyline := gdi32.NewProc("Polyline")
	ellipse := gdi32.NewProc("Ellipse")

	redPen, _, _ := createPen.Call(0, 3, 0x0000FF)
	oldPen := win.SelectObject(hdc, win.HGDIOBJ(redPen))
	oldBrush := win.SelectObject(hdc, win.GetStockObject(win.NULL_BRUSH))

	winPoints := make([]win.POINT, len(points))
	for i, p := range points {
		winPoints[i] = win.POINT{X: int32(p.X), Y: int32(p.Y)}
	}
	polyline.Call(uintptr(hdc), uintptr(unsafe.Pointer(&winPoints[0])), uintptr(len(winPoints)))

	start := points[0]
	anchorRadius := int32(6)
	ellipse.Call(
		uintptr(hdc),
		uintptr(int32(start.X)-anchorRadius),
		uintptr(int32(start.Y)-anchorRadius),
		uintptr(int32(start.X)+anchorRadius),
		uintptr(int32(start.Y)+anchorRadius),
	)

	win.SelectObject(hdc, oldPen)
	win.SelectObject(hdc, oldBrush)
	win.DeleteObject(win.HGDIOBJ(redPen))
}

func drawSelectionHints(hdc win.HDC) {
	line1 := "ESC cancel   SPACE toggle lasso"
	line2 := "Rect mode: click and drag"
	if simpleSelectionMode == modeLasso {
		line2 = "Lasso mode: drag and release near start to close"
	}

	win.SetBkMode(hdc, win.TRANSPARENT)
	win.SetTextColor(hdc, win.COLORREF(0x00FFFF))
	win.TextOut(hdc, 16, 16, syscall.StringToUTF16Ptr(line1), int32(len(line1)))
	win.TextOut(hdc, 16, 38, syscall.StringToUTF16Ptr(line2), int32(len(line2)))
}

// drawScreenBackground draws the captured screen as background
func drawScreenBackground(hdc win.HDC) {
	if screenImage == nil {
		return
	}

	// Create a compatible DC and bitmap for the screen image
	memDC := win.CreateCompatibleDC(hdc)
	defer win.DeleteDC(memDC)

	// Create bitmap from screen image
	bounds := screenImage.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Create DIB section
	bitmapInfo := win.BITMAPINFO{
		BmiHeader: win.BITMAPINFOHEADER{
			BiSize:        uint32(unsafe.Sizeof(win.BITMAPINFOHEADER{})),
			BiWidth:       int32(width),
			BiHeight:      -int32(height), // Negative for top-down
			BiPlanes:      1,
			BiBitCount:    32,
			BiCompression: win.BI_RGB,
		},
	}

	var pBits unsafe.Pointer
	hBitmap := win.CreateDIBSection(memDC, &bitmapInfo.BmiHeader, win.DIB_RGB_COLORS, &pBits, 0, 0)
	if hBitmap == 0 {
		return
	}
	defer win.DeleteObject(win.HGDIOBJ(hBitmap))

	// Select bitmap into memory DC
	oldBitmap := win.SelectObject(memDC, win.HGDIOBJ(hBitmap))
	defer win.SelectObject(memDC, oldBitmap)

	// Copy image data to bitmap (convert RGBA to BGRA) with bounds checking
	// Calculate the proper stride (DWORD-aligned row size)
	stride := (((int32(width)*32 + 31) &^ 31) / 8)

	for y := 0; y < height; y++ {
		// Calculate safe row pointer with bounds checking
		rowOffset := uintptr(y) * uintptr(stride)

		// Get safe pointer to current row
		rowPtr := (*[1 << 29]byte)(unsafe.Pointer(uintptr(pBits) + rowOffset))

		for x := 0; x < width; x++ {
			pixelOffset := x * 4
			// Ensure we don't exceed the row width (width * 4 bytes per pixel)
			if pixelOffset+3 >= width*4 {
				break // Safety check for row bounds
			}

			r, g, b, a := screenImage.At(x, y).RGBA()
			rowPtr[pixelOffset] = byte(b >> 8)   // B
			rowPtr[pixelOffset+1] = byte(g >> 8) // G
			rowPtr[pixelOffset+2] = byte(r >> 8) // R
			rowPtr[pixelOffset+3] = byte(a >> 8) // A
		}
	}

	// BitBlt the screen image to the window
	win.BitBlt(hdc, 0, 0, int32(width), int32(height), memDC, 0, 0, win.SRCCOPY)
}

// Helper functions
func simpleMin(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

func simpleMax(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

func simpleAbs(x int32) int32 {
	if x < 0 {
		return -x
	}
	return x
}
