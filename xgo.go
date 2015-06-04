// Go CGO cross compiler
// Copyright (c) 2014 Péter Szilágyi. All rights reserved.
//
// Released under the MIT license.

// Wrapper around the GCO cross compiler docker container.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// Cross compilation docker containers
var dockerBase = "karalabe/xgo-base"
var dockerDist = "karalabe/xgo-"

// Command line arguments to fine tune the compilation
var goVersion = flag.String("go", "latest", "Go release to use for cross compilation")
var inPackage = flag.String("pkg", "", "Sub-package to build if not root import")
var outPrefix = flag.String("out", "", "Prefix to use for output naming (empty = package name)")
var srcRemote = flag.String("remote", "", "Version control remote repository to build")
var srcBranch = flag.String("branch", "", "Version control branch to build")
var crossDeps = flag.String("deps", "", "CGO dependencies (configure/make based archives)")
var targets   = flag.String("targets", "all", "Specify a comma separated list of targets: linux-amd64,linux-386 linux-arm")



// Command line arguments to pass to go build
var buildVerbose = flag.Bool("v", false, "Print the names of packages as they are compiled")
var buildRace = flag.Bool("race", false, "Enable data race detection (supported only on amd64)")

func main() {
	flag.Parse()

	// Ensure docker is available
	if err := checkDocker(); err != nil {
		log.Fatalf("Failed to check docker installation: %v.", err)
	}
	// Validate the command line arguments
	if len(flag.Args()) != 1 {
		log.Fatalf("Usage: %s [options] <go import path>", os.Args[0])
	}
	// Check that all required images are available
	found, err := checkDockerImage(dockerDist + *goVersion)
	switch {
	case err != nil:
		log.Fatalf("Failed to check docker image availability: %v.", err)
	case !found:
		fmt.Println("not found!")
		if err := pullDockerImage(dockerDist + *goVersion); err != nil {
			log.Fatalf("Failed to pull docker image from the registry: %v.", err)
		}
	default:
		fmt.Println("found.")
	}
	// Cross compile the requested package into the local folder
	if err := compile(flag.Args()[0], *srcRemote, *srcBranch, *inPackage, *targets, *crossDeps, *outPrefix, *buildVerbose, *buildRace); err != nil {
		log.Fatalf("Failed to cross compile package: %v.", err)
	}
}

// Checks whether a docker installation can be found and is functional.
func checkDocker() error {
	fmt.Println("Checking docker installation...")
	if err := run(exec.Command("docker", "version")); err != nil {
		return err
	}
	fmt.Println()
	return nil
}

// Checks whether a required docker image is available locally.
func checkDockerImage(image string) (bool, error) {
	fmt.Printf("Checking for required docker image %s... ", image)
	out, err := exec.Command("docker", "images", "--no-trunc").Output()
	if err != nil {
		return false, err
	}
	return bytes.Contains(out, []byte(image)), nil
}

// Pulls an image from the docker registry.
func pullDockerImage(image string) error {
	fmt.Printf("Pulling %s from docker registry...\n", image)
	return run(exec.Command("docker", "pull", image))
}

// Checks if a string is in the array
func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// Check which targets to compile for
func getTargets(targets string) (linux64 string, linux386 string, linuxArm string, windows64 string, windows386 string, darwin64 string, darwin386 string) {

	// Targets
	linux64 = "false"
	linux386 = "false"
	linuxArm = "false"
	windows64 = "false"
	windows386 = "false"
	darwin64 = "false"
	darwin386 = "false"

	if targets != "all" {

		if stringInSlice("linux64", strings.Split(targets, ",")) {
			linux64 = "true"
		}
		if stringInSlice("linux386", strings.Split(targets, ",")) {
			linux386 = "true"
		}
		if stringInSlice("linuxArm", strings.Split(targets, ",")) {
			linuxArm = "true"
		}
		if stringInSlice("windows64", strings.Split(targets, ",")) {
			windows64 = "true"
		}
		if stringInSlice("windows386", strings.Split(targets, ",")) {
			windows386 = "true"
		}
		if stringInSlice("darwin64", strings.Split(targets, ",")) {
			darwin64 = "true"
		}
		if stringInSlice("darwin386", strings.Split(targets, ",")) {
			darwin386 = "true"
		}
	} else {
		fmt.Printf("Building for all arch")
		linux64 = "true"
		linux386 = "true"
		linuxArm = "true"
		windows64 = "true"
		windows386 = "true"
		darwin64 = "true"
		darwin386 = "true"
	}
	return linux64, linux386, linuxArm, windows64, windows386, darwin64, darwin386
}

// Cross compiles a requested package into the current working directory.
func compile(repo string, remote string, branch string, pack string, targets string, deps string, prefix string, verbose bool, race bool) error {
	folder, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to retrieve the working directory: %v.", err)
	}

	linux64, linux386, linuxArm, windows64, windows386, darwin64, darwin386 := getTargets(targets)

	fmt.Printf("Cross compiling %s...\n", repo)
	return run(exec.Command("docker", "run",
		"-v", folder+":/build",
		"-e", "REPO_REMOTE="+remote,
		"-e", "REPO_BRANCH="+branch,
		"-e", "PACK="+pack,
		"-e", "LINUX64="+linux64,
		"-e", "LINUX386="+linux386,
		"-e", "LINUXARM="+linuxArm,
		"-e", "WINDOWS64="+windows64,
		"-e", "WINDOWS386="+windows386,
		"-e", "DARWIN64="+darwin64,
		"-e", "DARWIN386=%s"+darwin386,
		"-e", "DEPS="+deps,
		"-e", "OUT="+prefix,
		"-e", fmt.Sprintf("FLAG_V=%v", verbose),
		"-e", fmt.Sprintf("FLAG_RACE=%v", race),
		dockerDist+*goVersion, repo))
}

// Executes a command synchronously, redirecting its output to stdout.
func run(cmd *exec.Cmd) error {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
