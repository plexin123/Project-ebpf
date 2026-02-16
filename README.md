# Project-ebpf

# Steps to run the program step 1:
# go to  cmd
# then run go run main.go

# Structure of the repository:
# agent: basically contains the code that will be attached on the kernel:
# - monitor.c: the ebgf program 
# - main.go : this will load the ebpf program and send to my user space 
