#!/usr/bin/env python3
"""
Test gRPC Port Connectivity

This script tests if the gRPC server is actually running on port 50051.
"""

import socket
import asyncio
import grpc
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def test_port_connectivity(host, port):
    """Test if a port is open and accepting connections"""
    try:
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.settimeout(5)
        result = sock.connect_ex((host, port))
        sock.close()
        return result == 0
    except Exception as e:
        logger.error(f"Port connectivity test failed: {e}")
        return False

async def test_grpc_connectivity(host, port):
    """Test gRPC connectivity"""
    try:
        channel = grpc.aio.insecure_channel(f"{host}:{port}")
        
        # Test if channel is ready
        await asyncio.wait_for(channel.channel_ready(), timeout=5.0)
        logger.info(f"gRPC channel to {host}:{port} is ready")
        
        await channel.close()
        return True
        
    except asyncio.TimeoutError:
        logger.error(f"gRPC channel to {host}:{port} timed out")
        return False
    except Exception as e:
        logger.error(f"gRPC connectivity test failed: {e}")
        return False

async def main():
    """Main test function"""
    host = "localhost"
    ports_to_test = [8040, 50051]
    
    logger.info("Testing port connectivity...")
    
    for port in ports_to_test:
        logger.info(f"\nTesting port {port}:")
        
        # Test basic port connectivity
        port_open = test_port_connectivity(host, port)
        logger.info(f"  Port {port} open: {port_open}")
        
        if port_open:
            if port == 50051:
                # Test gRPC connectivity for gRPC port
                grpc_ready = await test_grpc_connectivity(host, port)
                logger.info(f"  gRPC ready on port {port}: {grpc_ready}")
            else:
                # Test HTTP connectivity for HTTP port
                try:
                    import aiohttp
                    async with aiohttp.ClientSession() as session:
                        async with session.get(f"http://{host}:{port}/health", timeout=5) as response:
                            logger.info(f"  HTTP response on port {port}: {response.status}")
                except Exception as e:
                    logger.error(f"  HTTP test failed on port {port}: {e}")

if __name__ == "__main__":
    asyncio.run(main())
