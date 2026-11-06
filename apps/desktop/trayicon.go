package main

import _ "embed"

// trayIconPNG is the system-tray / notification-area icon (same asset as packaging).
//
//go:embed build/appicon.png
var trayIconPNG []byte
