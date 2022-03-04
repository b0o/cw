package main

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

var out = os.Stderr

func usage() {
	fmt.Fprintf(out, "Usage: %s [color]\n", path.Base(os.Args[0]))
}

// expandShortColor takes a short 16-bit representation of a 32-bit
// color and expands it to the full version by duplicating each 4 bits.
//
// For example:
//   0x0fff -> 0x00ffffff
//   0xffff -> 0xffffffff
//   0x0abc -> 0x00aabbcc
//   0x10e2 -> 0x1100ee22
func expandShortColor(n16 uint16) uint32 {
	n := uint32(n16)
	return n<<16&0xf0000000 +
		n<<12&0x0ff00000 +
		n<<8&0x000ff000 +
		n<<4&0x00000ff0 +
		n&0x0000000f
}

// parseColor attempts to convert a hexadecimal string representation of one of
// the following formats into a 32-bit argb color:
//
//   rgb
//   argb
//   rrggbb
//   aarrggbb
//
// The string can optionally be prefixed with "0x" or "#", but neither sequence
// has any effect on the behavior of this function.
//
// See: https://www.w3.org/TR/css-color-4/#hex-color, but note that in our case
// the alpha channel is specified by the highest byte rather than the lowest.
func parseColor(s string) (uint32, error) {
	s = strings.TrimPrefix(strings.TrimPrefix(strings.ToLower(s), "0x"), "#")

	switch len(s) {
	case 3:
		i, err := strconv.ParseUint(s, 16, 12)
		return expandShortColor(0xf000 + uint16(i&0x0fff)), err

	case 4:
		i, err := strconv.ParseUint(s, 16, 16)
		return expandShortColor(uint16(i & 0xffff)), err

	case 6:
		i, err := strconv.ParseUint(s, 16, 24)
		return 0xff000000 + uint32(i), err

	case 8:
		i, err := strconv.ParseUint(s, 16, 32)
		return uint32(i), err

	default:
		return 0, fmt.Errorf("invalid length: %d", len(s))
	}
}

func main() {
	var bgColor uint32 = 0xffffffff

	switch len(os.Args[1:]) {
	case 0:
		break
	case 1:
		var err error
		bgColor, err = parseColor(os.Args[1])
		if err != nil {
			fmt.Fprintf(out, "Invalid color: %s\n", os.Args[1])
			os.Exit(1)
		}
		break
	default:
		usage()
		os.Exit(1)
	}

	X, err := xgb.NewConn()
	if err != nil {
		fmt.Fprintln(out, err)
		return
	}

	// Based on https://github.com/jezek/xgb/blob/v1.0.0/examples/create-window/main.go

	// xproto.Setup retrieves the Setup information from the setup bytes
	// gathered during connection.
	setup := xproto.Setup(X)

	// This is the default screen with all its associated info.
	screen := setup.DefaultScreen(X)

	// Any time a new resource (i.e., a window, pixmap, graphics context, etc.)
	// is created, we need to generate a resource identifier.
	// If the resource is a window, then use xproto.NewWindowId. If it's for
	// a pixmap, then use xproto.NewPixmapId. And so on...
	wid, _ := xproto.NewWindowId(X)

	// CreateWindow takes a boatload of parameters.
	xproto.CreateWindow(X, screen.RootDepth, wid, screen.Root,
		0, 0, 500, 500, 0,
		xproto.WindowClassInputOutput, screen.RootVisual, 0, []uint32{})

	title := []byte("cw")
	xproto.ChangeProperty(X, xproto.PropModeReplace, wid,
		xproto.AtomWmName, xproto.AtomString,
		byte(8), uint32(len(title)), title)

	// This call to ChangeWindowAttributes could be factored out and
	// included with the above CreateWindow call, but it is left here for
	// instructive purposes. It tells X to send us events when the 'structure'
	// of the window is changed (i.e., when it is resized, mapped, unmapped,
	// etc.) and when a key press or a key release has been made when the
	// window has focus.
	xproto.ChangeWindowAttributes(X, wid,
		xproto.CwBackPixel|xproto.CwEventMask,
		[]uint32{ // values must be in the order defined by the protocol
			bgColor,
			xproto.EventMaskStructureNotify |
				xproto.EventMaskKeyPress |
				xproto.EventMaskKeyRelease})

	// MapWindow makes the window we've created appear on the screen.
	// We demonstrated the use of a 'checked' request here.
	// A checked request is a fancy way of saying, "do error handling
	// synchronously." Namely, if there is a problem with the MapWindow request,
	// we'll get the error *here*. If we were to do a normal unchecked
	// request (like the above CreateWindow and ChangeWindowAttributes
	// requests), then we would only see the error arrive in the main event
	// loop.
	//
	// Typically, checked requests are useful when you need to make sure they
	// succeed. Since they are synchronous, they incur a round trip cost before
	// the program can continue, but this is only going to be noticeable if
	// you're issuing tons of requests in succession.
	//
	// Note that requests without replies are by default unchecked while
	// requests *with* replies are checked by default.
	err = xproto.MapWindowChecked(X, wid).Check()
	if err != nil {
		fmt.Fprintf(out, "Checked Error for mapping window %d: %s\n", wid, err)
	} else {
		fmt.Fprintf(out, "Map window %d successful!\n", wid)
	}

	// Start the main event loop.
	for {
		// WaitForEvent either returns an event or an error and never both.
		// If both are nil, then something went wrong and the loop should be
		// halted.
		//
		// An error can only be seen here as a response to an unchecked
		// request.
		ev, xerr := X.WaitForEvent()
		if ev == nil && xerr == nil {
			fmt.Fprintln(out, "Both event and error are nil. Exiting...")
			os.Exit(0)
		}

		if ev != nil {
			fmt.Fprintf(out, "Event: %s\n", ev)
		}
		if xerr != nil {
			fmt.Fprintf(out, "Error: %s\n", xerr)
		}
	}
}
