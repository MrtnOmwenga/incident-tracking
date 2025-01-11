#!/bin/bash
# Set docker socket permissions
sudo chmod 666 /var/run/docker.sock

# Execute the original Jenkins entrypoint
exec /usr/local/bin/jenkins.sh "$@"