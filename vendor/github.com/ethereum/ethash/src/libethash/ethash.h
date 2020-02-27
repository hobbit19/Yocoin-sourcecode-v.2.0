/*
  This file is part of yochash.

  yochash is free software: you can redistribute it and/or modify
  it under the terms of the GNU General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  yochash is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU General Public License for more details.

  You should have received a copy of the GNU General Public License
  along with yochash.  If not, see <http://www.gnu.org/licenses/>.
*/

/** @file yochash.h
* @date 2015
*/
#pragma once

#include <stdint.h>
#include <stdbool.h>
#include <string.h>
#include <stddef.h>
#include "compiler.h"

#define ETHASH_REVISION 23
#define ETHASH_DATASET_BYTES_INIT 1073741824U // 2**30
#define ETHASH_DATASET_BYTES_GROWTH 8388608U  // 2**23
#define ETHASH_CACHE_BYTES_INIT 1073741824U // 2**24
#define ETHASH_CACHE_BYTES_GROWTH 131072U  // 2**17
#define ETHASH_EPOCH_LENGTH 30000U
#define ETHASH_MIX_BYTES 128
#define ETHASH_HASH_BYTES 64
#define ETHASH_DATASET_PARENTS 256
#define ETHASH_CACHE_ROUNDS 3
#define ETHASH_ACCESSES 64
#define ETHASH_DAG_MAGIC_NUM_SIZE 8
#define ETHASH_DAG_MAGIC_NUM 0xFEE1DEADBADDCAFE

#ifdef __cplusplus
extern "C" {
#endif

/// Type of a seedhash/blockhash e.t.c.
typedef struct yochash_h256 { uint8_t b[32]; } yochash_h256_t;

// convenience macro to statically initialize an h256_t
// usage:
// yochash_h256_t a = yochash_h256_static_init(1, 2, 3, ... )
// have to provide all 32 values. If you don't provide all the rest
// will simply be unitialized (not guranteed to be 0)
#define yochash_h256_static_init(...)			\
	{ {__VA_ARGS__} }

struct yochash_light;
typedef struct yochash_light* yochash_light_t;
struct yochash_full;
typedef struct yochash_full* yochash_full_t;
typedef int(*yochash_callback_t)(unsigned);

typedef struct yochash_return_value {
	yochash_h256_t result;
	yochash_h256_t mix_hash;
	bool success;
} yochash_return_value_t;

/**
 * Allocate and initialize a new yochash_light handler
 *
 * @param block_number   The block number for which to create the handler
 * @return               Newly allocated yochash_light handler or NULL in case of
 *                       ERRNOMEM or invalid parameters used for @ref yochash_compute_cache_nodes()
 */
yochash_light_t yochash_light_new(uint64_t block_number);
/**
 * Frees a previously allocated yochash_light handler
 * @param light        The light handler to free
 */
void yochash_light_delete(yochash_light_t light);
/**
 * Calculate the light client data
 *
 * @param light          The light client handler
 * @param header_hash    The header hash to pack into the mix
 * @param nonce          The nonce to pack into the mix
 * @return               an object of yochash_return_value_t holding the return values
 */
yochash_return_value_t yochash_light_compute(
	yochash_light_t light,
	yochash_h256_t const header_hash,
	uint64_t nonce
);

/**
 * Allocate and initialize a new yochash_full handler
 *
 * @param light         The light handler containing the cache.
 * @param callback      A callback function with signature of @ref yochash_callback_t
 *                      It accepts an unsigned with which a progress of DAG calculation
 *                      can be displayed. If all goes well the callback should return 0.
 *                      If a non-zero value is returned then DAG generation will stop.
 *                      Be advised. A progress value of 100 means that DAG creation is
 *                      almost complete and that this function will soon return succesfully.
 *                      It does not mean that the function has already had a succesfull return.
 * @return              Newly allocated yochash_full handler or NULL in case of
 *                      ERRNOMEM or invalid parameters used for @ref yochash_compute_full_data()
 */
yochash_full_t yochash_full_new(yochash_light_t light, yochash_callback_t callback);

/**
 * Frees a previously allocated yochash_full handler
 * @param full    The light handler to free
 */
void yochash_full_delete(yochash_full_t full);
/**
 * Calculate the full client data
 *
 * @param full           The full client handler
 * @param header_hash    The header hash to pack into the mix
 * @param nonce          The nonce to pack into the mix
 * @return               An object of yochash_return_value to hold the return value
 */
yochash_return_value_t yochash_full_compute(
	yochash_full_t full,
	yochash_h256_t const header_hash,
	uint64_t nonce
);
/**
 * Get a pointer to the full DAG data
 */
void const* yochash_full_dag(yochash_full_t full);
/**
 * Get the size of the DAG data
 */
uint64_t yochash_full_dag_size(yochash_full_t full);

/**
 * Calculate the seedhash for a given block number
 */
yochash_h256_t yochash_get_seedhash(uint64_t block_number);

#ifdef __cplusplus
}
#endif
