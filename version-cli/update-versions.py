#!/usr/bin/env python3
"""
FormsFlow.ai Version Update Script

This script reads the version from the VERSION file and updates all version references
across the repository based on configurable patterns.

Usage:
    python version-cli/update-versions.py [--dry-run] [--config config.json]
"""

import json
import re
import sys
import argparse
from pathlib import Path
from typing import Dict, List, Any, Optional


class VersionUpdater:
    def __init__(self, config_path: Optional[str] = None):
        self.root_dir = Path(__file__).parent.parent
        self.version_file = self.root_dir / "VERSION"
        self.config_path = config_path or str(self.root_dir / "version-cli" / "config.json")
        self.config = self.load_config()
        self.current_version = self.read_version()
        
    def read_version(self) -> str:
        """Read the current version from the VERSION file."""
        try:
            with open(self.version_file, 'r') as f:
                return f.read().strip()
        except FileNotFoundError:
            print(f"❌ VERSION file not found at {self.version_file}")
            sys.exit(1)
            
    def load_config(self) -> Dict[str, Any]:
        """Load configuration file that defines which files to update and how."""
        try:
            with open(self.config_path, 'r') as f:
                return json.load(f)
        except FileNotFoundError:
            print(f"❌ Config file not found at {self.config_path}")
            sys.exit(1)
        except json.JSONDecodeError as e:
            print(f"❌ Error parsing config file: {e}")
            sys.exit(1)
            
    def get_version_variants(self) -> Dict[str, str]:
        """Generate different version formats from the main version."""
        version = self.current_version
        
        # Remove 'v' prefix if present for some variants
        version_no_v = version.lstrip('v')
        
        return {
            'full': version,  # v7.1.0-alpha
            'no_prefix': version_no_v,  # 7.1.0-alpha
            'with_v': version if version.startswith('v') else f'v{version}',  # v7.1.0-alpha
            'at_version': f'@{version}',  # @v7.1.0-alpha
        }
        
    def update_file(self, file_path: str, patterns: List[Dict[str, str]], dry_run: bool = False) -> int:
        """Update a single file with the specified patterns."""
        full_path = self.root_dir / file_path
        
        if not full_path.exists():
            print(f"⚠️  File not found: {file_path}")
            return 0
            
        try:
            with open(full_path, 'r', encoding='utf-8') as f:
                content = f.read()
        except Exception as e:
            print(f"❌ Error reading {file_path}: {e}")
            return 0
            
        original_content = content
        changes_made = 0
        version_variants = self.get_version_variants()
        
        for pattern in patterns:
            search_pattern = pattern['search']
            replace_template = pattern['replace']
            
            # Replace version placeholders in the pattern
            for variant_name, variant_value in version_variants.items():
                search_pattern = search_pattern.replace(f'{{{variant_name}}}', variant_value)
                replace_template = replace_template.replace(f'{{{variant_name}}}', variant_value)
            
            # Perform the replacement
            try:
                if pattern.get('regex', False):
                    # Use regex replacement
                    new_content, count = re.subn(search_pattern, replace_template, content)
                else:
                    # Use simple string replacement
                    count = content.count(search_pattern)
                    new_content = content.replace(search_pattern, replace_template)
                    
                if count > 0:
                    content = new_content
                    changes_made += count
                    print(f"  ✅ Applied pattern (found {count} matches): {pattern.get('description', 'No description')}")
                    
            except re.error as e:
                print(f"  ❌ Regex error in pattern: {e}")
                continue
                
        # Check if content actually changed
        actual_changes = 0 if content == original_content else changes_made
        
        if actual_changes > 0 and not dry_run:
            try:
                with open(full_path, 'w', encoding='utf-8') as f:
                    f.write(content)
                print(f"📝 Updated {file_path} ({actual_changes} changes)")
            except Exception as e:
                print(f"❌ Error writing {file_path}: {e}")
                return 0
        elif actual_changes > 0:
            print(f"🔍 [DRY RUN] Would update {file_path} ({actual_changes} changes)")
        elif changes_made > 0 and actual_changes == 0:
            print(f"✅ {file_path} already up to date (no changes needed)")
            
        return actual_changes
        
    def update_all_files(self, dry_run: bool = False) -> None:
        """Update all files specified in the configuration."""
        print(f"🚀 Starting version update to: {self.current_version}")
        print(f"📁 Working directory: {self.root_dir}")
        
        if dry_run:
            print("🔍 DRY RUN MODE - No files will be modified")
            
        total_changes = 0
        
        for file_config in self.config.get('files', []):
            file_path = file_config['path']
            patterns = file_config['patterns']
            
            print(f"\n📄 Processing: {file_path}")
            changes = self.update_file(file_path, patterns, dry_run)
            total_changes += changes
            
        print(f"\n{'🔍 [DRY RUN] ' if dry_run else ''}✨ Update complete! Total changes: {total_changes}")
        
        if dry_run and total_changes > 0:
            print("Run without --dry-run to apply changes.")


def main():
    parser = argparse.ArgumentParser(
        description="Update versions across FormsFlow.ai repository",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  python version-cli/update-versions.py                    # Update all files
  python version-cli/update-versions.py --dry-run          # Preview changes
  python version-cli/update-versions.py --config my.json  # Use custom config
        """
    )
    
    parser.add_argument(
        '--dry-run', 
        action='store_true',
        help='Preview changes without modifying files'
    )
    
    parser.add_argument(
        '--config',
        type=str,
        help='Path to configuration file (default: version-cli/config.json)'
    )
    
    args = parser.parse_args()
    
    try:
        updater = VersionUpdater(args.config)
        updater.update_all_files(dry_run=args.dry_run)
    except KeyboardInterrupt:
        print("\n❌ Update cancelled by user")
        sys.exit(1)
    except Exception as e:
        print(f"❌ Unexpected error: {e}")
        sys.exit(1)


if __name__ == "__main__":
    main() 