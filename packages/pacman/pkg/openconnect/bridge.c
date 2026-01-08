#include <openconnect.h>
#include <stdarg.h>
#include <stdio.h>
#include <stdlib.h>
#include <pthread.h>

extern int go_validate_peer_cert(void *context, char *cert);
extern int go_process_auth_form(void *context, struct oc_auth_form *form);
extern void go_progress(void *context, int level, char *message);
extern int go_external_browser_callback(struct openconnect_info *vpninfo, char *uri, void *context);
extern void go_reconnected_handler(void *context);
extern void go_mainloop_result(struct openconnect_info *vpninfo, int result);

void go_progress_vargs(void *context, int level, const char *fmt, ...) {
	va_list args;
	va_start(args, fmt);

	// First call to vsnprintf with NULL to get required length
	int needed = vsnprintf(NULL, 0, fmt, args) + 1; // +1 for null terminator
	va_end(args);

	if (needed <= 0) {
		return; // Formatting error, do nothing
	}

	// Allocate memory dynamically
	char *buffer = (char *)malloc(needed);
	if (!buffer) {
		return; // Memory allocation failed, do nothing
	}

	// Format the message into the allocated buffer
	va_start(args, fmt);
	vsnprintf(buffer, needed, fmt, args);
	va_end(args);

	// Call the Go function with the formatted string
	go_progress(context, level, buffer);

	// Free allocated memory
	free(buffer);
}

struct openconnect_info *go_vpninfo_new(const char *useragent, void *privdata) {
	struct openconnect_info *vpninfo = openconnect_vpninfo_new(
		useragent, (openconnect_validate_peer_cert_vfn) go_validate_peer_cert,
		NULL, // Config Writer (Not implemented yet)
		(openconnect_process_auth_form_vfn) go_process_auth_form,
		go_progress_vargs,
		privdata
	);
	openconnect_set_external_browser_callback(
		vpninfo,
		(openconnect_open_webview_vfn) go_external_browser_callback
	);
	openconnect_set_reconnected_handler(
		vpninfo,
		(openconnect_reconnected_vfn) go_reconnected_handler
	);
	return vpninfo;
}

void* run_mainloop(void* arg) {
	struct openconnect_info *vpninfo = (struct openconnect_info *)arg;
	int result = openconnect_mainloop(vpninfo, 5, RECONNECT_INTERVAL_MIN);
	go_mainloop_result(vpninfo, result);
	return NULL;
}

int go_mainloop(struct openconnect_info *vpninfo) {
	pthread_t thread;
	int result = pthread_create(&thread, NULL, run_mainloop, vpninfo);
	if (result == 0) {
		pthread_detach(thread);
	}
	return -result;
}
