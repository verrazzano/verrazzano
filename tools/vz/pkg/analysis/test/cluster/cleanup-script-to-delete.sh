#!/bin/bash
# gitlab
# jenkins
# IP addresses
# oraclecorp
# oracledx
function sanitize_file() {
  echo "Santize $1"
  sed -E \
     -e "s;confluence.oraclecorp\.com/confluence;REDACTED-HOSTNAME;g" \
     -e "s/gitlab-odx\.oracledx\.com/REDACTED-HOSTNAME/g" \
     -e "s/verrazzano.*\.oracledx\.com/REDACTED-HOSTNAME/g" \
     -e "s/[a-zA-Z0-9]*\.us\.oracle\.com/REDACTED-HOSTNAME/g" \
     -e "s/[[:digit:]]{1,3}\.[[:digit:]]{1,3}\.[[:digit:]]{1,3}\.[[:digit:]]{1,3}/REDACTED-IP4-ADDRESS/g" \
     -e "s/[a-zA-Z0-9]*\.oraclecorp\.com/REDACTED-HOSTNAME/g" \
     $1 > $1_tmpfoo
  diff $1 $1_tmpfoo
  if [ $? == 0 ]; then
    rm $1_tmpfoo
  else
    mv $1_tmpfoo $1
  fi
}
for file in $(find $1 -name "*")
do
  if [ -f $file ]; then
    sanitize_file $file
  fi
done
