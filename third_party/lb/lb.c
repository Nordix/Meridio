#include <errno.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <string.h>
#include <time.h>
#include <arpa/inet.h>
#include <linux/types.h>

static int cmdCreate(int argc, char* argv[]);
static int cmdShow(int argc, char* argv[]);
static int cmdClean(int argc, char* argv[]);
static int cmdActivate(int argc, char* argv[]);
static int cmdDeactivate(int argc, char* argv[]);
static int cmdRun(int argc, char* argv[]);

int main(int argc, char *argv[])
{
	static struct Cmd {
		char const* const name;
		int (*fn)(int argc, char* argv[]);
	} cmd[] = {
		{"create", cmdCreate},
		{"show", cmdShow},
		{"clean", cmdClean},
		{"activate", cmdActivate},
		{"deactivate", cmdDeactivate},
		{"run", cmdRun},
		{NULL, NULL}
	};

	if (argc < 2) {
		printf("Usage: %s <command> [opt...]\n", argv[0]);
		exit(EXIT_FAILURE);
	}

	argc--;
	argv++;
	for (struct Cmd* c = cmd; c->fn != NULL; c++) {
		if (strcmp(*argv, c->name) == 0)
			return c->fn(argc, argv);
	}

	return 0;
}

/* ---------------------------------------------------------------------- */
#include "nfqueue-lb.h"
#include <sys/mman.h>
#include <fcntl.h>

static void die(char const* msg)
{
	perror(msg);
	exit(EXIT_FAILURE);
}

static char const* memName(void)
{
	char const* name = getenv(MEM_VAR);
	if (name == NULL) return MEM_NAME;
	return name;
}
static void createSharedData(struct SharedData* sh)
{
	int fd = shm_open(memName(), O_RDWR|O_CREAT, 0600);
	if (fd < 0) die("shm_open");
	write(fd, sh, sizeof(*sh));
	close(fd);
}

static struct SharedData* mapSharedData(int mode)
{
	int fd = shm_open(memName(), mode, (mode == O_RDONLY)?0400:0600);
	if (fd < 0) die("shm_open");
	struct SharedData* m = mmap(
		NULL, sizeof(struct SharedData),
		(mode == O_RDONLY)?PROT_READ:PROT_READ|PROT_WRITE, MAP_SHARED, fd, 0);
	if (m == MAP_FAILED) die("mmap");
	return m;
}

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

static void updateModulo(struct SharedData* sh)
{
	sh->modulo.nActive = 0;
	for (int i = 0; i < MAX_N; i++) {
		if (sh->magd.active[i]) {
			sh->modulo.lookup[sh->modulo.nActive] = i;
			sh->modulo.nActive++;
		}
	}
}

static int cmdCreate(int argc, char* argv[])
{
	static char const* const opt = "i:o:";
	int c, i=-1, fwOffset=1;
	for (c = getopt(argc, argv, opt); c > 0; c = getopt(argc, argv, opt)) {
		switch (c) {
		case 'i':
			i = atoi(optarg);
			break;
		case 'o':
			fwOffset = atoi(optarg);
			break;
		default:
			fprintf(stderr, "Unknown option [%c]", c);
			exit(EXIT_FAILURE);
		}
	}
	argc -= optind;
	argv += optind;

	struct SharedData sh;
	sh.ownFwmark = i;
	sh.fwOffset = fwOffset;
	unsigned M=997, N=10;
	if (argc > 0) {
		M = atoi(argv[0]);
		if (M < 20) M = 19;
		if (M > MAX_M) M = MAX_M;
		M = primeBelow(M);
	}
	if (argc > 1) {
		N = atoi(argv[1]);
		if (N < 4) N = 4;
		if (N > MAX_N) N = MAX_N;
	}
	initMagData(&sh.magd, M, N);
	for (int i = 0; i < 4; i++)
	   sh.magd.active[i] = 1;
	populate(&sh.magd);
	updateModulo(&sh);

	createSharedData(&sh);
	return 0;
}
static int cmdShow(int argc, char* argv[])
{
	struct SharedData* sh = mapSharedData(O_RDONLY);
	struct MagData* m = &sh->magd;
	printf("Own fwmark: %d\n", sh->ownFwmark);
	printf("Fwmark offset: %d\n", sh->fwOffset);
	printf("=== Maglev hashing;\n");
	printf("M=%u, N=%u\n", m->M, m->N);
	printf("Active;\n");
	for (int i = 0; i < m->N; i++)
		printf(" %u", m->active[i]);
	puts("");
	printf("Lookup;\n");
	for (int i = 0; i < 25; i++)
		printf(" %d", m->lookup[i]);
	puts(" ...");
	printf("=== Modulo hashing;\n");
	printf("nActive=%d\n", sh->modulo.nActive);
	printf("Lookup;\n");
	for (int i = 0; i < sh->modulo.nActive; i++)
		printf(" %d", sh->modulo.lookup[i]);
	puts("");
	return 0;
}
static int cmdClean(int argc, char* argv[])
{
	if (shm_unlink(memName()) != 0) die("shm_unlink");
	return 0;
}

static void setActivate(unsigned v, int argc, char *argv[])
{
	struct SharedData* sh = mapSharedData(O_RDWR);
	struct MagData* m = &sh->magd;
	argc--;
	argv++;
	while (argc-- > 0) {
		int i = atoi(*argv++) - sh->fwOffset;
		if (i >= 0 && i < m->N) m->active[i] = v;
	}
	populate(m);
	updateModulo(sh);
}
static int cmdActivate(int argc, char* argv[])
{
	setActivate(1, argc, argv);
	return 0;
}
static int cmdDeactivate(int argc, char* argv[])
{
	setActivate(0, argc, argv);
	return 0;
}

/* ----------------------------------------------------------------------
   Packet handling. The outcome is a fwmark.
*/
#include <netinet/if_ether.h>
#include <netinet/ip.h>
#include <netinet/ip6.h>
#include <netinet/ip_icmp.h>

/* These variables are set in cmdRun() */
static uint32_t (*get_mark)(uint32_t hash);
static struct SharedData* shData;
static unsigned portlen = 0;

static uint32_t get_maglev_mark(uint32_t hash)
{
	return shData->magd.lookup[hash % shData->magd.M] + shData->fwOffset;
}

static uint32_t get_modulo_mark(uint32_t hash)
{
	return shData->modulo.lookup[hash % shData->modulo.nActive] + shData->fwOffset;
}

static uint32_t
djb2_hash(uint8_t const* c, uint32_t len)
{
	uint32_t hash = 5381;
	while (len--)
		hash = ((hash << 5) + hash) + *c++; /* hash * 33 + c */
	return hash;
}

/*
  return: fwmark
 */
static uint32_t handlePacket(
	uint16_t protocol, uint8_t const* payload, uint16_t plen)
{
#if 0
	printf("packet received hw=0x%04x payload len %u\n", protocol, plen);
#endif
	/*
	  Addresses;
	  ipv4; payload[12], len 8
	  ipv6; payload[8], len 32
	  TODO: Handle ipv6 next-header and ipv4 options.
	  TODO; For ICMP "packet too big" amd others compute hash for
	  the "inner" address.
	 */
	switch (protocol) {
	case ETH_P_IP: {
		struct iphdr const* hdr = (struct iphdr const*)payload;
		if (hdr->ihl > 5) return 0; // Can't handle options
		uint16_t frag = ntohs(hdr->frag_off);
		if ((frag & (IP_OFFMASK|IP_MF)) != 0) return 0; // Can't handle fragments
		if (hdr->protocol == IPPROTO_TCP) {
			return get_mark(djb2_hash(payload + 12, 8 + portlen));
		} else if (hdr->protocol == IPPROTO_ICMP) {
			struct icmphdr const* hdr = (struct icmphdr const*)(payload+20);
			if (hdr->type == ICMP_DEST_UNREACH) {
				// Get the inner headed switch src<->dst and hash
				return 0;		/* NYI */
			}
		}
	}
	case ETH_P_IPV6: {
		struct ip6_hdr const* hdr = (struct ip6_hdr const*)payload;
		if (hdr->ip6_nxt == IPPROTO_TCP) {
			return get_mark(djb2_hash(payload + 8, 32 + portlen));
		} else if (hdr->ip6_nxt == IPPROTO_ICMP) {
			struct icmphdr const* hdr = (struct icmphdr const*)(payload+20);
			if (hdr->type == ICMP_DEST_UNREACH) {
				// Get the inner headed switch src<->dst and hash
				return 0;		/* NYI */
			}
		}
	}
	default:;
	}
	
	return 0;
}

/* ----------------------------------------------------------------------
   The NFQUEUE code is taken from the example in;
   libnetfilter_queue-1.0.3/examples/nf-queue.c
*/

#include <libmnl/libmnl.h>
#include <linux/netfilter.h>
#include <linux/netfilter/nfnetlink.h>
#include <linux/netfilter/nfnetlink_queue.h>
#include <libnetfilter_queue/libnetfilter_queue.h>
/* only for NFQA_CT, not needed otherwise: */
#include <linux/netfilter/nfnetlink_conntrack.h>


static struct mnl_socket *nl;

static struct nlmsghdr *
nfq_hdr_put(char *buf, int type, uint32_t queue_num)
{
	struct nlmsghdr *nlh = mnl_nlmsg_put_header(buf);
	nlh->nlmsg_type	= (NFNL_SUBSYS_QUEUE << 8) | type;
	nlh->nlmsg_flags = NLM_F_REQUEST;

	struct nfgenmsg *nfg = mnl_nlmsg_put_extra_header(nlh, sizeof(*nfg));
	nfg->nfgen_family = AF_UNSPEC;
	nfg->version = NFNETLINK_V0;
	nfg->res_id = htons(queue_num);

	return nlh;
}

static void
nfq_send_verdict(int queue_num, uint32_t id, uint32_t mark)
{
	char buf[MNL_SOCKET_BUFFER_SIZE];
	struct nlmsghdr *nlh;
	struct nlattr *nest;

	nlh = nfq_hdr_put(buf, NFQNL_MSG_VERDICT, queue_num);
	nfq_nlmsg_verdict_put(nlh, id, NF_ACCEPT);
	nfq_nlmsg_verdict_put_mark(nlh, mark);

	/* example to set the connmark. First, start NFQA_CT section: */
	nest = mnl_attr_nest_start(nlh, NFQA_CT);

	/* then, add the connmark attribute: */
	mnl_attr_put_u32(nlh, CTA_MARK, htonl(42));
	/* more conntrack attributes, e.g. CTA_LABEL, could be set here */

	/* end conntrack section */
	mnl_attr_nest_end(nlh, nest);

	if (mnl_socket_sendto(nl, nlh, nlh->nlmsg_len) < 0) {
		perror("mnl_socket_send");
		exit(EXIT_FAILURE);
	}
}

static int queue_cb(const struct nlmsghdr *nlh, void *data)
{
	struct nfqnl_msg_packet_hdr *ph = NULL;
	struct nlattr *attr[NFQA_MAX+1] = {};
	uint32_t id = 0;
	struct nfgenmsg *nfg;
	uint16_t plen;

	if (nfq_nlmsg_parse(nlh, attr) < 0) {
		perror("problems parsing");
		return MNL_CB_ERROR;
	}

	nfg = mnl_nlmsg_get_payload(nlh);

	if (attr[NFQA_PACKET_HDR] == NULL) {
		fputs("metaheader not set\n", stderr);
		return MNL_CB_ERROR;
	}

	ph = mnl_attr_get_payload(attr[NFQA_PACKET_HDR]);

	plen = mnl_attr_get_payload_len(attr[NFQA_PAYLOAD]);
	id = ntohl(ph->packet_id);

	uint8_t *payload = mnl_attr_get_payload(attr[NFQA_PAYLOAD]);
	nfq_send_verdict(
		ntohs(nfg->res_id), id, handlePacket(ntohs(ph->hw_protocol), payload, plen));

	return MNL_CB_OK;
}

static int cmdRun(int argc, char* argv[])
{
	char *buf;
	/* largest possible packet payload, plus netlink data overhead: */
	size_t sizeof_buf = 0xffff + (MNL_SOCKET_BUFFER_SIZE/2);
	struct nlmsghdr *nlh;
	int ret;
	unsigned int portid, queue_num = 2;
	char const* mode = "maglev";

	static char const* const opt = "q:pm:h";
	int c;
	for (c = getopt(argc, argv, opt); c > 0; c = getopt(argc, argv, opt)) {
		switch (c) {
		case 'q':
			queue_num = atoi(optarg);
			break;
		case 'p':
			portlen = 4;
			break;
		case 'm':
			mode = optarg;
			break;
		default:
			fprintf(stderr, "Unknown option [%c]", c);
			exit(EXIT_FAILURE);
		}
	}
	
	int fd = shm_open(memName(), O_RDONLY, 0400);
	if (fd < 0) die("shm_open");
	shData = mmap(
		NULL, sizeof(struct MagData), PROT_READ, MAP_SHARED, fd, 0);
	if (shData == MAP_FAILED) die("mmap");

	if (strcmp(mode, "modulo") == 0) {
		get_mark = get_modulo_mark;
	} else {
		get_mark = get_maglev_mark;
	}

	nl = mnl_socket_open(NETLINK_NETFILTER);
	if (nl == NULL) {
		perror("mnl_socket_open");
		exit(EXIT_FAILURE);
	}

	if (mnl_socket_bind(nl, 0, MNL_SOCKET_AUTOPID) < 0) {
		perror("mnl_socket_bind");
		exit(EXIT_FAILURE);
	}
	portid = mnl_socket_get_portid(nl);

	buf = malloc(sizeof_buf);
	if (!buf) {
		perror("allocate receive buffer");
		exit(EXIT_FAILURE);
	}

	/* PF_(UN)BIND is not needed with kernels 3.8 and later */
	nlh = nfq_hdr_put(buf, NFQNL_MSG_CONFIG, 0);
	nfq_nlmsg_cfg_put_cmd(nlh, AF_INET, NFQNL_CFG_CMD_PF_UNBIND);

	if (mnl_socket_sendto(nl, nlh, nlh->nlmsg_len) < 0) {
		perror("mnl_socket_send");
		exit(EXIT_FAILURE);
	}

	nlh = nfq_hdr_put(buf, NFQNL_MSG_CONFIG, 0);
	nfq_nlmsg_cfg_put_cmd(nlh, AF_INET, NFQNL_CFG_CMD_PF_BIND);

	if (mnl_socket_sendto(nl, nlh, nlh->nlmsg_len) < 0) {
		perror("mnl_socket_send");
		exit(EXIT_FAILURE);
	}

	nlh = nfq_hdr_put(buf, NFQNL_MSG_CONFIG, queue_num);
	nfq_nlmsg_cfg_put_cmd(nlh, AF_INET, NFQNL_CFG_CMD_BIND);

	if (mnl_socket_sendto(nl, nlh, nlh->nlmsg_len) < 0) {
		perror("mnl_socket_send");
		exit(EXIT_FAILURE);
	}

	nlh = nfq_hdr_put(buf, NFQNL_MSG_CONFIG, queue_num);
	nfq_nlmsg_cfg_put_params(nlh, NFQNL_COPY_PACKET, 0xffff);

	mnl_attr_put_u32(nlh, NFQA_CFG_FLAGS, htonl(NFQA_CFG_F_GSO));
	mnl_attr_put_u32(nlh, NFQA_CFG_MASK, htonl(NFQA_CFG_F_GSO));

	if (mnl_socket_sendto(nl, nlh, nlh->nlmsg_len) < 0) {
		perror("mnl_socket_send");
		exit(EXIT_FAILURE);
	}

	/* ENOBUFS is signalled to userspace when packets were lost
	 * on kernel side.  In most cases, userspace isn't interested
	 * in this information, so turn it off.
	 */
	ret = 1;
	mnl_socket_setsockopt(nl, NETLINK_NO_ENOBUFS, &ret, sizeof(int));

	for (;;) {
		ret = mnl_socket_recvfrom(nl, buf, sizeof_buf);
		if (ret == -1) {
			perror("mnl_socket_recvfrom");
			exit(EXIT_FAILURE);
		}

		ret = mnl_cb_run(buf, ret, 0, portid, queue_cb, NULL);
		if (ret < 0){
			perror("mnl_cb_run");
			exit(EXIT_FAILURE);
		}
	}

	mnl_socket_close(nl);
	return 0;
}

