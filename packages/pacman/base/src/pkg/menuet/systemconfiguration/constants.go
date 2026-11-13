// Package systemconfiguration provides object wrappers for the native C API.
package systemconfiguration

/*
#cgo LDFLAGS: -framework SystemConfiguration -framework CoreFoundation
#include <SystemConfiguration/SystemConfiguration.h>
*/
import "C"

import cf "github.com/gilliginsisland/pacman/pkg/menuet/corefoundation"

// These are borrowed framework constants; they require no retain or release.
var (
	NetworkProtocolTypeProxies              = cf.StringRef(C.kSCNetworkProtocolTypeProxies)
	PropNetProxiesFTPEnable                 = cf.StringRef(C.kSCPropNetProxiesFTPEnable)
	PropNetProxiesFTPProxy                  = cf.StringRef(C.kSCPropNetProxiesFTPProxy)
	PropNetProxiesGopherEnable              = cf.StringRef(C.kSCPropNetProxiesGopherEnable)
	PropNetProxiesGopherProxy               = cf.StringRef(C.kSCPropNetProxiesGopherProxy)
	PropNetProxiesHTTPEnable                = cf.StringRef(C.kSCPropNetProxiesHTTPEnable)
	PropNetProxiesHTTPProxy                 = cf.StringRef(C.kSCPropNetProxiesHTTPProxy)
	PropNetProxiesHTTPSEnable               = cf.StringRef(C.kSCPropNetProxiesHTTPSEnable)
	PropNetProxiesHTTPSProxy                = cf.StringRef(C.kSCPropNetProxiesHTTPSProxy)
	PropNetProxiesProxyAutoConfigEnable     = cf.StringRef(C.kSCPropNetProxiesProxyAutoConfigEnable)
	PropNetProxiesProxyAutoConfigJavaScript = cf.StringRef(C.kSCPropNetProxiesProxyAutoConfigJavaScript)
	PropNetProxiesProxyAutoConfigURLString  = cf.StringRef(C.kSCPropNetProxiesProxyAutoConfigURLString)
	PropNetProxiesProxyAutoDiscoveryEnable  = cf.StringRef(C.kSCPropNetProxiesProxyAutoDiscoveryEnable)
	PropNetProxiesRTSPEnable                = cf.StringRef(C.kSCPropNetProxiesRTSPEnable)
	PropNetProxiesRTSPProxy                 = cf.StringRef(C.kSCPropNetProxiesRTSPProxy)
	PropNetProxiesSOCKSEnable               = cf.StringRef(C.kSCPropNetProxiesSOCKSEnable)
	PropNetProxiesSOCKSProxy                = cf.StringRef(C.kSCPropNetProxiesSOCKSProxy)
)
