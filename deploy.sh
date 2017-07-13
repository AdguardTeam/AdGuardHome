#!/bin/bash
#if [ "$TRAVIS_COMMIT_MESSAGE" == "Auto update filters" ]
#then
#	exit 0
#fi

python Filters/parser.py

if [[ $? = 0 ]]; then
	git config --global user.name "Travis CI"
	git config --global user.email "travis@travis-ci.org"
	git config --global push.default simple

	git remote add deploy "https://$GITHUB_TOKEN@github.com/AdguardTeam/AdguardDNS.git"
	git fetch deploy

	git commit -a -m "Auto update filters"
	git push -q deploy HEAD:master
else 
	echo "failure: $?"
	exit 1
fi 
