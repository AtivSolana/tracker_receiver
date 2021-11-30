#!/bin/sh

# Abort on any error (including if wait-for-it fails).
set -e

# Wait for the backend to be up, if we know where it is.
if [ -n "$MQ_HOST" ]; then
  /receiver/wait-for-it.sh "$MQ_HOST:5672"
fi

# Run the main container command.
exec "$@"