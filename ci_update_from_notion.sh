#!/bin/bash
set -u -e -o pipefail

# setup git
git config --global user.email "kkowalczyk@gmail.com"
git config --global user.name "Krzysztof Kowalczyk"
git config --global github.user "kjk"
git config --global github.token "${GITHUB_TOKEN}"

# redownload latest versions from notion and checkin changes
# this in turn will trigger deploy on push from ci_netlify_deploy.sh
rm -rf netlify*
git checkout master

./blog -redownload-notion

git status
git checkout netlify.toml
git status
git add notion_cache/*
git status
now=`date "+%Y-%m-%d %a"`

# "git commit" returns 1 if there's nothing to commit, so don't report this as failed build
set +e
git commit -am "ci: update from notion on ${now}"
if [ "$?" -ne "0" ]; then
    echo "nothing to commit"
    exit 0
fi
set -e
git push "https://${GITHUB_TOKEN}@github.com/kjk/blog.git" master || true
