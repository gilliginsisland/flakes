#import <WebKit/WebKit.h>

struct WKWebView;

struct WKWebView *menuet_wk_web_view_new(void);
void menuet_wk_web_view_set_autoresizing_mask(struct WKWebView *view, NSAutoresizingMaskOptions mask);
void menuet_wk_web_view_set_under_page_background_color(struct WKWebView *view, void *color);
void menuet_wk_web_view_load_html_string(struct WKWebView *view, const char *html, void *baseURL);
