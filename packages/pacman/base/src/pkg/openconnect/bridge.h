#ifndef BRIDGE_H
#define BRIDGE_H

#include <openconnect.h>

struct openconnect_info *go_vpninfo_new(const char *useragent, void *privdata);
int go_mainloop(struct openconnect_info *vpninfo);

#endif
