#!/bin/bash
set -e

if ./scripts/is_branch_head.bash master ; then
  export JENKINS_SHOULD_RELEASE=ismaster
else
  echo "Not on master branch head, not releasing."
  exit 0
fi

set -x
git remote -v
git fetch origin --tags
git branch -D master || true
git checkout -b master origin/master

npm run semantic-release || true
