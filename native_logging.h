#ifndef HIGHS_GO_LOGGING_H
#define HIGHS_GO_LOGGING_H

#include <stdint.h>
#include <stdlib.h>
#include "lp_data/HighsCallbackStruct.h"

void highs_go_log(int type, const char* message, const HighsCallbackDataOut* out,
                  HighsCallbackDataIn* in, void* user_data);

#endif
