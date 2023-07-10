#!/bin/sh

# make instlal
make
make install

#copy env
env >> /etc/profile

#setup cron
echo "$SYNC_CRON /app/scripts/run.sh 2>&1 | tee -a /var/log/gogo-repo-stats.log" > /etc/cron.d/crontab
chmod 0644 /etc/cron.d/crontab
crontab /etc/cron.d/crontab
touch /var/log/cron.log

# start cron
/usr/sbin/crond -b
touch /var/log/gogo-repo-stats.log
tail -f /var/log/gogo-repo-stats.log