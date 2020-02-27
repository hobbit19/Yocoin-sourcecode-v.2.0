#pragma once

#include "compiler.h"
#include <stdint.h>
#include <stdlib.h>

#ifdef __cplusplus
extern "C" {
#endif

struct yochash_h256;

void SHA3_256(struct yochash_h256 const* ret, uint8_t const* data, size_t size);
void SHA3_512(uint8_t* const ret, uint8_t const* data, size_t size);

#ifdef __cplusplus
}
#endif
