#!/bin/bash

set -e

outdir=build/_outdir
releasedir=build/_releasedir

if [ ! -f "build/releasebuild.sh" ]; then
	echo "$0 must be run from the root of the repository."
	exit 2
fi

if [ "$1" == "clean" ]; then
	rm -rf $outdir $releasedir
	exit
fi

mkdir -p $outdir $releasedir

function gen_arc_name() {
	os=$1
	arch=$2

	ext='tar.gz'
	if [ "$os" == "win" ]; then
		ext='zip'
	fi

	echo $(gen_out_dir_name)-$os-$arch.$ext
}

function gen_out_dir_name() {
	vermarker=$(git rev-parse --short HEAD)

	gittag=$(git describe --tags --exact-match HEAD 2> /dev/null)
	if [ "$?" == "0" ]; then
		vermarker=$(echo "$gittag" | head -n 1)
	fi

	echo lit-$vermarker
}

function get_work_dir_path() {
	os=$1
	arch=$2
	echo $outdir/$os-$arch
}

function run_build_for_platform() {
	os=$1
	arch=$2

	# Set up the output directory.
	workdir=$(get_work_dir_path $os $arch)
	gooutdir=$workdir/$(gen_out_dir_name)
	mkdir -p $gooutdir

	# Figure out some other details.
	binext=''
	goos=$os
	goarch=$arch

	if [ "$os" == "win" ]; then
		goos='windows'
		binext='.exe'
	fi

	if [ "$arch" == "i386" ]; then
		goarch='386'
	fi

	# Actually build the binaries.
	env GOOS=$goos GOARCH=$goarch GO_BUILD_EX_ARGS="-o $gooutdir/lit$binext" make lit
	env GOOS=$goos GOARCH=$goarch GO_BUILD_EX_ARGS="-o $gooutdir/lit-af$binext" make lit-af

}

function compile_and_package() {
	os=$1
	arch=$2

	outdirname=$(gen_out_dir_name)
	arcname=$(gen_arc_name $os $arch)

	run_build_for_platform $os $arch

	# Copy some other files into what the archive's going to copy up.
	for f in 'README.md LICENSE litlogo145.png'; do
		cp -r $f $workdir/$outdirname
	done

	# This is where we actually make the archive of it.
	set +e
	workdir=$(get_work_dir_path $os $arch)
	pushd $workdir
	if [ "$os" != "win" ]; then
		tar -cvzf $arcname $outdirname
	else
		zip -r $arcname $outdirname
	fi
	popd
	set -e

	# And them move it into the release directory.
	if [ -e "$workdir/$arcname" ]; then
		mv $workdir/$arcname $releasedir
	fi

}

if [ -n "$1" ] && [ -n "$2" ]; then
	compile_and_package $1 $2
else

	if [ -z "$1" ]; then

		# Linux
		compile_and_package linux amd64
		compile_and_package linux i386
		compile_and_package linux arm

		# macOS (Darwin)
		compile_and_package darwin amd64

		# Windows
		compile_and_package win amd64
		compile_and_package win i386

	else
		echo "usage: $0 <os> <arch>"
		exit 1
	fi

fi
