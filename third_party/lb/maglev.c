// gcc -DSATEST -o /tmp/maglev src/maglev.c
// /tmp/maglev 1000 10 2 2

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "maglev.h"

void initMagData(struct MagData* d, unsigned m, unsigned n)
{
	memset(d, 0, sizeof(*d));
	d->M = m;
	d->N = n;
	for (int i = 0; i < d->N; i++) {
		unsigned offset = rand() % d->M;
		unsigned skip = rand() % (d->M - 1) + 1;
		unsigned j;
		for (j = 0; j < d->M; j++) {
			d->permutation[i][j] = (offset + j * skip) % d->M;
		}
	}
}

void populate(struct MagData* d)
{
	for (int i = 0; i < d->M; i++) {
		d->lookup[i] = -1;
	}

	// Corner case; no active targets
	unsigned nActive = 0;
	for (int i = 0; i < d->N; i++) {
		if (d->active[i] != 0) nActive++;
	}
	if (nActive == 0) return;
	
	unsigned next[MAX_N], c = 0;
	memset(next, 0, sizeof(next));
	unsigned n = 0;
	for (;;) {
		for (int i = 0; i < d->N; i++) {
			if (d->active[i] == 0) continue; /* Target not active */
			c = d->permutation[i][next[i]];
			while (d->lookup[c] >= 0) {
				next[i] = next[i] + 1;
				c = d->permutation[i][next[i]];
			}
			d->lookup[c] = i;
			next[i] = next[i] + 1;
			n = n + 1;
			if (n == d->M) return;
		}
	}
}




/* ------- Stand-Alone Test ---------------------------------------- */
#ifdef SATEST

// Prime numbers < 100
static unsigned primes100[25] = {
	2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47, 53, 59, 61, 67, 71, 73, 79, 83, 89, 97
};

static int isPrime(unsigned n)
{
	for (int i = 0; i < 25; i++) {
		if (n <= primes100[i]) return 1;
		if ((n % primes100[i]) == 0) return 0;
	}
	return 1;
}

static unsigned primeBelow(unsigned n)
{
	if (isPrime(n)) return n;
	if (n % 2 == 0) n--;
	while (n > 1) {
		if (isPrime(n)) break;
		n -= 2;
	}
	return n;
}

static unsigned seed;

static void initExample(struct MagData* d)
{
	printf(
		"Use the example from page 6 in;\n"
		"https://static.googleusercontent.com/media/research.google.com/en//pubs/archive/44824.pdf\n");

	memset(d, 0, sizeof(*d));
	d->M = 7;
	d->N = 3;
	for (int i = 0; i < d->N; i++) {
		unsigned offset;
		unsigned skip;
		switch (i) {
		case 0:
			offset = 3;
			skip = 4;
			break;
		case 1:
			offset = 0;
			skip = 2;
			break;
		case 2:
			offset = 3;
			skip = 1;
			break;
		}
		for (unsigned j = 0; j < d->M; j++) {
			d->permutation[i][j] = (offset + j * skip) % d->M;
		}
		d->active[i] = 1;
	}
}


static void printPermutations(struct MagData* d)
{
	printf("Permutations;\n");
	unsigned i, j;
	for (i = 0; i < d->N; i++) {
		for (j = 0; j < d->M; j++) {
			printf(" %02d", d->permutation[i][j]);
		}
		puts("");
	}
}
static void printLookup(struct MagData* d)
{
	printf("Active;\n");
	for (unsigned i = 0; i < d->N; i++) {
		printf(" %u", d->active[i]);
	}
	puts("");
	printf("Lookup;\n");
	for (unsigned i = 0; i < d->M; i++) {
		printf(" %d", d->lookup[i]);
	}
	puts("");
}



static void loopTest(struct MagData* d, int n)
{
	int lookup[MAX_M], diff, sum = 0, i;
	for (i = 0; i < n; i++) {
		srand(i + seed);
		for (int i = 0; i < d->N; i++) {
			unsigned offset = rand() % d->M;
			unsigned skip = rand() % (d->M - 1) + 1;
			unsigned j;
			for (j = 0; j < d->M; j++) {
				d->permutation[i][j] = (offset + j * skip) % d->M;
			}
		}
		d->active[0] = 1;
		populate(d);
		
		memcpy(lookup, d->lookup, sizeof(lookup));
		d->active[0] = 0;
		populate(d);

		diff = 0;
		for (int i = 0; i < d->M; i++) {
			if (lookup[i] != d->lookup[i]) diff++;
		}
		sum += diff;
		printf("diff %d, %d%%\n", diff, (diff + d->M/200) * 100 / d->M);
	}

	printf("Avg: %d%%\n", (sum + (n * d->M)/200) * 100 / (n * d->M));
}

int
main(int argc, char* argv[])
{
	struct MagData env;
	if (argc == 1) {
		// Show the example from p6 in the maglev doc
		initExample(&env);
		printPermutations(&env);
		populate(&env);
		printLookup(&env);
		env.active[1] = 0;
		populate(&env);
		printLookup(&env);
	}

	if (argc < 4) {
		printf("Syntax; maglev M N seed [loops]\n");
		return 0;
	}

	unsigned M=primeBelow(atoi(argv[1]));
	if (M > MAX_M) {
		printf("Error; M > %u\n", MAX_M);
		return 1;
	}
	unsigned N=atoi(argv[2]);
	if (N > MAX_N) {
		printf("Error; N > %u\n", MAX_N);
		return 1;
	}
	
	seed = atoi(argv[3]);
	srand(seed);
	initMagData(&env, M, N);
	printf("M=%u, N=%u\n", env.M, env.N);

	for (unsigned i = 0; i < env.N; i++) {
		env.active[i] = 1;
	}

	if (argc > 4) {
		loopTest(&env, atoi(argv[4]));
		return 0;
	}
	
	printPermutations(&env);

	populate(&env);
	printLookup(&env);

	env.active[0] = 0;
	populate(&env);
	printLookup(&env);

	env.active[0] = 1;
	populate(&env);
	printLookup(&env);

	return 0;
}

#endif
