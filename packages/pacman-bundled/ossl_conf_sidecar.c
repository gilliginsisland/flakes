#include <stdlib.h>
#include <stdio.h>
#include <libgen.h>
#include <mach-o/dyld.h>
#include <limits.h>

__attribute__((constructor))
static void setup_ossl_conf_sidecar() {
	char path[PATH_MAX];
	uint32_t size = sizeof(path);

	if (_NSGetExecutablePath(path, &size) == 0) {
		char *exe_dir = dirname(path);
		char buf[PATH_MAX];

		// Set OPENSSL_CONF relative to your app bundle
		snprintf(buf, sizeof(buf), "%s/../Resources/ssl/openssl.cnf", exe_dir);
		setenv("OPENSSL_CONF", buf, 1);

		snprintf(buf, sizeof(buf), "%s/../Resources/ssl/ct_log_list.cnf", exe_dir);
		setenv("CTLOG_FILE", buf, 1);

		// Set OPENSSL_MODULES for OSSL 3.x providers
		snprintf(buf, sizeof(buf), "%s/../PlugIns/ossl-modules", exe_dir);
		setenv("OPENSSL_MODULES", buf, 1);

		// Set OPENSSL_ENGINES - For OpenSSL 1.1.x or 3.x legacy engines
		snprintf(buf, sizeof(buf), "%s/../PlugIns/engines-3", exe_dir);
		setenv("OPENSSL_ENGINES", buf, 1);
	}
}
