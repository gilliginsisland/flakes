#ifndef BRIDGE_H
#define BRIDGE_H

#include <openconnect.h>

struct openconnect_info *go_vpninfo_new(const char *useragent, void *privdata);
void go_set_csd_callback(struct openconnect_info *vpninfo, int enabled);
int go_mainloop(struct openconnect_info *vpninfo);

#endif
