//go:build darwin
// +build darwin

package clip_helper

/*
#cgo darwin CFLAGS: -x objective-c -fobjc-arc
#cgo darwin LDFLAGS: -framework Cocoa

#import <Cocoa/Cocoa.h>
#import <CoreFoundation/CoreFoundation.h>
#import <stdlib.h>

static CFArrayRef GetPasteboardFilePaths() {
    NSPasteboard *pasteboard = [NSPasteboard generalPasteboard];
    NSArray<NSPasteboardItem *> *items = [pasteboard pasteboardItems];
    NSMutableArray<NSString *> *paths = [NSMutableArray array];

    for (NSPasteboardItem *item in items) {
        NSString *fileURLString = [item stringForType:NSPasteboardTypeFileURL];
        if (fileURLString.length > 0) {
            NSURL *url = [NSURL URLWithString:fileURLString];
            if (url.isFileURL && url.path.length > 0) {
                [paths addObject:url.path];
                continue;
            }
        }

        id propertyList = [item propertyListForType:NSFilenamesPboardType];
        if ([propertyList isKindOfClass:[NSArray class]] && [propertyList count] > 0) {
            NSString *path = [propertyList firstObject];
            if (path.length > 0) {
                [paths addObject:path];
            }
        }
    }

    if (paths.count == 0) {
        NSArray<NSURL *> *urls = [pasteboard readObjectsForClasses:@[[NSURL class]]
                                                          options:@{ NSPasteboardURLReadingFileURLsOnlyKey : @YES }];
        for (NSURL *url in urls) {
            if (url.path.length > 0) {
                [paths addObject:url.path];
            }
        }
    }

    if (paths.count == 0) {
        return NULL;
    }

    return (__bridge_retained CFArrayRef)paths;
}

static char *CopyUTF8String(CFStringRef str) {
    if (str == NULL) {
        return NULL;
    }

    CFIndex length = CFStringGetLength(str);
    CFIndex maxSize = CFStringGetMaximumSizeForEncoding(length, kCFStringEncodingUTF8) + 1;
    char *buffer = (char *)malloc((size_t)maxSize);
    if (buffer == NULL) {
        return NULL;
    }

    if (CFStringGetCString(str, buffer, maxSize, kCFStringEncodingUTF8)) {
        return buffer;
    }

    free(buffer);
    return NULL;
}
*/
import "C"
import "unsafe"

// getFilePathsFromPasteboard returns absolute file paths currently stored in the macOS pasteboard.
func getFilePathsFromPasteboard() []string {
	arr := C.GetPasteboardFilePaths()
	if arr == 0 {
		return nil
	}
	defer C.CFRelease(C.CFTypeRef(arr))

	count := C.CFArrayGetCount(arr)
	paths := make([]string, 0, int(count))
	for i := C.CFIndex(0); i < count; i++ {
		value := C.CFArrayGetValueAtIndex(arr, i)
		cfStr := C.CFStringRef(value)
		if cfStr == 0 {
			continue
		}

		cstr := C.CopyUTF8String(cfStr)
		if cstr == nil {
			continue
		}

		paths = append(paths, C.GoString(cstr))
		C.free(unsafe.Pointer(cstr))
	}

	return paths
}
