#!/bin/bash
# Build script for code-relay admin

echo "Building React frontend..."
cd web
npm install
npm run build
cd ..

echo "Frontend build complete!"
echo "Build output is in: web/dist"

echo ""
echo "To build Go backend, run:"
echo "go build -o code-relay ."
