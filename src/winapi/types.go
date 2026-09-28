package winapi

import (
	"golang.org/x/sys/windows"
)

// Win32 Base Types & Structs
type RECT struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

func (r RECT) Width() int32 {
	return r.Right - r.Left
}

func (r RECT) Height() int32 {
	return r.Bottom - r.Top
}

type POINT struct {
	X int32
	Y int32
}

type MSG struct {
	HWnd    windows.HWND
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

type WNDCLASSEXW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     windows.Handle
	HIcon         windows.Handle
	HCursor       windows.Handle
	HbrBackground windows.Handle
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       windows.Handle
}

type PAINTSTRUCT struct {
	Hdc         windows.Handle
	FErase      int32
	RcPaint     RECT
	FRestore    int32
	FIncUpdate  int32
	RgbReserved [32]byte
}

type TRACKMOUSEEVENT struct {
	CbSize      uint32
	DwFlags     uint32
	HwndTrack   windows.HWND
	DwHoverTime uint32
}

// Shell_NotifyIcon Types
type NOTIFYICONDATAW struct {
	CbSize           uint32
	HWnd             windows.HWND
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	HIcon            windows.Handle
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	TimeoutOrVersion uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         windows.GUID
	HBalloonIcon     windows.Handle
}

type NOTIFYICONIDENTIFIER struct {
	CbSize   uint32
	HWnd     windows.HWND
	UID      uint32
	GuidItem windows.GUID
}

// Monitor Info
type MONITORINFO struct {
	CbSize    uint32
	RcMonitor RECT
	RcWork    RECT
	DwFlags   uint32
}

// GDI & Drawing Types
type BITMAPINFOHEADER struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

type RGBQUAD struct {
	RgbBlue     byte
	RgbGreen    byte
	RgbRed      byte
	RgbReserved byte
}

type BITMAPINFO struct {
	BmiHeader BITMAPINFOHEADER
	BmiColors [1]RGBQUAD
}

type ICONINFO struct {
	FIcon    int32
	XHotspot uint32
	YHotspot uint32
	HbmMask  windows.Handle
	HbmColor windows.Handle
}

// DNS Interface Settings for SetInterfaceDnsSettings (iphlpapi.dll)
type DNS_INTERFACE_SETTINGS struct {
	Version              uint32
	Flags                uint64
	Domain               *uint16
	NameServer           *uint16
	SearchList           *uint16
	RegistrationEnabled  uint32
	RegisterAdapterName  uint32
	EnableLLMNR          uint32
	QueryAdapterName     uint32
	ProfileNameServer    *uint16
}

const (
	// Window Messages
	WM_NULL             = 0x0000
	WM_CREATE           = 0x0001
	WM_DESTROY          = 0x0002
	WM_ACTIVATE         = 0x0006
	WM_SETFOCUS         = 0x0007
	WM_KILLFOCUS        = 0x0008
	WM_PAINT            = 0x000F
	WM_CLOSE            = 0x0010
	WM_ERASEBKGND       = 0x0014
	WM_SETTINGCHANGE    = 0x001A
	WM_CONTEXTMENU      = 0x007B
	WM_COMMAND          = 0x0111
	WM_TIMER            = 0x0113
	WM_MOUSEMOVE        = 0x0200
	WM_LBUTTONDOWN      = 0x0201
	WM_LBUTTONUP        = 0x0202
	WM_RBUTTONDOWN      = 0x0204
	WM_RBUTTONUP        = 0x0205
	WM_MOUSELEAVE       = 0x02A3
	WM_DPICHANGED       = 0x02E0
	WM_USER             = 0x0400

	WA_INACTIVE = 0

	// TrackMouseEvent flags
	TME_LEAVE = 0x00000002

	// Window Styles
	WS_POPUP        = 0x80000000
	WS_CLIPSIBLINGS = 0x04000000
	WS_VISIBLE      = 0x10000000

	// Extended Window Styles
	WS_EX_TOPMOST    = 0x00000008
	WS_EX_TOOLWINDOW = 0x00000080
	WS_EX_LAYERED    = 0x00080000
	WS_EX_NOACTIVATE = 0x08000000

	// ShowWindow commands
	SW_HIDE            = 0
	SW_SHOWNORMAL      = 1
	SW_SHOW            = 5
	SW_SHOWNA          = 8
	SW_SHOWNOACTIVATE  = 4

	// SetWindowPos flags
	SWP_NOSIZE     = 0x0001
	SWP_NOMOVE     = 0x0002
	SWP_NOZORDER   = 0x0004
	SWP_SHOWWINDOW = 0x0040
	HWND_TOPMOST   = ^uintptr(0) // -1

	// Shell_NotifyIcon flags & messages
	NIM_ADD        = 0x00000000
	NIM_MODIFY     = 0x00000001
	NIM_DELETE     = 0x00000002
	NIM_SETVERSION = 0x00000004

	NOTIFYICON_VERSION_4 = 4

	NIF_MESSAGE = 0x00000001
	NIF_ICON    = 0x00000002
	NIF_TIP     = 0x00000004
	NIF_STATE   = 0x00000008
	NIF_INFO    = 0x00000010
	NIF_GUID    = 0x00000020
	NIF_SHOWTIP = 0x00000080

	NIN_SELECT      = WM_USER + 0
	NIN_KEYSELECT   = WM_USER + 1
	NIN_POPUPOPEN   = WM_USER + 6
	NIN_POPUPCLOSE  = WM_USER + 7

	// DWM Window Attributes
	DWMWA_USE_IMMERSIVE_DARK_MODE   = 20
	DWMWA_WINDOW_CORNER_PREFERENCE  = 33
	DWMWA_SYSTEMBACKDROP_TYPE       = 38

	// DWM Corner Preferences
	DWMWCP_DEFAULT    = 0
	DWMWCP_DONOTROUND = 1
	DWMWCP_ROUND      = 2
	DWMWCP_ROUNDSMALL = 3

	// DWM System Backdrop Types
	DWMSBT_AUTO            = 0
	DWMSBT_NONE            = 1
	DWMSBT_MAINWINDOW      = 2 // Mica
	DWMSBT_TRANSIENTWINDOW = 3 // Acrylic (Flyout Mica)
	DWMSBT_TABBEDWINDOW    = 4 // Tabbed Mica

	// Monitor From Point flags
	MONITOR_DEFAULTTONEAREST = 2

	// GDI
	BI_RGB       = 0
	DIB_RGB_COLORS = 0
	SRCCOPY      = 0x00CC0020
	TRANSPARENT  = 1
	OPAQUE       = 2

	// Menu flags
	MF_STRING    = 0x00000000
	MF_SEPARATOR = 0x00000800
	MF_BYCOMMAND = 0x00000000
	TPM_RIGHTBUTTON = 0x0002
	TPM_RETURNCMD   = 0x0100

	// System Metrics
	SM_CXSMICON = 49

	// DNS Interface Settings
	DNS_INTERFACE_SETTINGS_VERSION1 = 1
	DNS_SETTING_NAMESERVER          = 0x00000002

	// MessageBox flags and return values
	MB_OK              = 0x00000000
	MB_YESNO           = 0x00000004
	MB_ICONWARNING     = 0x00000030
	MB_ICONINFORMATION = 0x00000040
	MB_SETFOREGROUND   = 0x00010000
	MB_TOPMOST         = 0x00040000
	IDYES              = 6
	IDNO               = 7
)

// Stable GUID for Nodal tray icon: {9B4A7E20-3C1E-4E6B-A749-644721D05F2B}
var NodalTrayGUID = windows.GUID{
	Data1: 0x9B4A7E20,
	Data2: 0x3C1E,
	Data3: 0x4E6B,
	Data4: [8]byte{0xA7, 0x49, 0x64, 0x47, 0x21, 0xD0, 0x5F, 0x2B},
}
