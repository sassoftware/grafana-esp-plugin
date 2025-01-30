#/bin/bash
# This script is a workaround for Snyk's broken directory exclusion feature.
# The Snyk CLI requires initial authenticaiton. See: https://docs.snyk.io/snyk-cli/authenticate-the-cli-with-your-account
set -e

npx snyk auth ${SNYK_API_KEY}

declare -a directoriesToScan=(
	src
    pkg
    internal
)

rm -rf ./include/

for includedDir in ${directoriesToScan[*]}
do
	mkdir -p $(dirname ./include/$includedDir)
	target=$(realpath ../$includedDir)
	ln -s $target ./include/$includedDir
done

cd ./include/ # Snyk can't handle scanning properly unless scanning inside the current directory.
npx snyk code test --json | npx snyk-to-html -o ../report.html
cd ../

rm -rf ./include


cd ..
