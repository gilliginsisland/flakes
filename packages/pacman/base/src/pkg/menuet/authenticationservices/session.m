#import "session.h"
#import "../core/core.h"

struct ASWebAuthenticationSession *menuet_as_web_authentication_session_new(const char *url, const char *scheme) {
	__block ASWebAuthenticationSession *result;
	menuet_on_main(^{
		NSURL *target = [NSURL URLWithString:[NSString stringWithUTF8String:url]];
		if (target == nil) {
			return;
		}
		NSString *callbackScheme = scheme[0] ? [NSString stringWithUTF8String:scheme] : nil;
		__block __weak ASWebAuthenticationSession *weakSession;
		result = [[ASWebAuthenticationSession alloc]
		          initWithURL:target
		    callbackURLScheme:callbackScheme
		    completionHandler:^(NSURL *callbackURL, NSError *error) {
			    // Hold the session while Go consumes its pointer-keyed completion.
			    ASWebAuthenticationSession *session = weakSession;
			    if (session == nil) {
				    return;
			    }
			    menuet_as_web_authentication_session_did_complete(
			        (__bridge struct ASWebAuthenticationSession *)session,
			        (char *)callbackURL.absoluteString.UTF8String,
			        (char *)error.localizedDescription.UTF8String
			    );
		    }];
		weakSession = result;
	});
	return (__bridge_retained struct ASWebAuthenticationSession *)result;
}

bool menuet_as_web_authentication_session_start(struct ASWebAuthenticationSession *session) {
	__block BOOL started;
	menuet_on_main(^{
		started = [(__bridge ASWebAuthenticationSession *)session start];
	});
	return started;
}

void menuet_as_web_authentication_session_cancel(struct ASWebAuthenticationSession *session) {
	menuet_on_main(^{
		[(__bridge ASWebAuthenticationSession *)session cancel];
	});
}

void menuet_as_web_authentication_session_set_presentation_context_provider(
    struct ASWebAuthenticationSession *session,
    void *provider
) {
	menuet_on_main(^{
		((__bridge ASWebAuthenticationSession *)session).presentationContextProvider =
		    (__bridge id<ASWebAuthenticationPresentationContextProviding>)provider;
	});
}

void menuet_as_web_authentication_session_set_prefers_ephemeral_browser_session(
    struct ASWebAuthenticationSession *session,
    bool ephemeral
) {
	menuet_on_main(^{
		((__bridge ASWebAuthenticationSession *)session).prefersEphemeralWebBrowserSession = ephemeral;
	});
}
