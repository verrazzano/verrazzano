#!/bin/bash
#
# Copyright (c) 2020, 2022, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
#
BACKUP_DIR="/var/lib/mysql/data-backup"
backup() {
  FILE_PATH=${BACKUP_DIR}/$1
  mysqldump --all-databases --single-transaction --quick --lock-tables=false > ${FILE_PATH} -u root -p${MYSQL_ROOT_PASSWORD}
  if [ $? -eq 0 ]; then
         echo "Mysqldump  successful"
         exit 0
    else
        echo "Mysqldump failed"
        exit 1
  fi
}

restore() {
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
# backup <fileName>
# restore <fileName>
$1 $2
