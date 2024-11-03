#!/bin/bash

set -e

DATA_DIR="./data"
CONTAINER_NAME="ebpf-firewall"
IMAGE_NAME="ebpf-firewall"
IMAGE_TAG="latest"
INTERFACE=""
API_ADDR=":5678"
AUTH_TOKEN=""
NO_CACHE=false
FOLLOW=false
TAIL=""

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
	echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
	echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
	echo -e "${RED}[ERROR]${NC} $1"
}

container_exists() {
	docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"
}

build() {
	log_info "Building ${IMAGE_NAME}:${IMAGE_TAG}"
	
	BUILD_ARGS=""
	if [ "$NO_CACHE" = true ]; then
		BUILD_ARGS="--no-cache"
		log_info "Building without cache"
	fi

	if ! docker build \
		-t "${IMAGE_NAME}:${IMAGE_TAG}" \
		--progress=plain \
		$BUILD_ARGS \
		-f Dockerfile .; then
		log_error "Build failed"
		exit 1
	fi
	log_info "Build completed successfully"
}

prune() {
	log_info "Cleaning up docker build cache"
	if ! docker builder prune -af --force; then
		log_error "Failed to clean up build cache"
		exit 1
	fi

	log_info "Cleaning up dangling images"
	if docker images -f "dangling=true" -q | grep -q .; then
		if ! docker rmi $(docker images -f "dangling=true" -q); then
			log_warn "Failed to remove some dangling images"
		else
			log_info "Dangling images cleaned up successfully"
		fi
	else
		log_info "No dangling images to clean up"
	fi

	log_info "Cache cleanup completed"
}

format_api_url() {
	local addr="$1"
	if [[ "$addr" == :* ]]; then
		echo "http://127.0.0.1${addr}"
	elif [[ "$addr" == 0.0.0.0:* ]]; then
		echo "http://127.0.0.1:${addr#0.0.0.0:}"
	else
		echo "http://${addr}"
	fi
}

start() {
	if container_exists; then
		if [ "$(docker ps -q -f name=${CONTAINER_NAME})" ]; then
			log_warn "Container ${CONTAINER_NAME} is already running"
			exit 0
		fi
		log_info "Starting existing container ${CONTAINER_NAME}"
		docker start "${CONTAINER_NAME}"
	else
		log_info "Creating and starting new container ${CONTAINER_NAME}"

		ENV_VARS=""
		if [ -n "$INTERFACE" ]; then
			ENV_VARS="$ENV_VARS -e EBPF_INTERFACE=$INTERFACE"
		fi
		if [ -n "$API_ADDR" ]; then
			ENV_VARS="$ENV_VARS -e EBPF_ADDR=$API_ADDR"
		fi
		if [ -n "$AUTH_TOKEN" ]; then
			ENV_VARS="$ENV_VARS -e EBPF_AUTH=$AUTH_TOKEN"
		fi

		docker run -d \
			--name "${CONTAINER_NAME}" \
			--cap-add=SYS_ADMIN \
			--cap-add=NET_ADMIN \
			--cap-add=SYS_RESOURCE \
			--restart=always \
			-v "${DATA_DIR}:/app/data" \
			-v /sys/kernel/debug:/sys/kernel/debug \
			-v /sys/fs/bpf:/sys/fs/bpf \
			-v /proc:/proc \
			--net=host \
			$ENV_VARS \
			"${IMAGE_NAME}:${IMAGE_TAG}"
	fi
	log_info "Container ${CONTAINER_NAME} is now running"
	sleep 2
	if [ -n "$AUTH_TOKEN" ]; then
		log_info "Auth Token: ${AUTH_TOKEN}"
	else
		AUTH_TOKEN=$(docker logs "${CONTAINER_NAME}" 2>&1 | grep "auth token:" | awk -F': ' '{print $2}')
		if [ -n "$AUTH_TOKEN" ]; then
			log_info "Auth Token: ${AUTH_TOKEN}"
		fi
	fi
	log_info "API URL: $(format_api_url ${API_ADDR})/?token=${AUTH_TOKEN}"
}

stop() {
	if ! container_exists; then
		log_warn "Container ${CONTAINER_NAME} does not exist"
		exit 0
	fi
	
	if [ ! "$(docker ps -q -f name=${CONTAINER_NAME})" ]; then
		log_warn "Container ${CONTAINER_NAME} is not running"
		exit 0
	fi
	
	log_info "Stopping container ${CONTAINER_NAME}"
	docker stop "${CONTAINER_NAME}"
	log_info "Container stopped successfully"
}

remove() {
	if ! container_exists; then
		log_warn "Container ${CONTAINER_NAME} does not exist"
		exit 0
	fi
	
	if [ "$(docker ps -q -f name=${CONTAINER_NAME})" ]; then
		log_info "Stopping container ${CONTAINER_NAME}"
		docker stop "${CONTAINER_NAME}"
	else
		log_info "Container ${CONTAINER_NAME} is not running"
	fi
	log_info "Removing container ${CONTAINER_NAME}"
	docker rm "${CONTAINER_NAME}"
	log_info "Container removed successfully"
}

clean() {
	if container_exists; then
		if [ "$(docker ps -q -f name=${CONTAINER_NAME})" ]; then
			log_info "Stopping container ${CONTAINER_NAME}"
			docker stop "${CONTAINER_NAME}"
		fi
		log_info "Removing container ${CONTAINER_NAME}"
		docker rm "${CONTAINER_NAME}"
	else
		log_info "No container ${CONTAINER_NAME} to remove"
	fi

	if docker images "${IMAGE_NAME}:${IMAGE_TAG}" --quiet | grep -q .; then
		log_info "Removing image ${IMAGE_NAME}:${IMAGE_TAG}"
		docker rmi "${IMAGE_NAME}:${IMAGE_TAG}"
		log_info "Image removed successfully"
	else
		log_info "No image ${IMAGE_NAME}:${IMAGE_TAG} to remove"
	fi
}

run() {
	log_info "Building and running ${CONTAINER_NAME}"
	build
	log_info "Starting container ${CONTAINER_NAME}"
	start
	log_info "Container ${CONTAINER_NAME} is now running with latest build"
}

logs() {
	if ! container_exists; then
		log_error "Container ${CONTAINER_NAME} does not exist"
		exit 1
	fi
	
	LOG_OPTS=""
	if [ "$FOLLOW" = true ]; then
		LOG_OPTS="${LOG_OPTS} -f"
	fi
	if [ -n "$TAIL" ]; then
		LOG_OPTS="${LOG_OPTS} --tail ${TAIL}"
	fi
	
	log_info "Showing logs for ${CONTAINER_NAME}"
	docker logs ${LOG_OPTS} "${CONTAINER_NAME}"
}

usage() {
	cat << EOF
Usage: $0 <command> [options]

Commands:
    build   Build the container image
    start   Start the container
    stop    Stop the container
    remove  Stop and remove the container
    clean   Remove both container and image
    run     Build image and start container
    prune   Clean up docker build cache
    logs    View container logs

Options:
    -t, --tag <tag>         Specify image tag (default: latest)
    -n, --no-cache          Build without using cache
    -f, --follow            Follow log output
    --tail <n>              Number of lines to show from the end of logs

Runtime Options:
    -i, --interface <name>  Network interface to monitor
    -p, --port <port>       API port (default: 5678)
    --addr <address>        API address (default: :5678)
    -d, --data <path>       Data directory path (default: ./data)
    -a, --auth <token>      API authentication token (default: auto generated)

Examples:
    $0 build --tag v1.0.0
    $0 run -i eth0 -p 8080
    $0 start -i ens33 --addr :18080
    $0 start -i ens33 --addr 192.168.1.100:8080 -a 1234567890
    $0 logs -f --tail 100
EOF
	exit 1
}

parse_args() {
	if [ $# -eq 0 ]; then
		usage
	fi

	COMMAND=$1
	shift

	while [ $# -gt 0 ]; do
		case $1 in
			-t|--tag)
				if [ -z "$2" ]; then
					log_error "Missing value for $1 option"
					usage
				fi
				IMAGE_TAG="$2"
				shift 2
				;;
			-n|--no-cache)
				NO_CACHE=true
				shift
				;;
			-f|--follow)
				FOLLOW=true
				shift
				;;
			--tail)
				if [ -z "$2" ]; then
					log_error "Missing value for $1 option"
					usage
				fi
				TAIL="$2"
				shift 2
				;;
			-i|--interface)
				if [ -z "$2" ]; then
					log_error "Missing value for $1 option"
					usage
				fi
				INTERFACE="$2"
				shift 2
				;;
			-p|--port)
				if [ -z "$2" ]; then
					log_error "Missing value for $1 option"
					usage
				fi
				API_ADDR=":$2"
				shift 2
				;;
			--addr)
				if [ -z "$2" ]; then
					log_error "Missing value for $1 option"
					usage
				fi
				API_ADDR="$2"
				shift 2
				;;
			-d|--data)
				if [ -z "$2" ]; then
					log_error "Missing value for $1 option"
					usage
				fi
				DATA_DIR="$2"
				shift 2
				;;
			-a|--auth)
				if [ -z "$2" ]; then
					log_error "Missing value for $1 option"
					usage
				fi
				AUTH_TOKEN="$2"
				shift 2
				;;
			*)
				log_error "Unknown option: $1"
				usage
				;;
		esac
	done

	case "${COMMAND}" in
		run|build|start|stop|remove|clean|prune|logs)
			;;
		*)
			log_error "Unknown command: ${COMMAND}"
			usage
			;;
	esac
}

main() {
	parse_args "$@"
	$COMMAND
}

main "$@"