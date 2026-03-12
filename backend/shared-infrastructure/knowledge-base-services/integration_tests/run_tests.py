#!/usr/bin/env python3
"""
Simple test runner for cross-service integration tests
"""

import asyncio
import sys
import os
import argparse
import logging
from datetime import datetime

# Add the current directory to path to import test modules
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from cross_service_tests import CrossServiceTestRunner

def setup_logging(verbose=False):
    """Setup logging configuration"""
    level = logging.DEBUG if verbose else logging.INFO
    
    logging.basicConfig(
        level=level,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
        handlers=[
            logging.StreamHandler(sys.stdout),
            logging.FileHandler(f'integration_tests_{datetime.now().strftime("%Y%m%d_%H%M%S")}.log')
        ]
    )

async def main():
    """Main entry point"""
    parser = argparse.ArgumentParser(description="Run cross-service integration tests")
    parser.add_argument("--scenario", help="Run specific scenario by name")
    parser.add_argument("--list-scenarios", action="store_true", help="List available scenarios")
    parser.add_argument("--report", help="Output report file path")
    parser.add_argument("--verbose", action="store_true", help="Enable verbose logging")
    parser.add_argument("--quick", action="store_true", help="Run quick health checks only")
    
    args = parser.parse_args()
    
    setup_logging(args.verbose)
    logger = logging.getLogger(__name__)
    
    runner = CrossServiceTestRunner()
    
    if args.list_scenarios:
        print("\nAvailable test scenarios:")
        for i, scenario in enumerate(runner.scenarios, 1):
            print(f"{i}. {scenario.name}")
            print(f"   Description: {scenario.description}")
            print(f"   Services: {', '.join(scenario.services)}")
            print(f"   Timeout: {scenario.timeout}s")
            print()
        return 0
    
    if args.quick:
        # Run quick health checks only
        logger.info("Running quick health checks...")
        
        runner.session = aiohttp.ClientSession()
        try:
            health_tasks = []
            for service_id in runner.services.keys():
                health_tasks.append(runner.check_service_health(service_id))
            
            health_results = await asyncio.gather(*health_tasks)
            
            print("\nService Health Status:")
            print("-" * 50)
            
            healthy_count = 0
            for health in health_results:
                status_icon = "✅" if health["status"] == "healthy" else "❌"
                print(f"{status_icon} {health['service']:<30} {health['status']}")
                if health["status"] == "healthy":
                    healthy_count += 1
            
            print("-" * 50)
            print(f"Healthy services: {healthy_count}/{len(health_results)}")
            
            if healthy_count == len(health_results):
                print("✅ All services are healthy!")
                return 0
            else:
                print("❌ Some services are unhealthy")
                return 1
                
        finally:
            await runner.session.close()
    
    if args.scenario:
        # Run specific scenario
        scenario = next((s for s in runner.scenarios if s.name == args.scenario), None)
        if not scenario:
            logger.error(f"Scenario '{args.scenario}' not found")
            available = [s.name for s in runner.scenarios]
            logger.info(f"Available scenarios: {', '.join(available)}")
            return 1
        
        logger.info(f"Running scenario: {scenario.name}")
        
        runner.session = aiohttp.ClientSession()
        try:
            result = await runner.run_scenario(scenario)
            
            if result["success"]:
                print(f"✅ Scenario '{scenario.name}' PASSED")
                print(f"   Execution time: {result['execution_time']:.2f}s")
                print(f"   Steps completed: {len(result.get('step_results', []))}")
            else:
                print(f"❌ Scenario '{scenario.name}' FAILED")
                print(f"   Error: {result.get('error', 'Unknown error')}")
                print(f"   Execution time: {result['execution_time']:.2f}s")
            
            return 0 if result["success"] else 1
            
        finally:
            await runner.session.close()
    else:
        # Run all scenarios
        logger.info("Running all integration test scenarios...")
        
        results = await runner.run_all_scenarios()
        
        # Generate report
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        report_file = args.report or f"integration_test_report_{timestamp}.md"
        report_content = runner.generate_report(results, report_file)
        
        # Print summary to console
        print("\n" + "="*60)
        print("INTEGRATION TEST SUMMARY")
        print("="*60)
        print(f"Total scenarios: {results['total_scenarios']}")
        print(f"Successful: {results['successful_scenarios']}")
        print(f"Failed: {results['failed_scenarios']}")
        print(f"Success rate: {results['success_rate']:.1f}%")
        print(f"Total execution time: {results['total_execution_time']:.2f}s")
        print(f"Report saved to: {report_file}")
        
        if results["failed_scenarios"] > 0:
            print("\nFailed scenarios:")
            for scenario_result in results["scenario_results"]:
                if not scenario_result["success"]:
                    print(f"  ❌ {scenario_result['scenario']}: {scenario_result.get('error', 'Unknown error')}")
        
        print("="*60)
        
        return 0 if results["failed_scenarios"] == 0 else 1

if __name__ == "__main__":
    try:
        import aiohttp
    except ImportError:
        print("Error: aiohttp is required. Install with: pip install aiohttp")
        sys.exit(1)
    
    try:
        exit_code = asyncio.run(main())
        sys.exit(exit_code)
    except KeyboardInterrupt:
        print("\nTest execution interrupted by user")
        sys.exit(1)
    except Exception as e:
        print(f"Unexpected error: {e}")
        sys.exit(1)