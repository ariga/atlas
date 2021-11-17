#!/usr/bin/env bash

package="ariga.io/atlas/cmd/atlas"
package_split=(${package//\// })
package_name=${package_split[3]}

platforms=("windows/amd64" "darwin/amd64")
TAG=$(git describe --tags --abbrev=0)


for platform in "${platforms[@]}"
do
    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}
    output_name=$package_name'-'$GOOS'-'$GOARCH'-'$TAG
    if [ $GOOS = "windows" ]; then
        output_name+='.exe'
    fi

    env GOOS=$GOOS GOARCH=$GOARCH go build -o $output_name $package
    if [ $? -ne 0 ]; then
        echo 'An error has occurred! Aborting the script execution...'
        exit 1
    fi
done
