#!/usr/bin/env python3
import subprocess
import os
import random
import argparse

def main():
    parser = argparse.ArgumentParser(description='Launch nested vm for testing guest')
    parser.add_argument('--vhost-scsi', '-s',
                        action='store_true', default=False,
                        help='Add vhost-scsi device instead of vhost-nvme')
    parser.add_argument('--dry-run', action='store_true', default=False,
                        help='Do not launch only print the command',)

    parser.add_argument('--cgroup', '-c', action='store', default=None,
                        help='Run in a memory cgroup')

    args = parser.parse_args()

    my_pid = os.getpid()
    print(f"my pid is {my_pid}")

    if args.cgroup:
        mem_cgrp_path = os.path.join('/sys/fs/cgroup/memory', args.cgroup)
        if not os.path.exists(mem_cgrp_path):
            os.mkdir(mem_cgrp_path)

        with open(mem_cgrp_path + '/tasks', 'w') as f:
            f.write(f"{my_pid}")

    script_path = os.path.realpath(__file__)
    script_path = os.path.dirname(script_path)

    qemu_cmd = [
        f"{script_path}/qemu/build/qemu-system-x86_64",
         "-nographic",
        "-enable-kvm", "-m", "1024",
        '-cpu', 'host',
        '-smp', '2',
        "-device", "e1000,netdev=net0",
        "-netdev", "user,id=net0,hostfwd=tcp::5551-:22",
        "-append", "console=ttyS0 root=/dev/vda3",
        "-kernel", f"{script_path}/linux/arch/x86_64/boot/bzImage",
        "-drive", f"file={script_path}/alpine.qcow2,if=virtio",
    ]

    if args.vhost_scsi:
        qemu_cmd += ['-device',  'vhost-scsi-pci,wwpn=naa.000000000000000b,bus=pci.0,addr=0x6']
    else:
        qemu_cmd += ['-device',  'vhost-kernel-nvme,bus=pci.0,addr=0x5,serial=deadbeaf']

    print("launching:\n" + subprocess.list2cmdline(qemu_cmd))

    if args.dry_run:
        os.sys.exit(0)

    p = subprocess.Popen(qemu_cmd)
    p.wait()
    print("~~~ Goodbye ~~~")
    
if __name__ == "__main__":
    main()
