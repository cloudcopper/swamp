#!/bin/sh
set -ex
[ -z $1 ] && exit 1

cd $1

dd if=/dev/urandom of=file1.bin bs=1k count=128
date +%s > _createdAt.txt
export > _export.txt
sha256sum file1.bin _createdAt.txt _export.txt > _checksum.txt
mv _checksum.txt $(sha256sum _checksum.txt|cut -d' ' -f1).sha256sum

