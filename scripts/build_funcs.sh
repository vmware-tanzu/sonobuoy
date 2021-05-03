#!/usr/bin/env bash


# Helpful debug info while script runs.
PS4='\033[1m[$0:${LINENO}] $\033[0m '

BINARY=sonobuoy
TARGET=sonobuoy
GOTARGET=github.com/vmware-tanzu/"$TARGET"
GOPATH=$(go env GOPATH)
REGISTRY=sonobuoy
LINUX_ARCH=(amd64 arm64)
WIN_ARCH=amd64
KIND_CLUSTER=kind

# Not used for pushing images, just for local building on other GOOS. Defaults to
# grabbing from the local go env but can be set manually to avoid that requirement.
HOST_GOOS=$(go env GOOS)
HOST_GOARCH=$(go env GOARCH)

# --tags allows detecting non-annotated tags as well as annotated ones
GIT_VERSION=$(git describe --always --dirty --tags)
IMAGE_VERSION=$(git describe --always --dirty --tags)
IMAGE_TAG=$(echo "$IMAGE_VERSION" | cut -d. -f1,2)
IMAGE_BRANCH=$(git rev-parse --abbrev-ref HEAD | sed 's/\///g')
GIT_REF_SHORT=$(git rev-parse --short=8 --verify HEAD)
GIT_REF_LONG=$(git rev-parse --verify HEAD)

BUILDMNT=/go/src/$GOTARGET
BUILD_IMAGE=golang:1.16
AMD_IMAGE=gcr.io/distroless/static:nonroot
ARM_IMAGE=gcr.io/distroless/static:nonroot-arm64
WIN_IMAGE=mcr.microsoft.com/windows/servercore:1809
TEST_IMAGE="sonobuoy/testimage:v0.1"

unit_local() {
	GODEBUG=x509ignoreCN=0 go test ${VERBOSE:+-v} -timeout 60s -coverprofile=coverage.txt -covermode=atomic $GOTARGET/cmd/... $GOTARGET/pkg/...
}

unit() {
	docker run --rm -v "$(pwd)":$BUILDMNT -w $BUILDMNT $BUILD_IMAGE /bin/sh -c \
    "GODEBUG=x509ignoreCN=0 go test ${VERBOSE:+-v} -timeout 60s -coverprofile=coverage.txt -covermode=atomic $GOTARGET/cmd/... $GOTARGET/pkg/..."
}

stress() {
	docker run --rm -v "$(pwd)":$BUILDMNT -w $BUILDMNT $BUILD_IMAGE /bin/sh -c \
    "GODEBUG=x509ignoreCN=0 go test ${VERBOSE:+-v} -timeout 60s -coverprofile=coverage.txt -covermode=atomic $GOTARGET/test/stress/..."
}

integration() {
	docker run --rm \
        -v "$(pwd)":$BUILDMNT \
        -v /tmp/artifacts:/tmp/artifacts \
        -v "${HOME}"/.kube/config:/root/.kube/kubeconfig \
        --env KUBECONFIG=/root/.kube/kubeconfig \
        -w "$BUILDMNT" \
        --env ARTIFACTS_DIR=/tmp/artifacts \
        --env SONOBUOY_CLI="$SONOBUOY_CLI" \
        --env GODEBUG=x509ignoreCN=0 \
        --network host \
        "$BUILD_IMAGE" \
    go test ${VERBOSE:+-v} -timeout 3m -tags=integration "$GOTARGET"/test/integration/...
}

lint() {
	docker run --rm -v "$(pwd)":$BUILDMNT -w $BUILDMNT $BUILD_IMAGE /bin/sh -c \
    "golint -set_exit_status ${VERBOSE:+-v} -timeout 60s $GOTARGET/cmd/... $GOTARGET/pkg/..."
}

vet() {
	docker run --rm -v "$(pwd)":$BUILDMNT -w $BUILDMNT $BUILD_IMAGE /bin/sh -c \
    "CGO_ENABLED=0 go vet ${VERBOSE:+-v} -timeout 60s $GOTARGET/cmd/... $GOTARGET/pkg/..."
}

pre() {
	wget https://github.com/estesp/manifest-tool/releases/download/v1.0.1/manifest-tool-linux-amd64 \
        -O manifest-tool
	chmod +x ./manifest-tool
	echo "$DOCKERHUB_TOKEN" | docker login --username sonobuoybot --password-stdin
}

build_container_dockerfile_image() {
	docker build \
        -t "$REGISTRY/$2:$IMAGE_VERSION" \
        -t "$REGISTRY/$2:$IMAGE_TAG" \
        -t "$REGISTRY/$2:$IMAGE_BRANCH" \
        -t "$REGISTRY/$2:$GIT_REF_SHORT" \
        -f "$1" \
		.
}

gen_dockerfile_for_os_arch(){
    if [ "$1" = "linux" ]; then
        if [ "$2" = "amd64" ]; then
            sed -e "s|BASEIMAGE|$AMD_IMAGE|g" \
                -e 's|CMD1||g' \
                -e 's|BINARY|build/linux/amd64/sonobuoy|g' Dockerfile > "Dockerfile-$arch"
        elif [ "$2" = "arm64" ]; then
            sed -e "s|BASEIMAGE|$ARM_IMAGE|g" \
                -e 's|CMD1||g' \
                -e 's|BINARY|build/linux/arm64/sonobuoy|g' Dockerfile > "Dockerfile-$arch"
        else
            echo "Linux ARCH unknown"
        fi
    elif [ "$2" = "windows" ]; then
        if [ "$arch" = "amd64" ]; then
			sed -e "s|BASEIMAGE|$WIN_IMAGE|g" \
			    -e 's|BINARY|build/windows/amd64/sonobuoy.exe|g' DockerfileWindows > "DockerfileWindows-$arch"
			build_container_dockerfile_image "DockerfileWindows-$arch" sonobuoy
			build_container_dockerfile_image "DockerfileWindows-$arch" "sonobuoy-win-$arch"
		else 
			echo "Windows ARCH unknown"
        fi
    else
        echo "OS unknown"
    fi
}

build_container_os_arch_image(){
    gen_dockerfile_for_os_arch "$1" "$2"
    if [ "$1" = "windows" ]; then dockerfile=DockerfileWindows ; else dockerfile=Dockerfile ; fi
    build_container_dockerfile_image "$dockerfile-$2" "$3"
}

linux_containers() {
	for arch in "${LINUX_ARCH[@]}"; do
        build_container_os_arch_image linux "$arch" sonobuoy
        build_container_os_arch_image linux "$arch" sonobuoy-"$arch"
        build_container_dockerfile_image Dockerfile-"$arch" sonobuoy-"$arch"
	done
}

windows_containers() {
	for arch in "${WIN_ARCH[@]}"; do
        build_container_dockerfile_image DockerfileWindows-"$arch" sonobuoy
        build_container_dockerfile_image DockerfileWindows-"$arch" sonobuoy-win-"$arch"
	done
}

build_binary_GOOS_GOARCH() {
	LDFLAGS="-s -w -X $GOTARGET/pkg/buildinfo.Version=$GIT_VERSION -X $GOTARGET/pkg/buildinfo.GitSHA=$GIT_REF_LONG"
    args=(${VERBOSE:+-v} -ldflags "${LDFLAGS}" "$GOTARGET")
    if [ "$VERBOSE" ]; then args+=("-v"); fi;

    echo Building "$1"/"$2"
    mkdir -p build/"$1"/"$2"

    if [ "$1" = "windows" ]; then
        BINARY="sonobuoy.exe"
    else
        BINARY="sonobuoy"
    fi

    # Avoid quoting nightmare by not running in /bin/sh
    docker run --rm -v "$(pwd)":"$BUILDMNT" -w "$BUILDMNT" \
        -e CGO_ENABLED=0 -e GOOS="$1" -e GOARCH="$2" "$BUILD_IMAGE" \
        go build -o build/"$1"/"$2"/"$BINARY" "${args[@]}" "$GOTARGET"
}

build_binaries() {
    for arch in "${LINUX_ARCH[@]}"; do
        build_binary_GOOS_GOARCH linux "$arch"
    done
    for arch in "${WIN_ARCH[@]}"; do
        build_binary_GOOS_GOARCH windows "$arch"
    done
}

native() {
    LDFLAGS="-s -w -X $GOTARGET/pkg/buildinfo.Version=$GIT_VERSION -X $GOTARGET/pkg/buildinfo.GitSHA=$GIT_REF_LONG"
    args=("${VERBOSE:+-v}" -ldflags "${LDFLAGS}" "$GOTARGET")
    CGO_ENABLED=0 GOOS="$HOST_GOOS" GOARCH="$HOST_GOARCH" go build -o sonobuoy "${args[@]}"
}

push_images() {
	docker push "$REGISTRY/$TARGET:$IMAGE_BRANCH"
	docker push "$REGISTRY/$TARGET:$GIT_REF_SHORT"
	if git describe --tags --exact-match >/dev/null 2>&1; then
		docker tag "$REGISTRY/$TARGET:$IMAGE_VERSION" "$REGISTRY/$TARGET:$IMAGE_TAG"
		docker tag "$REGISTRY/$TARGET:$IMAGE_VERSION" "$REGISTRY/$TARGET:latest"
		docker push "$REGISTRY/$TARGET:$IMAGE_VERSION"
		docker push "$REGISTRY/$TARGET:$IMAGE_TAG"
		docker push "$REGISTRY/$TARGET:latest"
	fi
}

gen_manifest_with_tag() {
	mkdir -p build

    if [ "$PUSH_WINDOWS" ]; then
        sed -e "s|TAG|$1|g" \
        -e "s|REGISTRY|$REGISTRY|g" \
        -e 's/WIN_ONLY//g' \
        manifest_spec.yaml.tmpl > ./build/manifest_spec.yaml;
    else
        echo 'PUSH_WINDOWS not set, not including Windows in manifest'
        sed -e "s|TAG|$1|g" \
        -e "s|REGISTRY|$REGISTRY|g" \
        -e '/^WIN_ONLY/d' \
        manifest_spec.yaml.tmpl > ./build/manifest_spec.yaml;
    fi
}

push_manifest_with_tag() {
    gen_manifest_with_tag "$1"
	./manifest-tool push from-spec ./build/manifest_spec.yaml
}

push() {
	# Assumes you have the images built or loaded already. Not
	# added as dependency due to having both Linux/Windows
	# prereqs which can't be done on the same machine.
    if [ "$PUSH_WINDOWS" ] ; then
        for arch in "${WIN_ARCH[@]}"; do
            push_images TARGET="sonobuoy-win-$arch"
        done
    else
        echo 'PUSH_WINDOWS not set, not pushing Windows images'
        for arch in "${LINUX_ARCH[@]}"; do
            push_images TARGET="sonobuoy-$arch"
        done
    fi

	push_manifest_with_tag "$IMAGE_BRANCH"
	push_manifest_with_tag "$GIT_REF_SHORT"

	if git describe --tags --exact-match >/dev/null 2>&1 ; then
		push_manifest_with_tag "$IMAGE_VERSION"
		push_manifest_with_tag latest
		push_manifest_with_tag "$IMAGE_TAG"
	fi
}

remove-image() {
	docker rmi -f "$(docker images "$REGISTRY/$1" -a -q)" || true
}

clean() {
    # Best effort for clean; don't exit if failure.
    set +e
	rm -f "$TARGET"
	rm Dockerfile-*
	rm DockerfileWindows-*
	rm -rf build

	for arch in "${LINUX_ARCH[@]}"; do
		remove-image "$TARGET-$arch"
	done
	for arch in "${WIN_ARCH[@]}"; do
		remove-image "$TARGET-win-$arch"
	done
    remove-image "$TARGET"
    set -e
}

# kind_images will build the kind-node image. Generally building the base image is not necessary
# and we can use the upstream kindest/base image.
kind_images() {
    K8S_PATH="$GOPATH/src/github.com/kubernetes/kubernetes"
    KIND_K8S_TAG="$(cd "$K8S_PATH" && git describe)"
	kind build node-image --kube-root="$K8S_PATH" --image "$REGISTRY/kind-node:$KIND_K8S_TAG"
}

# push_kind_images will push the same image kind_images just built our registry.
push_kind_images() {
    K8S_PATH="$GOPATH"/src/github.com/kubernetes/kubernetes
    KIND_K8S_TAG="$(cd "$K8S_PATH" && git describe)"
	docker push "$REGISTRY/kind-node:$KIND_K8S_TAG"
}

# check-kind-env will show you what will be built/tagged before doing so with kind_images
check-kind-env() {
    if [ -z "$K8S_PATH" ] ; then
        echo K8S_PATH is undefined
        exit 1
    fi
    if [ -z "$KIND_K8S_TAG" ] ; then
        echo KIND_K8S_TAG is undefined
        exit 1
    fi
	echo --kube-root="$K8S_PATH" tagging as --image "$REGISTRY/kind-node:$KIND_K8S_TAG"
}

setup_kind_cluster(){
    if ! kind get clusters | grep -q "^$KIND_CLUSTER$"; then
        kind create cluster --name "$KIND_CLUSTER" --config kind-config.yaml
        # Although the cluster has been created, not all the pods in kube-system are created/available
        sleep 20
    fi
}

# High level function; composes functions above and additional functionality. Typically just used by CI.
prepare_and_run_integration() {
    clean
    setup_kind_cluster
    
    # Build the test images
    (
        cd test/integration/testImage
        ./build.sh
    )
    build_binaries
    linux_containers

    # Load images into kind cluster. Assume sonobuoy images already built.
    kind load docker-image --name "$KIND_CLUSTER" "$REGISTRY/$TARGET:$IMAGE_VERSION" || true
    kind load docker-image --name "$KIND_CLUSTER" "$TEST_IMAGE"

    VERBOSE=true SONOBUOY_CLI=../../build/linux/amd64/sonobuoy integration
}
