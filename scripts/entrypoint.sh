#!/bin/sh
set -e

# linkko-api entrypoint script
# This script handles different startup modes for EasyPanel/Docker

case "$1" in
    migrate)
        echo "Starting Database Migrations..."
        exec /usr/local/bin/linkko-api migrate
        ;;
    serve)
        echo "Starting Linkko API server..."
        exec /usr/local/bin/linkko-api serve
        ;;
    *)
        # Default behavior: if no known command, try to execute the arguments
        if [ -n "$1" ]; then
            exec /usr/local/bin/linkko-api "$@"
        else
            echo "No command specified, defaulting to serve"
            exec /usr/local/bin/linkko-api serve
        fi
        ;;
esac
