#!/bin/bash
#
# Copyright (c) 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#

BACKUP_DIR="/var/lib/mysql/data-backup"

# takes backup of mysql
function backup() {
  FILE_PATH=${BACKUP_DIR}/$1
  mysqldump --all-databases --single-transaction --quick --lock-tables=false > ${FILE_PATH} -u root -p${MYSQL_ROOT_PASSWORD}
  if [ $? -eq 0 ]; then
         echo "Mysqldump successful"
         exit 0
    else
        echo "Mysqldump failed"
        exit 1
  fi
}

# Checks if mysql is healthy
# then restores mysql from an existing dump file
function restore() {
  FILE_PATH=${BACKUP_DIR}/$1

  if test -f "${FILE_PATH}"; then
      echo "'${FILE_PATH}' exists."
  else
     echo "'${FILE_PATH}' does not exist"
     exit 1
  fi

  while ! mysqladmin ping -u root -p${MYSQL_ROOT_PASSWORD} --silent; do
          sleep 5
  done
  sleep 10

  mysql -u root -p${MYSQL_ROOT_PASSWORD} < ${FILE_PATH}
  if [ $? -eq 0 ]; then
       echo "Mysql restore successful"
       exit 0
  else
      echo "Mysql restore failed"
      exit  1
  fi
}


mkdir -p ${BACKUP_DIR}

function usage {
    echo
    echo "usage: $0 [-o operation ] [-f filename]"
    echo "  -o operation  The operation to be performed on mysql (backup/restore)"
    echo "  -f filename   The filename of the MYSQL dump file"
    echo "  -h            Help"
    echo
    exit 1
}

while getopts o:f:h flag
do
    case "${flag}" in
        o) OPERATION=${OPTARG};;
        f) MYSQL_DUMP_FILE_NAME=${OPTARG};;
        h) usage;;
        *) usage;;
    esac
done

if [ -z "${OPERATION:-}" ]; then
    echo " Operation cannot be empty !"
    usage
    exit 1
else
  if [ $OPERATION != "backup" ] && [ $OPERATION != "restore" ]; then
    echo "Invalid Operation - $OPERATION. Allowed operation values are backup or restore"
    exit 1
  fi
fi

if [ -z "${MYSQL_DUMP_FILE_NAME:-}"  ]; then
    echo "Dump file name cannot be empty !"
    usage
    exit 1
fi

${OPERATION} ${MYSQL_DUMP_FILE_NAME}
