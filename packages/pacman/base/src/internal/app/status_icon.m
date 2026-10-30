#import "status_icon.h"
#import "../../pkg/menuet/core/core.h"
#include <math.h>
#include <stdatomic.h>
#include <stdint.h>

enum { PACmanInputShift = 32 };

// Drawing handlers preserve vector geometry at either display scale.
static NSImage *pacman_frame(double phase) {
	NSImage *image = [NSImage imageWithSize:NSMakeSize(20, 20) flipped:NO drawingHandler:^BOOL(NSRect rect) {
		CGContextRef context = NSGraphicsContext.currentContext.CGContext;
		CGContextSaveGState(context);
		CGContextTranslateCTM(context, rect.origin.x, rect.origin.y);
		CGContextScaleCTM(context, rect.size.width / 20, rect.size.height / 20);
		CGContextSetGrayFillColor(context, 0, 1);
		CGContextSetGrayStrokeColor(context, 0, 1);

		// Open, close as the globe enters the mouth, then open again.
		double angle = (4 + 36 * (1 + cos(2 * M_PI * phase)) / 2) * M_PI / 180;
		CGMutablePathRef body = CGPathCreateMutable();
		CGPathMoveToPoint(body, NULL, 8.5, 10);
		CGPathAddArc(body, NULL, 8.5, 10, 8.2, angle, 2 * M_PI - angle, false);
		CGPathCloseSubpath(body);

		// The body masks the globe as it is swallowed. The next globe
		// approaches from outside the canvas during the second half-cycle.
		CGContextSaveGState(context);
		CGContextAddRect(context, CGRectMake(0, 0, 20, 20));
		CGContextAddPath(context, body);
		CGContextEOClip(context);
		double x = phase < 0.5 ? 16 - 16 * phase : 28 - 12 * phase;
		CGRect globe = CGRectMake(x - 3.3, 6.7, 6.6, 6.6);
		CGContextSetLineWidth(context, 0.7);
		CGContextStrokeEllipseInRect(context, globe);
		CGContextStrokeEllipseInRect(context, CGRectMake(x - 1.5, 6.7, 3, 6.6));
		CGContextStrokeEllipseInRect(context, CGRectMake(x - 3.3, 8.5, 6.6, 3));
		CGContextMoveToPoint(context, x - 3.3, 10);
		CGContextAddLineToPoint(context, x + 3.3, 10);
		CGContextStrokePath(context);
		CGContextRestoreGState(context);

		CGContextAddPath(context, body);
		CGContextAddEllipseInRect(context, CGRectMake(7, 13.5, 2, 2));
		CGContextEOFillPath(context);
		CGPathRelease(body);
		CGContextRestoreGState(context);
		return YES;
	}];
	image.template = YES;
	return image;
}

static double pacman_ease(double progress) {
	progress = fmax(0, fmin(1, progress));
	return progress * progress * (3 - 2 * progress);
}

static NSImage *pacman_key_frame(double angleDegrees, double keyOffset) {
	NSImage *image = [NSImage imageWithSize:NSMakeSize(20, 20) flipped:NO drawingHandler:^BOOL(NSRect rect) {
		CGContextRef context = NSGraphicsContext.currentContext.CGContext;
		CGContextSaveGState(context);
		CGContextTranslateCTM(context, rect.origin.x, rect.origin.y);
		CGContextScaleCTM(context, rect.size.width / 20, rect.size.height / 20);
		CGContextClipToRect(context, CGRectMake(0, 0, 20, 20));
		CGContextSetGrayFillColor(context, 0, 1);
		CGContextSetGrayStrokeColor(context, 0, 1);
		double angle = angleDegrees * M_PI / 180;
		CGMutablePathRef body = CGPathCreateMutable();
		CGPathMoveToPoint(body, NULL, 8.5, 10);
		CGPathAddArc(body, NULL, 8.5, 10, 8.2, angle, 2 * M_PI - angle, false);
		CGPathCloseSubpath(body);
		CGContextSaveGState(context);
		CGContextAddRect(context, CGRectMake(0, 0, 20, 20));
		CGContextAddPath(context, body);
		CGContextEOClip(context);
		CGContextTranslateCTM(context, keyOffset, 0);
		CGContextSetLineWidth(context, 1.05);
		CGContextSetLineCap(context, kCGLineCapRound);
		CGContextStrokeEllipseInRect(context, CGRectMake(14.4, 7.7, 4.6, 4.6));
		CGContextMoveToPoint(context, 14.4, 10);
		CGContextAddLineToPoint(context, 8, 10);
		CGContextMoveToPoint(context, 10.3, 10);
		CGContextAddLineToPoint(context, 10.3, 8.4);
		CGContextMoveToPoint(context, 12.4, 10);
		CGContextAddLineToPoint(context, 12.4, 8.4);
		CGContextStrokePath(context);
		CGContextRestoreGState(context);
		CGContextAddPath(context, body);
		CGContextAddEllipseInRect(context, CGRectMake(7, 13.5, 2, 2));
		CGContextEOFillPath(context);
		CGPathRelease(body);
		CGContextRestoreGState(context);
		return YES;
	}];
	image.template = YES;
	return image;
}

static NSImage *pacman_error_frame(NSImage *sourceImage, double progress) {
	NSImage *image = [NSImage imageWithSize:NSMakeSize(20, 20) flipped:NO drawingHandler:^BOOL(NSRect rect) {
		CGContextRef context = NSGraphicsContext.currentContext.CGContext;
		CGContextSaveGState(context);
		CGContextTranslateCTM(context, rect.origin.x, rect.origin.y);
		CGContextScaleCTM(context, rect.size.width / 20, rect.size.height / 20);
		CGContextClipToRect(context, CGRectMake(0, 0, 20, 20));
		CGContextBeginTransparencyLayer(context, NULL);
		double approach = pacman_ease((progress - 0.1) / 0.7);
		double swallowed = pacman_ease((progress - 0.45) / 0.4);
		double scale = 0.8 + 0.2 * pacman_ease((progress - 0.65) / 0.35);

		// Swallow the current pose, including its globe or stuck key.
		CGContextSaveGState(context);
		CGContextSetAlpha(context, 1 - swallowed);
		CGContextTranslateCTM(context, 10 + 4 * approach, 10);
		CGContextScaleCTM(context, 1 - 0.8 * swallowed, 1 - 0.8 * swallowed);
		[sourceImage drawInRect:NSMakeRect(-10, -10, 20, 20)];
		CGContextRestoreGState(context);

		CGContextTranslateCTM(context, -9 + 19 * approach, 10);
		CGContextScaleCTM(context, scale, scale);
		CGContextTranslateCTM(context, -10, -10);
		CGContextSetGrayFillColor(context, 0, 1);
		CGContextMoveToPoint(context, 2, 2.5);
		CGContextAddLineToPoint(context, 2, 11);
		CGContextAddCurveToPoint(context, 2, 20, 18, 20, 18, 11);
		CGContextAddLineToPoint(context, 18, 2.5);
		CGContextAddLineToPoint(context, 15.5, 4.4);
		CGContextAddLineToPoint(context, 10, 2.5);
		CGContextAddLineToPoint(context, 4.5, 4.4);
		CGContextClosePath(context);
		CGContextFillPath(context);
		CGContextSetBlendMode(context, kCGBlendModeClear);
		CGContextSetLineWidth(context, 1.2);
		CGContextSetLineCap(context, kCGLineCapRound);
		for (NSUInteger i = 0; i < 2; i++) {
			double x = 6.2 + 7.6 * i;
			CGContextMoveToPoint(context, x - 1.5, 10);
			CGContextAddLineToPoint(context, x + 1.5, 13);
			CGContextMoveToPoint(context, x - 1.5, 13);
			CGContextAddLineToPoint(context, x + 1.5, 10);
			CGContextStrokePath(context);
		}
		CGContextEndTransparencyLayer(context);
		CGContextRestoreGState(context);
		return YES;
	}];
	image.template = YES;
	return image;
}

typedef NS_ENUM(NSUInteger, PACmanAnimationState) {
	PACmanAnimationIdle,
	PACmanAnimationConnecting,
	PACmanAnimationInput,
	PACmanAnimationSubmitted,
	PACmanAnimationFailed,
};

static BOOL pacman_adjust_count(_Atomic NSInteger *count, BOOL old, BOOL next) {
	if (old == next) {
		return NO;
	}
	NSInteger delta = next ? 1 : -1;
	NSInteger previous = atomic_fetch_add(count, delta);
	return previous == 0 || previous + delta == 0;
}

@interface PACmanStatusAnimation : NSObject {
	__weak NSStatusItem *_status;
	NSImage *_idleImage;
	NSArray<NSImage *> *_frames;
	NSArray<NSImage *> *_playback;
	PACmanAnimationState _state;
	_Atomic NSInteger _connecting;
	_Atomic NSInteger _failed;
	// Low 32 bits: aggregate dialer activity. High 32 bits: active app inputs.
	_Atomic uint64_t _activity;
	NSTimer *_timer;
	NSUInteger _frame;
}
- (instancetype)initWithStatus:(NSStatusItem *)status;
- (void)updateFrom:(PACmanStatusState)old to:(PACmanStatusState)next;
- (void)addInput:(NSInteger)delta;
@end

@implementation PACmanStatusAnimation

- (instancetype)initWithStatus:(NSStatusItem *)status {
	self = [super init];
	if (self) {
		_status = status;
		_idleImage = status.button.image;
		NSMutableArray<NSImage *> *frames = [NSMutableArray new];
		for (NSUInteger i = 0; i < 24; i++) {
			[frames addObject:pacman_frame((double)i / 24)];
		}
		_frames = frames;
	}
	return self;
}

- (void)updateFrom:(PACmanStatusState)old to:(PACmanStatusState)next {
	BOOL changed = pacman_adjust_count(&_connecting, old == PACmanStatusConnecting, next == PACmanStatusConnecting);
	changed = pacman_adjust_count(&_failed, old == PACmanStatusFailed, next == PACmanStatusFailed) || changed;
	if (!changed) {
		return;
	}
	menuet_on_main(^{
		// Read here so concurrent transitions cannot apply stale start/stop decisions.
		[self updateState];
	});
}

- (void)addInput:(NSInteger)delta {
	uint64_t change = (uint64_t)delta << PACmanInputShift;
	uint64_t previous = atomic_fetch_add(&_activity, change);
	if ((previous >> PACmanInputShift) != 0 && ((previous + change) >> PACmanInputShift) != 0) {
		return;
	}
	menuet_on_main(^{
		[self updateState];
	});
}

- (void)updateState {
	PACmanStatusState base = atomic_load(&_failed) > 0 ? PACmanStatusFailed :
	                        atomic_load(&_connecting) > 0 ? PACmanStatusConnecting : PACmanStatusIdle;
	uint64_t previous = atomic_load(&_activity);
	uint64_t status;
	do {
		status = (previous & ~((uint64_t)UINT32_MAX)) | base;
	} while (!atomic_compare_exchange_weak(&_activity, &previous, status));
	if ((status >> PACmanInputShift) != 0) {
		if (_state == PACmanAnimationInput || (_state == PACmanAnimationSubmitted && _timer)) {
			return;
		}
		NSMutableArray<NSImage *> *frames = [NSMutableArray new];
		NSUInteger start = _state == PACmanAnimationConnecting ? _frame : 0;
		if (_state != PACmanAnimationConnecting) {
			[frames addObject:_status.button.image];
		}
		// Finish swallowing the current globe before the key arrives.
		NSUInteger count = start < 12 ? 12 - start : 36 - start;
		for (NSUInteger i = 0; i < count; i++) {
			[frames addObject:_frames[(start + i) % _frames.count]];
		}
		for (NSUInteger i = 0; i < 24; i++) {
			double progress = (double)i / 23;
			double angle = 4 + 36 * pacman_ease(progress * 2) - 14 * pacman_ease((progress - 0.65) / 0.35);
			[frames addObject:pacman_key_frame(angle, 20 * (1 - pacman_ease(progress)))];
		}
		[self playFrames:frames state:PACmanAnimationInput];
		return;
	}
	if (base == PACmanStatusFailed) {
		if (_state == PACmanAnimationFailed) {
			return;
		}
		NSMutableArray<NSImage *> *frames = [NSMutableArray new];
		for (NSUInteger i = 0; i < 48; i++) {
			NSImage *source = _state == PACmanAnimationConnecting ? _frames[(_frame + i) % _frames.count] : _status.button.image;
			[frames addObject:pacman_error_frame(source, (double)i / 47)];
		}
		[self playFrames:frames state:PACmanAnimationFailed];
		return;
	}
	if (_state == PACmanAnimationInput) {
		[self submitInput];
		return;
	}
	if (_state == PACmanAnimationSubmitted && _timer) {
		return;
	}
	if (base == PACmanStatusConnecting) {
		if (_state == PACmanAnimationConnecting) {
			return;
		}
		[self playFrames:_frames state:PACmanAnimationConnecting];
		return;
	}
	[_timer invalidate];
	_timer = nil;
	_playback = nil;
	_state = PACmanAnimationIdle;
	_frame = 0;
	_status.button.image = _idleImage;
}

- (void)playFrames:(NSArray<NSImage *> *)frames state:(PACmanAnimationState)state {
	[_timer invalidate];
	_playback = frames;
	_state = state;
	_frame = 0;
	_status.button.image = frames[0];
	__weak PACmanStatusAnimation *weakSelf = self;
	_timer = [NSTimer timerWithTimeInterval:1.0 / 24 repeats:YES block:^(NSTimer *timer) {
		PACmanStatusAnimation *animation = weakSelf;
		if (animation) {
			NSArray<NSImage *> *frames = animation->_playback;
			animation->_frame = (animation->_frame + 1) % frames.count;
			animation->_status.button.image = frames[animation->_frame];
			if (animation->_state != PACmanAnimationConnecting && animation->_frame == frames.count - 1) {
				[timer invalidate];
				animation->_timer = nil;
				if (animation->_state == PACmanAnimationSubmitted) {
					[animation updateState];
				}
			}
		}
	}];
	[NSRunLoop.mainRunLoop addTimer:_timer forMode:NSRunLoopCommonModes];
}

- (void)submitInput {
	NSMutableArray<NSImage *> *frames = [[_playback subarrayWithRange:NSMakeRange(_frame, _playback.count - _frame)] mutableCopy];
	for (NSUInteger i = 0; i < 24; i++) {
		double progress = pacman_ease((double)i / 23);
		[frames addObject:pacman_key_frame(26 - 22 * progress, -16 * progress)];
	}
	// Reopen into the connecting cycle after swallowing the key.
	for (NSUInteger i = 12; i <= 24; i++) {
		[frames addObject:_frames[i % _frames.count]];
	}
	[self playFrames:frames state:PACmanAnimationSubmitted];
}

- (void)dealloc {
	// Go cleanup may release the owner on a background thread.
	NSTimer *timer = _timer;
	dispatch_async(dispatch_get_main_queue(), ^{
		[timer invalidate];
	});
}

@end

struct PACmanStatusAnimation *pacman_status_animation_new(struct NSStatusItem *status) {
	__block PACmanStatusAnimation *animation;
	menuet_on_main(^{
		animation = [[PACmanStatusAnimation alloc] initWithStatus:(__bridge NSStatusItem *)status];
	});
	return (__bridge_retained struct PACmanStatusAnimation *)animation;
}

void pacman_status_animation_update_state(struct PACmanStatusAnimation *animation, PACmanStatusState old, PACmanStatusState next) {
	[(__bridge PACmanStatusAnimation *)animation updateFrom:old to:next];
}

void pacman_status_animation_add_input(struct PACmanStatusAnimation *animation, NSInteger delta) {
	[(__bridge PACmanStatusAnimation *)animation addInput:delta];
}
