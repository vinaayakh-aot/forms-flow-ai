#!/bin/bash

# FormsFlow.ai Version Update Script
# Wrapper script for easy version management

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
PYTHON_SCRIPT="$SCRIPT_DIR/update-versions.py"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_usage() {
    echo "FormsFlow.ai Version Update Tool"
    echo ""
    echo "Usage: $0 [COMMAND] [OPTIONS]"
    echo ""
    echo "Commands:"
    echo "  check         Preview what would be changed (dry-run)"
    echo "  update        Apply version updates to all configured files"
    echo "  status        Show current version and affected files"
    echo "  help          Show this help message"
    echo ""
    echo "Options:"
    echo "  --config FILE Use custom configuration file"
    echo ""
    echo "Examples:"
    echo "  $0 check                    # Preview changes"
    echo "  $0 update                   # Apply updates"
    echo "  $0 status                   # Show current status"
    echo "  $0 update --config my.json # Use custom config"
}

show_status() {
    echo -e "${BLUE}📋 FormsFlow.ai Version Status${NC}"
    echo ""
    
    if [ -f "$PROJECT_ROOT/VERSION" ]; then
        VERSION=$(cat "$PROJECT_ROOT/VERSION" | tr -d '\n')
        echo -e "${GREEN}Current Version: $VERSION${NC}"
    else
        echo -e "${RED}❌ VERSION file not found at $PROJECT_ROOT/VERSION${NC}"
        exit 1
    fi
    
    echo ""
    echo "📁 Configured files to update:"
    
    if [ -f "$SCRIPT_DIR/config.json" ]; then
        python3 -c "
import json
import sys
with open('$SCRIPT_DIR/config.json', 'r') as f:
    config = json.load(f)
for file_config in config.get('files', []):
    print(f'  • {file_config[\"path\"]}')
"
    else
        echo -e "${RED}❌ Config file not found at $SCRIPT_DIR/config.json${NC}"
        exit 1
    fi
    
    echo ""
    echo "💡 Run '$0 check' to preview changes or '$0 update' to apply them."
}

check_dependencies() {
    if ! command -v python3 &> /dev/null; then
        echo -e "${RED}❌ Python 3 is required but not installed.${NC}"
        exit 1
    fi
}

run_dry_run() {
    echo -e "${YELLOW}🔍 Checking what would be updated...${NC}"
    echo ""
    python3 "$PYTHON_SCRIPT" --dry-run "$@"
}

run_update() {
    echo -e "${GREEN}🚀 Updating versions...${NC}"
    echo ""
    python3 "$PYTHON_SCRIPT" "$@"
    
    if [ $? -eq 0 ]; then
        echo ""
        echo -e "${GREEN}✅ Version update completed successfully!${NC}"
        echo ""
        echo "Next steps:"
        echo "  1. Review the changes: git diff"
        echo "  2. Test your applications"
        echo "  3. Commit the changes: git add . && git commit -m 'chore: update versions to $(cat "$PROJECT_ROOT/VERSION")'"
    else
        echo -e "${RED}❌ Version update failed!${NC}"
        exit 1
    fi
}

# Main script logic
case "${1:-help}" in
    "check"|"preview"|"dry-run")
        check_dependencies
        shift
        run_dry_run "$@"
        ;;
    "update"|"apply")
        check_dependencies
        shift
        run_update "$@"
        ;;
    "status"|"info")
        show_status
        ;;
    "help"|"--help"|"-h")
        print_usage
        ;;
    *)
        echo -e "${RED}❌ Unknown command: $1${NC}"
        echo ""
        print_usage
        exit 1
        ;;
esac 