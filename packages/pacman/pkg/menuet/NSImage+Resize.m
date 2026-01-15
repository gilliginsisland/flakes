#import "NSImage+Resize.h"

@interface ImageCache : NSObject
+ (ImageCache *)instance;
- (void)setImage:(NSImage *)image forName:(NSString *)name withHeight:(CGFloat)height;
- (NSImage *)getImageForName:(NSString *)name withHeight:(CGFloat)height;
@property(nonatomic, strong) NSCache *imageCache;
@end

static ImageCache *instance;

@implementation ImageCache

+ (ImageCache *)instance {
	static dispatch_once_t onceToken;
	dispatch_once(&onceToken, ^{
		instance = [[ImageCache alloc] init];
	});
	return instance;
}

- (instancetype)init {
	self = [super init];
	if (self) {
		self.imageCache = [[NSCache alloc] init];
	}
	return self;
}

- (NSString *)keyForName:(NSString *)name withHeight:(CGFloat)height {
	return [NSString stringWithFormat:@"%f/%@", height, name];
}

- (void)setImage:(NSImage *)image forName:(NSString *)name withHeight:(CGFloat)height {
	NSString* key = [self keyForName:name withHeight:height];
	[self.imageCache setObject:image forKey:key];
}

- (NSImage *)getImageForName:(NSString *)name withHeight:(CGFloat)height {
	NSString* key = [self keyForName:name withHeight:height];
	return [self.imageCache objectForKey:key];
}

@end

@implementation NSImage (Resize)

- (NSImage *)imageWithHeight:(CGFloat)height {
	if (!self.isValid) {
		NSLog(@"Can't resize invalid image");
		return nil;
	}
	NSSize size = NSMakeSize(self.size.width * height / self.size.height, height);
	NSImage *image = [[NSImage alloc] initWithSize:size];
	[image lockFocus];
	self.size = size;
	[NSGraphicsContext.currentContext setImageInterpolation:NSImageInterpolationDefault];
	CGRect rect = CGRectMake(0, 0, size.width, size.height);
	[self drawAtPoint:NSZeroPoint fromRect:rect operation:NSCompositingOperationCopy fraction:1.0];
	[image unlockFocus];
	return image;
}

+ (NSImage *)imageFromName:(NSString *)name withHeight:(CGFloat)height {
	if (name.length == 0) {
		return nil;
	}
	NSImage *image = [ImageCache.instance getImageForName:name withHeight:height];
	if (image != nil) {
		return image;
	}
	if ([name hasPrefix:@"http"]) {
		image = [[NSImage alloc] initWithContentsOfURL:[NSURL URLWithString:name]];
	} else {
		image = [NSImage imageNamed:name];
	}
	if (height > 0 && image.size.height > height) {
		image = [image imageWithHeight:height];
	}
	[ImageCache.instance setImage:image forName:name withHeight:height];
	return image;
}

@end
