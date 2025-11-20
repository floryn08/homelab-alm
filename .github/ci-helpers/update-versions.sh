#!/bin/bash

set -euo pipefail

version=$1

if [ -z "$version" ]; then
  echo "Usage: $0 <version>"
  exit 1
fi

echo "Setting version to $version..."

if [ ! -f deployment/helm/Chart.yaml ] || [ ! -f deployment/helm/values.yaml ]; then
  echo "Chart.yaml or values.yaml not found!"
  exit 1
fi

echo "Before:"
grep "image:" deployment/helm/values.yaml || true

# Update Chart.yaml
sed -i "s/^version:.*/version: ${version}/" deployment/helm/Chart.yaml
sed -i "s/^appVersion:.*/appVersion: ${version}/" deployment/helm/Chart.yaml

# Update image tag in values.yaml
sed -i "s|\(image: .*:\).*|\1v${version}|" deployment/helm/values.yaml

echo "After:"
grep "image:" deployment/helm/values.yaml || true
