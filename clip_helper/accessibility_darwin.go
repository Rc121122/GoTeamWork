//go:build darwin

package clip_helper

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework ApplicationServices
#include <ApplicationServices/ApplicationServices.h>

bool gtw_is_process_trusted() {
    return AXIsProcessTrusted();
}

bool gtw_request_accessibility_permission() {
    const void *keys[] = { kAXTrustedCheckOptionPrompt };
    const void *values[] = { kCFBooleanTrue };
    CFDictionaryRef options = CFDictionaryCreate(kCFAllocatorDefault,
        keys,
        values,
        1,
        &kCFTypeDictionaryKeyCallBacks,
        &kCFTypeDictionaryValueCallBacks);
    bool trusted = AXIsProcessTrustedWithOptions(options);
    CFRelease(options);
    return trusted;
}
*/
import "C"

func HasAccessibilityPermission() bool {
	return bool(C.gtw_is_process_trusted())
}

func RequestAccessibilityPermission() bool {
	return bool(C.gtw_request_accessibility_permission())
}
