#ifndef BRIDGE_H
#define BRIDGE_H

#include <openconnect.h>

struct openconnect_info *go_vpninfo_new(const char *useragent, void *privdata);
void go_set_csd_callback(struct openconnect_info *vpninfo, int enabled);
extern const struct openconnect_packet_bridge go_packet_bridge_callbacks;

#endif
