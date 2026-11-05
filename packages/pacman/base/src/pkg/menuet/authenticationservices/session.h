#import <AuthenticationServices/AuthenticationServices.h>
#include <stdint.h>

struct ASWebAuthenticationSession;

struct ASWebAuthenticationSession *
menuet_as_web_authentication_session_new(const char *url, const char *scheme);
bool menuet_as_web_authentication_session_start(struct ASWebAuthenticationSession *session);
void menuet_as_web_authentication_session_cancel(struct ASWebAuthenticationSession *session);
void menuet_as_web_authentication_session_set_presentation_context_provider(
    struct ASWebAuthenticationSession *session,
    void *provider
);
void menuet_as_web_authentication_session_set_prefers_ephemeral_browser_session(
    struct ASWebAuthenticationSession *session,
    bool ephemeral
);
void menuet_as_web_authentication_session_did_complete(
    struct ASWebAuthenticationSession *session,
    char *url,
    char *error
);
