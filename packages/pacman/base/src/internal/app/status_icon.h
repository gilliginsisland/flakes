#import <Cocoa/Cocoa.h>

struct PACmanStatusAnimation;
struct NSStatusItem;

// Values mirror dialer.ConnectionState.
typedef NS_ENUM(NSUInteger, PACmanStatusState) {
	PACmanStatusIdle = 0,
	PACmanStatusConnecting = 1,
	PACmanStatusFailed = 2,
	PACmanStatusOnline = 3,
};

struct PACmanStatusAnimation *pacman_status_animation_new(struct NSStatusItem *status);
void pacman_status_animation_update_state(struct PACmanStatusAnimation *animation, PACmanStatusState old, PACmanStatusState next);
void pacman_status_animation_add_input(struct PACmanStatusAnimation *animation, NSInteger delta);
