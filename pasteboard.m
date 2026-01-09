#import <Cocoa/Cocoa.h>
#import <Carbon/Carbon.h>
#include "pasteboard.h"
#import <ApplicationServices/ApplicationServices.h>

PasteboardContent getPasteboardContent(void) {
    PasteboardContent result = {NULL, 0, NULL};
    NSPasteboard *pb = [NSPasteboard generalPasteboard];

    NSArray *types = @[
        NSPasteboardTypePNG,
        NSPasteboardTypeTIFF,
        @"public.jpeg",
        NSPasteboardTypePDF,
        NSPasteboardTypeString
    ];

    for (NSString *type in types) {
        NSData *data = [pb dataForType:type];
        if (data) {
            result.data = malloc([data length]);
            memcpy(result.data, [data bytes], [data length]);
            result.length = (int)[data length];
            result.uti = strdup([type UTF8String]);
            return result;
        }
    }

    NSString *str = [pb stringForType:NSPasteboardTypeString];
    if (str) {
        const char *utf8 = [str UTF8String];
        result.length = (int)strlen(utf8);
        result.data = malloc(result.length);
        memcpy(result.data, utf8, result.length);
        result.uti = strdup("public.utf8-plain-text");
    }

    return result;
}

void setPasteboardData(const void* data, int length, const char* uti) {
    NSPasteboard *pb = [NSPasteboard generalPasteboard];
    [pb clearContents];

    NSString *utiStr = [NSString stringWithUTF8String:uti];

    if ([utiStr isEqualToString:@"public.utf8-plain-text"]) {
        NSString *str = [[NSString alloc] initWithBytes:data length:length encoding:NSUTF8StringEncoding];
        [pb setString:str forType:NSPasteboardTypeString];
    } else {
        NSData *nsdata = [NSData dataWithBytes:data length:length];
        [pb setData:nsdata forType:utiStr];
    }
}

static void emitKeyCombo(int keycode, bool cmd, bool shift) {
    CGEventSourceRef source = CGEventSourceCreate(kCGEventSourceStateHIDSystemState);
    CGEventRef keyDown = CGEventCreateKeyboardEvent(source, keycode, true);
    CGEventRef keyUp = CGEventCreateKeyboardEvent(source, keycode, false);

    CGEventFlags flags = 0;
    if (cmd) flags |= kCGEventFlagMaskCommand;
    if (shift) flags |= kCGEventFlagMaskShift;

    CGEventSetFlags(keyDown, flags);
    CGEventSetFlags(keyUp, flags);

    CGEventPost(kCGSessionEventTap, keyDown);
    CGEventPost(kCGSessionEventTap, keyUp);

    CFRelease(keyDown);
    CFRelease(keyUp);
    CFRelease(source);
}

void emitCopy(void) {
    emitKeyCombo(8, true, false);
}


void emitPaste(void) {
    emitKeyCombo(9, true, false);
    usleep(100000);
}

void freeContent(PasteboardContent c) {
    if (c.data) free(c.data);
    if (c.uti) free(c.uti);
}
