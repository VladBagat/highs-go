#include "native_logging.h"
#include "_cgo_export.h"

void highs_go_log(int type, const char* message,
                         const HighsCallbackDataOut* out,
                         HighsCallbackDataIn* in, void* user_data) {
    (void)in;
    (void)type; /* Only the logging callback is activated. */
    goHighsLog(*(uintptr_t*)user_data, out->log_type, (char*)message);
}
