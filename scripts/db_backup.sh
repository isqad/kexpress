#!/bin/bash

set -e

backups_path=/work/projects/markeplace-parser/kexpress/db/backups
file_path=${backups_path}/kexpress_$(date +%F).sql.gz

pg_dump -h localhost -p 15432 -U postgres -d kexpress |gzip > ${file_path}
cp ${file_path} ${backups_path}/latest/latest.sql.gz
