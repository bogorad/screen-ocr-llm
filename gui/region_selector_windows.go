//go:build windows

package gui

import (
	"fmt"
	"image"
	"image/draw"
	"log"
	"syscall"
	"time"
	"unsafe"

	"screen-ocr-llm/screenshot"

	"github.com/lxn/win"
)

// Global state for the simple overlay
var (
	simpleOverlayHwnd win.HWND
	simpleIsSelecting bool
	simpleStartX, simpleStartY int32
	simpleEndX, simpleEndY int32
	simpleScreenWidth int32
	simpleScreenHeight int32
	simpleSelectionResult chan screenshot.Region
)

// Global variables for screen capture
var (
	screenImage *image.RGBA
	screenHDC   win.HDC
	screenHBitmap win.HBITMAP
)

// StartInteractiveRegionSelection creates a working overlay with screen background
func StartInteractiveRegionSelection() (screenshot.Region, error) {
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

	log.Printf("Screen dimensions: %dx%d", simpleScreenWidth, simpleScreenHeight)

	// Capture the screen first
	var err error
	screenImage, err = captureScreen(int(simpleScreenWidth), int(simpleScreenHeight))
	if err != nil {
		return screenshot.Region{}, fmt.Errorf("failed to capture screen: %v", err)
	}
	log.Printf("Screen captured successfully")

	// Initialize selection state
	simpleSelectionResult = make(chan screenshot.Region, 1)
	simpleIsSelecting = false

	// Register window class with unique name to avoid conflicts
	classNameStr := fmt.Sprintf("WorkingOverlay_%d", time.Now().UnixNano())
	className := syscall.StringToUTF16Ptr(classNameStr)
	wndClass := win.WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(win.WNDCLASSEX{})),
		Style:         win.CS_HREDRAW | win.CS_VREDRAW,
		LpfnWndProc:   syscall.NewCallback(workingWndProc),
		HInstance:     win.GetModuleHandle(nil),
		HCursor:       win.LoadCursor(0, win.MAKEINTRESOURCE(win.IDC_CROSS)),
		HbrBackground: 0, // No background brush - we'll paint ourselves
		LpszClassName: className,
	}

	atom := win.RegisterClassEx(&wndClass)
	if atom == 0 {
		return screenshot.Region{}, fmt.Errorf("failed to register window class")
	}
	defer win.UnregisterClass(className)

	// Create fullscreen window covering the entire virtual screen
	simpleOverlayHwnd = win.CreateWindowEx(
		win.WS_EX_TOPMOST,
		className,
		syscall.StringToUTF16Ptr("Select Region - Click and drag, ESC to cancel"),
		win.WS_POPUP|win.WS_VISIBLE,
		vx, vy, vw, vh,
		0, 0, win.GetModuleHandle(nil), nil,
	)

	if simpleOverlayHwnd == 0 {
		return screenshot.Region{}, fmt.Errorf("failed to create overlay window")
	}

	log.Printf("Working overlay window created: %v", simpleOverlayHwnd)

	// Show window and bring to front
	win.ShowWindow(simpleOverlayHwnd, win.SW_SHOW)
	win.SetForegroundWindow(simpleOverlayHwnd)
	win.SetFocus(simpleOverlayHwnd)
	win.UpdateWindow(simpleOverlayHwnd)

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
	// Use the project's screenshot package to capture the screen
	img, err := screenshot.Capture()
	if err != nil {
		return nil, err
	}

	// The image is already RGBA, but let's ensure it matches our expected size
	if img.Bounds().Dx() != width || img.Bounds().Dy() != height {
		// Resize if needed
		rgba := image.NewRGBA(image.Rect(0, 0, width, height))
		draw.Draw(rgba, rgba.Bounds(), img, image.Point{}, draw.Src)
		return rgba, nil
	}

	return img, nil
}

// workingWndProc handles window messages for the working overlay
func workingWndProc(hwnd win.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case win.WM_LBUTTONDOWN:
		x := int32(win.LOWORD(uint32(lParam)))
		y := int32(win.HIWORD(uint32(lParam)))
		log.Printf("Mouse down at (%d, %d)", x, y)

		win.SetCapture(hwnd)
		simpleIsSelecting = true
		simpleStartX = x
		simpleStartY = y
		simpleEndX = x
		simpleEndY = y

		// Force immediate repaint
		win.InvalidateRect(hwnd, nil, false)
		win.UpdateWindow(hwnd)
		return 0

	case win.WM_MOUSEMOVE:
		if simpleIsSelecting {
			x := int32(win.LOWORD(uint32(lParam)))
			y := int32(win.HIWORD(uint32(lParam)))
			simpleEndX = x
			simpleEndY = y

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
			simpleIsSelecting = false

			// Calculate region
			left := simpleMin(simpleStartX, simpleEndX)
			top := simpleMin(simpleStartY, simpleEndY)
			width := simpleAbs(simpleEndX - simpleStartX)
			height := simpleAbs(simpleEndY - simpleStartY)

			log.Printf("Mouse up at (%d, %d), selection: %d,%d,%d,%d", x, y, left, top, width, height)

			if width > 5 && height > 5 {
				region := screenshot.Region{
					X:      int(left),
					Y:      int(top),
					Width:  int(width),
					Height: int(height),
				}
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

		// Draw selection rectangle if selecting
		if simpleIsSelecting {
			log.Printf("Drawing selection rectangle: (%d,%d) to (%d,%d)", simpleStartX, simpleStartY, simpleEndX, simpleEndY)
			// Use direct GDI calls
			gdi32 := syscall.NewLazyDLL("gdi32.dll")
			createPen := gdi32.NewProc("CreatePen")
			rectangle := gdi32.NewProc("Rectangle")

			// Create red pen for selection rectangle
			redPen, _, _ := createPen.Call(0, 3, 0x0000FF) // PS_SOLID, width 3, red color (BGR)
			oldPen := win.SelectObject(hdc, win.HGDIOBJ(redPen))

			// Set transparent brush
			oldBrush := win.SelectObject(hdc, win.GetStockObject(win.NULL_BRUSH))

			// Draw rectangle
			left := simpleMin(simpleStartX, simpleEndX)
			top := simpleMin(simpleStartY, simpleEndY)
			right := simpleMax(simpleStartX, simpleEndX)
			bottom := simpleMax(simpleStartY, simpleEndY)

			rectangle.Call(uintptr(hdc), uintptr(left), uintptr(top), uintptr(right), uintptr(bottom))

			// Restore old objects
			win.SelectObject(hdc, oldPen)
			win.SelectObject(hdc, oldBrush)
			win.DeleteObject(win.HGDIOBJ(redPen))
		}

		win.EndPaint(hwnd, &ps)
		return 0

	case win.WM_KEYDOWN:
		if wParam == win.VK_ESCAPE {
			log.Printf("Escape pressed, cancelling selection")
			win.PostQuitMessage(0)
		}
		return 0

	case win.WM_NCHITTEST:
		// Force all points to be client area so the window receives mouse events
		return uintptr(win.HTCLIENT)

	case win.WM_DESTROY:
		log.Printf("WM_DESTROY received")
		win.PostQuitMessage(0)
		return 0
	}

	return win.DefWindowProc(hwnd, msg, wParam, lParam)
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

	// Copy image data to bitmap (convert RGBA to BGRA)
	bitmapData := (*[1 << 30]byte)(pBits)[:width*height*4:width*height*4]
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, a := screenImage.At(x, y).RGBA()
			offset := (y*width + x) * 4
			bitmapData[offset] = byte(b >> 8)   // B
			bitmapData[offset+1] = byte(g >> 8) // G
			bitmapData[offset+2] = byte(r >> 8) // R
			bitmapData[offset+3] = byte(a >> 8) // A
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
