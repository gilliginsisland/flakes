#import <Cocoa/Cocoa.h>
#import <WebKit/WebKit.h>

void openWebView(const char *chtml) {
    [[NSRunLoop mainRunLoop] performInModes:@[NSRunLoopCommonModes] block: ^{
        // Convert C string to NSString
        NSString *html = [NSString stringWithUTF8String:chtml];

        WKWebView *webView = [[WKWebView alloc] initWithFrame:NSZeroRect];
        webView.autoresizingMask = NSViewWidthSizable | NSViewHeightSizable;
        webView.underPageBackgroundColor = NSColor.clearColor;
        [webView loadHTMLString:html baseURL:nil];

        // Create a window to host the scroll view
        NSWindow *window = [[NSWindow alloc]
            initWithContentRect:NSMakeRect(0, 0, 600, 400)
            styleMask:NSWindowStyleMaskTitled | NSWindowStyleMaskClosable | NSWindowStyleMaskResizable
            backing:NSBackingStoreBuffered
            defer:NO];
        window.title = @"Help";
        window.level = NSNormalWindowLevel;
        window.releasedWhenClosed = NO;
        window.contentView = webView;

        [window center]; // Center the window on the screen
        [window makeKeyAndOrderFront:nil]; // Show the window

        [NSApp activateIgnoringOtherApps:YES];
    }];
}
