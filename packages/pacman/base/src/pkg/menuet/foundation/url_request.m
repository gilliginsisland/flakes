#import "url_request.h"

struct NSURLRequest *menuet_ns_url_request_new(struct NSURL *url) {
	@autoreleasepool {
		return (__bridge_retained struct NSURLRequest *)[NSURLRequest requestWithURL:(__bridge NSURL *)url];
	}
}
