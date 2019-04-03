#include "_cgo_export.h"
#import <Carbon/Carbon.h>

// void RunApplicationEventLoop();

// OSStatus keyHandler(EventHandlerCallRef nextHandler, EventRef anEvent, void *userData) {
//   EventHotKeyID key;
//   GetEventParameter(anEvent, kEventParamDirectObject, typeEventHotKeyID, NULL, sizeof(key), NULL, &key);
//   keyGoCallback();
//   return noErr;
// }

// int runKeyHandler(int virtualKey) {
//   EventTypeSpec eventType;
//   eventType.eventClass = kEventClassKeyboard;
//   eventType.eventKind = kEventHotKeyPressed;

//   InstallEventHandler(GetApplicationEventTarget(), &keyHandler, 1, &eventType, 0, 0);

//   EventHotKeyRef helpKeyRef;
//   EventHotKeyID helpKey;
//   helpKey.signature = 'htk1';
//   helpKey.id = 1;

//   RegisterEventHotKey(virtualKey, 0, helpKey, GetApplicationEventTarget(), kEventHotKeyExclusive, &helpKeyRef);
//   RunApplicationEventLoop();

//   return 0;
// }

static CFMachPortRef globalEventTap = NULL;
static CFRunLoopRef globalRunLoop = NULL;
static CFRunLoopSourceRef globalEventTapSource = NULL;

// The following callback method is invoked on every keypress.
CGEventRef eventCallback(CGEventTapProxy proxy, CGEventType type, CGEventRef event, void *user) {

  if (type != kCGEventKeyDown) {
    // we seemingly don't get called with kCGEventTapDisabledByTimeout, but enable here just-in-case
    printf("got type=%d\n", type);
    fflush(stdout);
    CGEventTapEnable(globalEventTap, true);
    return event;
  }

  // Retrieve the incoming keycode.
  CGKeyCode keyCode = (CGKeyCode) CGEventGetIntegerValueField(event, kCGKeyboardEventKeycode);
  if (keyCode == 111) {
    printf("F12\n");
  } else {
    // ignore
  }
  fflush(stdout);

  return event;
}

int refreshEventTap() {
  if (globalEventTap) {
    CGEventTapEnable(globalEventTap, false);
    CFRunLoopRemoveSource(globalRunLoop, globalEventTapSource, kCFRunLoopDefaultMode);
    CFRelease(globalEventTap);
    CFRelease(globalEventTapSource);
    globalEventTap = NULL;
    globalEventTapSource = NULL;
  }

  CGEventMask mask = CGEventMaskBit(kCGEventKeyDown);
  globalEventTap = CGEventTapCreate(kCGHIDEventTap, kCGTailAppendEventTap, kCGEventTapOptionDefault, mask, eventCallback, globalEventTap);
  if (!globalEventTap) {
    fprintf(stderr, "ERROR: Unable to create event tap: %d\n", errno);
    return 1;
  }
  CGEventTapEnable(globalEventTap, true);

  globalEventTapSource = CFMachPortCreateRunLoopSource(kCFAllocatorDefault, globalEventTap, 0);
  CFRunLoopAddSource(globalRunLoop, globalEventTapSource, kCFRunLoopDefaultMode);

  return 0;
}

void timerCallback(CFRunLoopTimerRef timer, void *user) {
  if (globalEventTap) {
    bool enabled = CGEventTapIsEnabled(globalEventTap);
    if (enabled) {
      return;  // ok
    }

    CGEventTapEnable(globalEventTap, true);  // try to enable again
    enabled = CGEventTapIsEnabled(globalEventTap);
    if (enabled) {
      return;  // ok
    }
  }

  printf("got timer, tap failed to enable errno=%d\n", errno);
  if (refreshEventTap()) {
    // nb. probably happens while root at loginwindow
    printf("couldn't recreate even tap :(\n");
  }
  fflush(stdout);
}

int runKeyHandler() {
  globalRunLoop = CFRunLoopGetCurrent();

  if (refreshEventTap()) {
    fprintf(stderr, "ERROR: Unable to create event tap: %d\n", errno);
    return 1;
  }

  double runEverySeconds = 3.6;
  CFRunLoopTimerRef timer = CFRunLoopTimerCreate(NULL, 0.0, runEverySeconds, 0, 0, timerCallback, NULL);
  CFRunLoopAddTimer(globalRunLoop, timer, kCFRunLoopDefaultMode);

  CFRunLoopRun();
  return 0;
}