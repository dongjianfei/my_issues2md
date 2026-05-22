---
allowed-tools: Bash(docker:*), Bash(make docker-build)
---

Build the Docker image for issue2md using `make docker-build`.

Image tag: use "$1" if provided, otherwise default to "latest".

Steps:
1. Determine the tag: if "$1" is empty, use "latest"
2. Execute `make docker-build DOCKER_TAG=<tag>`
3. If successful, show the image size with `docker images issue2md:<tag>`
4. If failed, analyze the error output and suggest fixes
