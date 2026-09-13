#!/bin/bash
set -e

OS=$(uname)
if [ $OS != 'Darwin' ]; then 
    echo "not running on macos, is this prod?"; 
    exit 1; 
fi

echo "compiling main web app";
echo "compilation time:";
TIMEFORMAT=%R;
time go build -o crbin ../apps/web/main  ;
if [ $? -ne 0 ]; then 
    echo "build failed, check error above";
    exit 1;
fi

echo "compiling dripper app";
echo "compilation time:";
TIMEFORMAT=%R;
time go build -o dripbin ../apps/web/dripper  ;
if [ $? -ne 0 ]; then 
    echo "build failed, check error above";
    exit 1;
fi

echo "compiling rssimport app";
echo "compilation time:";
TIMEFORMAT=%R;
time go build -o import ../apps/rssimport  ;
if [ $? -ne 0 ]; then 
    echo "build failed, check error above";
    exit 1;
fi

set -a
source .env
set +a

export DATABASE_URL=postgres://$DB_USER:$DB_PASS@localhost:$DB_PORT/$DB_NAME?sslmode=disable

echo "running rss import"
./import

echo "starting broswer";
(sleep 1; open http://$MAIN_HOST:$MAIN_HTTP_PORT) &
./dripbin & ./crbin && kill $!

