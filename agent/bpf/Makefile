CLANG := clang
CFLAGS := -g -O2 -target bpf -D__TARGET_ARCH_arm64

all: profiler.bpf.o monitor.bpf.o 

monitor.bpf.o: monitor.bpf.c ebpf_structures.h vmlinux.h
	$(CLANG) $(CFLAGS) \
		-I. \
		-I/usr/include/bpf \
		-c monitor.bpf.c \
		-o monitor.bpf.o
profiler.bpf.o: profiler.bpf.c ebpf_structures.h vmlinux.h
	$(CLANG) $(CFLAGS)\
		-I. \
		-I/usr/include/bpf \
		-c profiler.bpf.c \
		-o profiler.bpf.o

clean:
	rm -f *.bpf.o
