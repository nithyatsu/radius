#!/bin/bash

# Get debug directory from environment or default
DEBUG_DEV_ROOT=${DEBUG_DEV_ROOT:-"$(pwd)/debug_files"}

cd "$DEBUG_DEV_ROOT"

echo "📊 Radius Component Status:"
echo "=========================="

components=("ucp" "controller" "applications-rp" "dynamic-rp")

# Service port that each component listens on once fully initialized. The PID file
# typically points at dlv, which can stay alive after the wrapped binary exits, so
# checking the listener port is the real liveness signal. Using a case statement
# instead of `declare -A` for compatibility with macOS's bash 3.2.
component_port() {
  case "$1" in
    ucp) echo 9000 ;;
    controller) echo 7073 ;;
    applications-rp) echo 8080 ;;
    dynamic-rp) echo 8082 ;;
    *) echo "" ;;
  esac
}

for component in "${components[@]}"; do
  port=$(component_port "$component")
  listener_up=0
  if [ -n "$port" ] && lsof -nP -iTCP:"$port" -sTCP:LISTEN >/dev/null 2>&1; then
    listener_up=1
  fi
  if [ -f "logs/${component}.pid" ]; then
    pid=$(cat "logs/${component}.pid")
    if kill -0 "$pid" 2>/dev/null; then
      if [ "$listener_up" -eq 1 ]; then
        echo "✅ $component (PID: $pid, port: $port) - Running"
      else
        echo "⚠️  $component (PID: $pid) - dlv alive but binary not listening on :$port (check logs/${component}.log)"
      fi
    else
      echo "❌ $component - PID file exists but process not running"
    fi
  else
    echo "❌ $component - Not running (no PID file)"
  fi
done

# Check deployment engine (k3d deployment)
echo ""
echo "🚢 Deployment Engine Status:"
echo "=========================="

if command -v kubectl >/dev/null 2>&1; then
  if kubectl --context k3d-radius-debug get deployment deployment-engine -n default >/dev/null 2>&1; then
    status=$(kubectl --context k3d-radius-debug get deployment deployment-engine -n default -o jsonpath='{.status.conditions[?(@.type=="Available")].status}' 2>/dev/null)
    if [ "$status" = "True" ]; then
      replicas=$(kubectl --context k3d-radius-debug get deployment deployment-engine -n default -o jsonpath='{.status.readyReplicas}' 2>/dev/null)
      echo "✅ deployment-engine (k3d) - Running ($replicas replicas ready)"
    else
      echo "❌ deployment-engine (k3d) - Not ready"
    fi
  else
    echo "❌ deployment-engine - Not found in k3d cluster"
  fi
else
  echo "⚠️  deployment-engine - Cannot check status (kubectl not available)"
fi
