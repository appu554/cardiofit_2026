#!/usr/bin/env python3
"""
Test DNS Resolution for Supabase

This script helps diagnose DNS issues and provides IP-based alternatives.
"""
import socket
import logging

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


def test_dns_resolution():
    """Test DNS resolution for Supabase hostname"""
    hostname = "db.auugxeqzgrnknklgwqrh.supabase.co"
    
    logger.info("🔍 Testing DNS Resolution")
    logger.info("=" * 50)
    
    try:
        # Try to resolve hostname
        logger.info(f"Resolving: {hostname}")
        ip_address = socket.gethostbyname(hostname)
        logger.info(f"✅ SUCCESS: {hostname} → {ip_address}")
        
        # Test connection to IP
        logger.info(f"Testing connection to {ip_address}:5432...")
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.settimeout(10)
        result = sock.connect_ex((ip_address, 5432))
        sock.close()
        
        if result == 0:
            logger.info(f"✅ Port 5432 is reachable on {ip_address}")
            
            # Provide IP-based connection string
            logger.info("")
            logger.info("🔧 IP-BASED CONNECTION STRING:")
            logger.info(f"postgresql://postgres:9FTqQnA4LRCsu8sw@{ip_address}:5432/postgres")
            logger.info("")
            logger.info("You can temporarily use this IP address in your config if DNS continues to fail.")
            
        else:
            logger.warning(f"⚠️ Port 5432 not reachable on {ip_address}")
        
        return ip_address
        
    except socket.gaierror as e:
        logger.error(f"❌ DNS Resolution failed: {e}")
        logger.info("")
        logger.info("🔧 TROUBLESHOOTING STEPS:")
        logger.info("1. Flush DNS cache: ipconfig /flushdns")
        logger.info("2. Change DNS servers to 8.8.8.8, 8.8.4.4")
        logger.info("3. Check internet connection: ping google.com")
        logger.info("4. Try mobile hotspot")
        logger.info("5. Check Windows Firewall")
        return None
    
    except Exception as e:
        logger.error(f"❌ Connection test failed: {e}")
        return None


def test_alternative_dns():
    """Test with different DNS servers"""
    logger.info("")
    logger.info("🔄 Testing Alternative DNS Resolution")
    logger.info("=" * 50)
    
    # Try with Google DNS
    try:
        import subprocess
        result = subprocess.run(
            ["nslookup", "db.auugxeqzgrnknklgwqrh.supabase.co", "8.8.8.8"],
            capture_output=True,
            text=True,
            timeout=10
        )
        
        if result.returncode == 0:
            logger.info("✅ Google DNS (8.8.8.8) can resolve the hostname")
            logger.info("Consider changing your DNS settings")
        else:
            logger.warning("⚠️ Google DNS also failed")
            
    except Exception as e:
        logger.warning(f"Could not test alternative DNS: {e}")


def main():
    """Main test function"""
    logger.info("🚀 Supabase DNS Resolution Diagnostic")
    
    # Test current DNS
    ip_address = test_dns_resolution()
    
    # Test alternative DNS
    test_alternative_dns()
    
    # Summary
    logger.info("")
    logger.info("=" * 50)
    logger.info("📋 SUMMARY")
    logger.info("=" * 50)
    
    if ip_address:
        logger.info("✅ DNS resolution working - connection should work")
        logger.info("The issue might be temporary network connectivity")
    else:
        logger.info("❌ DNS resolution failing")
        logger.info("This is likely a local network/DNS configuration issue")
        logger.info("")
        logger.info("🔧 IMMEDIATE SOLUTIONS:")
        logger.info("1. Restart your router/modem")
        logger.info("2. Change DNS to 8.8.8.8, 8.8.4.4")
        logger.info("3. Try mobile hotspot")
        logger.info("4. Contact your ISP if problem persists")


if __name__ == "__main__":
    main()
