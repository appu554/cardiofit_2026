#!/usr/bin/env python3
"""
Automated Service Startup for Stage 1 & Stage 2
Starts both services and waits for them to be ready for testing
"""

import subprocess
import time
import requests
import signal
import sys
import os
from threading import Thread

class Colors:
    GREEN = '\033[92m'
    RED = '\033[91m'
    YELLOW = '\033[93m'
    BLUE = '\033[94m'
    BOLD = '\033[1m'
    END = '\033[0m'

class ServiceManager:
    def __init__(self):
        self.stage1_process = None
        self.stage2_process = None
        self.services_ready = False
        
    def log(self, message: str, level: str = "INFO"):
        color = Colors.GREEN if level == "INFO" else Colors.RED if level == "ERROR" else Colors.YELLOW
        print(f"{color}[{level}] {message}{Colors.END}")
        
    def check_prerequisites(self):
        """Check if required tools are installed"""
        self.log("🔍 Checking prerequisites...")
        
        # Check Java
        try:
            result = subprocess.run(['java', '-version'], capture_output=True, text=True)
            if result.returncode == 0:
                self.log("✅ Java found")
            else:
                self.log("❌ Java not found", "ERROR")
                return False
        except FileNotFoundError:
            self.log("❌ Java not found", "ERROR")
            return False
            
        # Check Maven
        try:
            result = subprocess.run(['mvn', '-version'], capture_output=True, text=True)
            if result.returncode == 0:
                self.log("✅ Maven found")
            else:
                self.log("❌ Maven not found", "ERROR")
                return False
        except FileNotFoundError:
            self.log("❌ Maven not found", "ERROR")
            return False
            
        # Check Python
        try:
            result = subprocess.run(['python3', '--version'], capture_output=True, text=True)
            if result.returncode == 0:
                self.log("✅ Python 3 found")
            else:
                self.log("❌ Python 3 not found", "ERROR")
                return False
        except FileNotFoundError:
            self.log("❌ Python 3 not found", "ERROR")
            return False
            
        return True
        
    def setup_stage1(self):
        """Build Stage 1 if needed"""
        self.log("🔨 Setting up Stage 1...")
        
        if not os.path.exists("stage1-validator-enricher"):
            self.log("❌ Stage 1 directory not found", "ERROR")
            return False
            
        try:
            # Change to Stage 1 directory and build
            os.chdir("stage1-validator-enricher")
            
            self.log("Building Stage 1...")
            result = subprocess.run(['mvn', 'clean', 'package', '-DskipTests'], 
                                  capture_output=True, text=True)
            
            if result.returncode == 0:
                self.log("✅ Stage 1 built successfully")
                os.chdir("..")
                return True
            else:
                self.log(f"❌ Stage 1 build failed: {result.stderr}", "ERROR")
                os.chdir("..")
                return False
                
        except Exception as e:
            self.log(f"❌ Stage 1 setup error: {e}", "ERROR")
            os.chdir("..")
            return False
            
    def setup_stage2(self):
        """Setup Stage 2 dependencies"""
        self.log("🔨 Setting up Stage 2...")
        
        if not os.path.exists("stage2-storage-fanout"):
            self.log("❌ Stage 2 directory not found", "ERROR")
            return False
            
        try:
            # Change to Stage 2 directory and install dependencies
            os.chdir("stage2-storage-fanout")
            
            self.log("Installing Stage 2 dependencies...")
            result = subprocess.run(['pip', 'install', '-r', 'requirements.txt'], 
                                  capture_output=True, text=True)
            
            if result.returncode == 0:
                self.log("✅ Stage 2 dependencies installed")
                os.chdir("..")
                return True
            else:
                self.log(f"❌ Stage 2 setup failed: {result.stderr}", "ERROR")
                os.chdir("..")
                return False
                
        except Exception as e:
            self.log(f"❌ Stage 2 setup error: {e}", "ERROR")
            os.chdir("..")
            return False
            
    def start_stage1(self):
        """Start Stage 1 service"""
        self.log("🚀 Starting Stage 1 (Validator & Enricher)...")
        
        try:
            os.chdir("stage1-validator-enricher")
            
            # Set environment variables
            env = os.environ.copy()
            env.update({
                'KAFKA_BOOTSTRAP_SERVERS': 'pkc-619z3.us-east1.gcp.confluent.cloud:9092',
                'KAFKA_API_KEY': 'LGJ3AQ2L6VRPW4S2',
                'KAFKA_API_SECRET': '2hYzQLmG1XGyQ9oLZcjwAIBdAZUS6N4JoWD8oZQhk0qVBmmyVVHU7TqoLjYef0kl',
                'REDIS_HOST': 'localhost',
                'PATIENT_SERVICE_URL': 'http://localhost:8003/api/v1/patient'
            })
            
            # Start Stage 1
            self.stage1_process = subprocess.Popen(
                ['mvn', 'spring-boot:run', '-Dspring-boot.run.profiles=dev'],
                env=env,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True
            )
            
            os.chdir("..")
            self.log("✅ Stage 1 starting...")
            return True
            
        except Exception as e:
            self.log(f"❌ Failed to start Stage 1: {e}", "ERROR")
            os.chdir("..")
            return False
            
    def start_stage2(self):
        """Start Stage 2 service"""
        self.log("🚀 Starting Stage 2 (Storage Fan-Out)...")
        
        try:
            os.chdir("stage2-storage-fanout")
            
            # Set environment variables
            env = os.environ.copy()
            env.update({
                'KAFKA_BOOTSTRAP_SERVERS': 'pkc-619z3.us-east1.gcp.confluent.cloud:9092',
                'KAFKA_API_KEY': 'LGJ3AQ2L6VRPW4S2',
                'KAFKA_API_SECRET': '2hYzQLmG1XGyQ9oLZcjwAIBdAZUS6N4JoWD8oZQhk0qVBmmyVVHU7TqoLjYef0kl',
                'PORT': '8042',
                'DEBUG': 'true',
                'MONGODB_ENABLED': 'true',
                'FHIR_STORE_ENABLED': 'false',
                'ELASTICSEARCH_ENABLED': 'false'
            })
            
            # Start Stage 2
            self.stage2_process = subprocess.Popen(
                ['python', '-m', 'uvicorn', 'app.main:app', '--host', '0.0.0.0', '--port', '8042'],
                env=env,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True
            )
            
            os.chdir("..")
            self.log("✅ Stage 2 starting...")
            return True
            
        except Exception as e:
            self.log(f"❌ Failed to start Stage 2: {e}", "ERROR")
            os.chdir("..")
            return False
            
    def wait_for_services(self, timeout=120):
        """Wait for both services to be ready"""
        self.log("⏳ Waiting for services to be ready...")
        
        start_time = time.time()
        stage1_ready = False
        stage2_ready = False
        
        while time.time() - start_time < timeout:
            # Check Stage 1
            if not stage1_ready:
                try:
                    response = requests.get("http://localhost:8041/api/v1/health", timeout=2)
                    if response.status_code == 200:
                        stage1_ready = True
                        self.log("✅ Stage 1 is ready!")
                except:
                    pass
                    
            # Check Stage 2
            if not stage2_ready:
                try:
                    response = requests.get("http://localhost:8042/api/v1/health", timeout=2)
                    if response.status_code == 200:
                        stage2_ready = True
                        self.log("✅ Stage 2 is ready!")
                except:
                    pass
                    
            if stage1_ready and stage2_ready:
                self.services_ready = True
                self.log("🎉 Both services are ready for testing!")
                return True
                
            time.sleep(2)
            
        self.log("⏰ Timeout waiting for services to be ready", "ERROR")
        return False
        
    def stop_services(self):
        """Stop both services"""
        self.log("🛑 Stopping services...")
        
        if self.stage1_process:
            self.stage1_process.terminate()
            try:
                self.stage1_process.wait(timeout=10)
                self.log("✅ Stage 1 stopped")
            except subprocess.TimeoutExpired:
                self.stage1_process.kill()
                self.log("🔪 Stage 1 force killed")
                
        if self.stage2_process:
            self.stage2_process.terminate()
            try:
                self.stage2_process.wait(timeout=10)
                self.log("✅ Stage 2 stopped")
            except subprocess.TimeoutExpired:
                self.stage2_process.kill()
                self.log("🔪 Stage 2 force killed")
                
    def run_automated_tests(self):
        """Run the automated test suite"""
        self.log("🧪 Running automated test suite...")
        
        try:
            result = subprocess.run(['python3', 'automated-test-suite.py'], 
                                  capture_output=False, text=True)
            return result.returncode == 0
        except Exception as e:
            self.log(f"❌ Test suite failed: {e}", "ERROR")
            return False
            
    def signal_handler(self, signum, frame):
        """Handle shutdown signals"""
        self.log("\n🛑 Received shutdown signal, stopping services...")
        self.stop_services()
        sys.exit(0)
        
    def run_full_test_cycle(self):
        """Run the complete automated test cycle"""
        # Register signal handlers
        signal.signal(signal.SIGINT, self.signal_handler)
        signal.signal(signal.SIGTERM, self.signal_handler)
        
        try:
            self.log("🤖 Starting Automated Test Cycle for Stage 1 & Stage 2")
            self.log("=" * 60)
            
            # Step 1: Check prerequisites
            if not self.check_prerequisites():
                return False
                
            # Step 2: Setup services
            if not self.setup_stage1():
                return False
            if not self.setup_stage2():
                return False
                
            # Step 3: Start services
            if not self.start_stage1():
                return False
            if not self.start_stage2():
                return False
                
            # Step 4: Wait for services to be ready
            if not self.wait_for_services():
                return False
                
            # Step 5: Run automated tests
            test_success = self.run_automated_tests()
            
            # Step 6: Keep services running for manual inspection
            if test_success:
                self.log("🎉 Automated tests completed successfully!")
                self.log("Services are still running for manual inspection.")
                self.log("Press Ctrl+C to stop services and exit.")
                
                # Keep running until interrupted
                try:
                    while True:
                        time.sleep(1)
                except KeyboardInterrupt:
                    pass
            else:
                self.log("❌ Automated tests failed", "ERROR")
                
            return test_success
            
        except Exception as e:
            self.log(f"❌ Test cycle failed: {e}", "ERROR")
            return False
        finally:
            self.stop_services()

def main():
    """Main execution"""
    manager = ServiceManager()
    success = manager.run_full_test_cycle()
    sys.exit(0 if success else 1)

if __name__ == "__main__":
    main()
