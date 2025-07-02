#!/usr/bin/env python3
"""
FormsFlow.ai Version Update Script - Human-Friendly Edition

This script supports both the original complex regex config and a new simplified config format
that's much easier for humans to read and edit.

Usage:
    python version-cli/update-versions-simple.py [--dry-run] [--config config-simple.json]
"""

import json
import re
import sys
import argparse
from pathlib import Path
from typing import Dict, List, Any, Optional


class SimpleVersionUpdater:
    def __init__(self, config_path: Optional[str] = None):
        self.root_dir = Path(__file__).parent.parent
        self.version_file = self.root_dir / "VERSION"
        self.config_path = config_path or str(self.root_dir / "version-cli" / "config-simple.json")
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
    
    def get_version_components(self) -> Dict[str, str]:
        """Extract version components for different formats."""
        version = self.current_version
        version_no_v = version.lstrip('v')
        
        return {
            'VERSION': version,
            'VERSION_NO_V': version_no_v,
            'VERSION_WITH_V': version if version.startswith('v') else f'v{version}'
        }
    
    def expand_version_patterns(self) -> Dict[str, str]:
        """Expand version patterns defined in config."""
        if 'version_patterns' not in self.config:
            return {}
            
        patterns = {}
        version_components = self.get_version_components()
        
        for pattern_name, pattern_template in self.config['version_patterns'].items():
            expanded = pattern_template
            for component, value in version_components.items():
                expanded = expanded.replace(f'{{{component}}}', value)
            patterns[pattern_name] = expanded
            
        return patterns
    
    def wildcard_to_regex(self, pattern: str) -> str:
        """Convert simple wildcard pattern to regex."""
        # Escape special regex characters except *
        escaped = re.escape(pattern)
        # Replace escaped \* with actual wildcard regex
        regex_pattern = escaped.replace(r'\*', r'[^"\\s]*')
        return regex_pattern
    
    def create_pattern_from_simple(self, update_config: Dict[str, Any], version_patterns: Dict[str, str]) -> tuple:
        """Convert simple config format to search/replace patterns."""
        find_pattern = update_config['find']
        replace_pattern = update_config['replace']
        update_type = update_config.get('type', 'simple_pattern')
        
        # Expand pattern references like {{url_version}}
        for pattern_name, pattern_value in version_patterns.items():
            replace_pattern = replace_pattern.replace(f'{{{{{pattern_name}}}}}', pattern_value)
        
        if update_type == 'regex':
            # Use the pattern as-is (it's already a regex)
            search_regex = find_pattern
        elif update_type == 'version_in_url':
            # For URL versions, we need to be more specific
            search_regex = self.wildcard_to_regex(find_pattern)
            search_regex = search_regex.replace(r'[^"\\s]*', r'v[0-9]+\.[0-9]+\.[0-9]+-[a-zA-Z0-9]+')
        elif update_type == 'json_field':
            # For JSON fields, create a more specific pattern
            search_regex = self.wildcard_to_regex(find_pattern)
            search_regex = search_regex.replace(r'[^"\\s]*', r'[0-9]+\.[0-9]+\.[0-9]+-[a-zA-Z0-9]+')
        else:
            # Simple pattern matching
            search_regex = self.wildcard_to_regex(find_pattern)
            search_regex = search_regex.replace(r'[^"\\s]*', r'v?[0-9]+\.[0-9]+\.[0-9]+-[a-zA-Z0-9]+')
        
        return search_regex, replace_pattern
    
    def apply_context_filtering(self, content: str, pattern: str, context: Dict[str, Any]) -> List[tuple]:
        """Apply context filtering to find the right matches."""
        matches = list(re.finditer(pattern, content))
        
        if 'near' not in context:
            return [(m.start(), m.end(), m.group()) for m in matches]
        
        near_pattern = context['near']
        max_matches = context.get('max_matches', 1)
        max_distance = context.get('max_distance', 200)  # characters
        
        # Find all "near" pattern occurrences
        near_matches = list(re.finditer(re.escape(near_pattern), content))
        
        filtered_matches = []
        for near_match in near_matches:
            near_pos = near_match.start()
            
            # Find matches within distance of this "near" pattern
            for match in matches:
                distance = abs(match.start() - near_pos)
                if distance <= max_distance:
                    filtered_matches.append((match.start(), match.end(), match.group()))
                    if len(filtered_matches) >= max_matches:
                        break
            
            if len(filtered_matches) >= max_matches:
                break
        
        return filtered_matches[:max_matches]
    
    def update_file_simple(self, file_config: Dict[str, Any], dry_run: bool = False) -> int:
        """Update a file using the simple configuration format."""
        file_path = file_config['path']
        file_name = file_config.get('name', file_path)
        updates = file_config.get('updates', [])
        
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
        total_changes = 0
        version_patterns = self.expand_version_patterns()
        
        for update in updates:
            description = update.get('description', 'No description')
            
            try:
                search_pattern, replace_template = self.create_pattern_from_simple(update, version_patterns)
                
                if 'context' in update:
                    # Use context-aware matching
                    matches = self.apply_context_filtering(content, search_pattern, update['context'])
                    changes = len(matches)
                    
                    # Replace matches in reverse order to maintain positions
                    for start, end, match_text in reversed(matches):
                        replacement = re.sub(search_pattern, replace_template, match_text)
                        content = content[:start] + replacement + content[end:]
                        
                else:
                    # Standard regex replacement
                    new_content, changes = re.subn(search_pattern, replace_template, content)
                    content = new_content
                
                if changes > 0:
                    total_changes += changes
                    print(f"  ✅ {description} ({changes} matches)")
                
            except re.error as e:
                print(f"  ❌ Pattern error in '{description}': {e}")
                continue
        
        if total_changes > 0 and not dry_run:
            try:
                with open(full_path, 'w', encoding='utf-8') as f:
                    f.write(content)
                print(f"📝 Updated {file_name} ({total_changes} changes)")
            except Exception as e:
                print(f"❌ Error writing {file_path}: {e}")
                return 0
        elif total_changes > 0:
            print(f"🔍 [DRY RUN] Would update {file_name} ({total_changes} changes)")
            
        return total_changes
    
    def update_all_files(self, dry_run: bool = False) -> None:
        """Update all files specified in the configuration."""
        print(f"🚀 Starting version update to: {self.current_version}")
        print(f"📁 Working directory: {self.root_dir}")
        
        if dry_run:
            print("🔍 DRY RUN MODE - No files will be modified")
            
        print(f"📋 Using config: {self.config_path}")
        
        # Show expanded version patterns
        version_patterns = self.expand_version_patterns()
        if version_patterns:
            print("\n🔧 Version patterns:")
            for name, value in version_patterns.items():
                print(f"  {name}: {value}")
            
        total_changes = 0
        
        for file_config in self.config.get('files', []):
            file_name = file_config.get('name', file_config['path'])
            print(f"\n📄 Processing: {file_name}")
            changes = self.update_file_simple(file_config, dry_run)
            total_changes += changes
            
        print(f"\n{'🔍 [DRY RUN] ' if dry_run else ''}✨ Update complete! Total changes: {total_changes}")
        
        if dry_run and total_changes > 0:
            print("Run without --dry-run to apply changes.")


def main():
    parser = argparse.ArgumentParser(
        description="Update versions across FormsFlow.ai repository (Human-Friendly Edition)",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
This version supports a simplified, human-readable configuration format.

Examples:
  python version-cli/update-versions-simple.py --dry-run    # Preview changes
  python version-cli/update-versions-simple.py             # Apply updates
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
        help='Path to configuration file (default: version-cli/config-simple.json)'
    )
    
    args = parser.parse_args()
    
    try:
        updater = SimpleVersionUpdater(args.config)
        updater.update_all_files(dry_run=args.dry_run)
    except KeyboardInterrupt:
        print("\n❌ Update cancelled by user")
        sys.exit(1)
    except Exception as e:
        print(f"❌ Unexpected error: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == "__main__":
    main() 