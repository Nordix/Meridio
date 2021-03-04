#ifndef MAX_M
#define MAX_M 10000
#define MAX_N 100
#endif

struct MagData {
	unsigned M, N;
	int lookup[MAX_M];
	unsigned permutation[MAX_N][MAX_M];
	unsigned active[MAX_N];
};

void initMagData(struct MagData* d, unsigned m, unsigned n);
void populate(struct MagData* d);
