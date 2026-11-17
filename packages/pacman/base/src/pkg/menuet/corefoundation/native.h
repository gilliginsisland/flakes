#ifndef MENUET_COREFOUNDATION_NATIVE_H
#define MENUET_COREFOUNDATION_NATIVE_H

#include <CoreFoundation/CoreFoundation.h>

// Keep CF reference conversions to the collection APIs' void pointers in C.
static inline CFTypeRef menuet_cf_dictionary_get_value(CFDictionaryRef dictionary, CFStringRef key) {
    return CFDictionaryGetValue(dictionary, key);
}

static inline void menuet_cf_dictionary_set_value(CFMutableDictionaryRef dictionary, CFStringRef key, CFTypeRef value) {
    CFDictionarySetValue(dictionary, key, value);
}

static inline void menuet_cf_dictionary_remove_value(CFMutableDictionaryRef dictionary, CFStringRef key) {
    CFDictionaryRemoveValue(dictionary, key);
}

static inline CFTypeRef menuet_cf_array_get_value(CFArrayRef array, CFIndex index) {
    return CFArrayGetValueAtIndex(array, index);
}

#endif
