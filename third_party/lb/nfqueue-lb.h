/*
  The shared memory structure for nfqueue-lb.
 */

#include "maglev.h"
#define MEM_VAR "SHM_NAME"
#define MEM_NAME "nfqueue-lb"
struct SharedData {
	int ownFwmark;
	int fwOffset;
	struct MagData magd;
	struct {
		int nActive;
		int lookup[MAX_N];
	} modulo;
};
