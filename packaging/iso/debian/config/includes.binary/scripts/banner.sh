#!/bin/sh
# NithronOS Installation Banner - Non-blocking
# Displays a brief ASCII banner and exits immediately

# Run in subshell to ensure non-blocking
(
    # Clear screen and display banner
    clear 2>/dev/null || true
    
    cat << 'EOF'

================================================================================
     _   _ _ _   _                   ___  ____  
    | \ | (_) |_| |__  _ __ ___  _ _/ _ \/ ___| 
    |  \| | | __| '_ \| '__/ _ \| ' | | | \___ \ 
    | |\  | | |_| | | | | | (_) | | | |_| |___) |
    |_| \_|_|\__|_| |_|_|  \___/|_| |\___/|____/ 
                                                  
                  NithronOS Installation System
                       Version 1.0 - Bookworm
================================================================================

    Initializing Debian Installer...
    
    This automated installation will:
    • Configure system with safe defaults
    • Install NithronOS core services
    • Enable SSH for remote management
    
    Installation will begin shortly...

================================================================================

EOF
    
    # Brief pause to show banner (runs in background)
    sleep 3
) &

# Exit immediately to not block installer
exit 0
