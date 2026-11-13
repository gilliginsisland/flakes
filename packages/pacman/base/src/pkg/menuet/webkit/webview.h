#import <WebKit/WebKit.h>

struct NSColor;
struct NSURL;
struct WKWebView;
struct NSURLRequest;

struct WKWebView *menuet_wk_web_view_new(void);
void menuet_wk_web_view_set_autoresizing_mask(struct WKWebView *view, NSAutoresizingMaskOptions mask);
void menuet_wk_web_view_set_under_page_background_color(struct WKWebView *view, struct NSColor *color);
void menuet_wk_web_view_load_html_string(struct WKWebView *view, const char *html, struct NSURL *baseURL);
void menuet_wk_web_view_load_request(struct WKWebView *view, struct NSURLRequest *request);
