package main

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework Cocoa -framework Carbon -framework CoreGraphics
// #include "pasteboard.h"
import "C"
