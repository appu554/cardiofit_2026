#!/usr/bin/env python3
"""
Data Source Validation Script
Validates that all required real data sources are available before running pipeline
NO FALLBACK DATA - All sources must be authentic
"""

import asyncio
import sys
import aiohttp
from pathlib import Path
import structlog

# Add src to path
sys.path.insert(0, str(Path(__file__).parent / "src"))

from core.config import settings

# Configure logging
structlog.configure(
    processors=[
        structlog.stdlib.filter_by_level,
        structlog.stdlib.add_logger_name,
        structlog.stdlib.add_log_level,
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.JSONRenderer()
    ],
    context_class=dict,
    logger_factory=structlog.stdlib.LoggerFactory(),
    wrapper_class=structlog.stdlib.BoundLogger,
    cache_logger_on_first_use=True,
)

logger = structlog.get_logger(__name__)


class DataSourceValidator:
    """Validates availability of real clinical data sources"""
    
    def __init__(self):
        self.validation_results = {
            'rxnorm': {'available': False, 'errors': []},
            'drugbank': {'available': False, 'errors': []},
            'umls': {'available': False, 'errors': []},
            'snomed': {'available': False, 'errors': []},
            'loinc': {'available': False, 'errors': []},
            'crediblemeds': {'available': False, 'errors': []},
            'ahrq': {'available': False, 'errors': []},
            'openfda': {'available': False, 'errors': []},
            'graphdb': {'available': False, 'errors': []}
        }

    async def validate_specific_sources(self, sources: list) -> bool:
        """Validate only specific clinical data sources"""
        logger.info(f"🔍 Validating specific clinical data sources: {', '.join(sources)} - NO FALLBACKS ALLOWED")

        # Map of source names to validation methods
        validation_methods = {
            'rxnorm': self.validate_rxnorm,
            'drugbank': self.validate_drugbank,
            'umls': self.validate_umls,
            'snomed': self.validate_snomed,
            'loinc': self.validate_loinc,
            'crediblemeds': self.validate_crediblemeds,
            'ahrq': self.validate_ahrq,
            'openfda': self.validate_openfda
        }

        # Always validate GraphDB
        await self.validate_graphdb()

        # Validate only requested sources
        for source in sources:
            if source in validation_methods:
                await validation_methods[source]()
            else:
                logger.warning(f"Unknown source: {source}")

        # Print results for requested sources only
        self.print_specific_validation_summary(sources)

        # Check if all requested sources are available
        requested_results = {k: v for k, v in self.validation_results.items()
                           if k in sources or k == 'graphdb'}

        all_available = all(
            result['available'] for result in requested_results.values()
        )

        return all_available

    async def validate_all_sources(self) -> bool:
        """Validate all data sources"""
        logger.info("🔍 Validating real clinical data sources - NO FALLBACKS ALLOWED")
        
        # Validate each source
        await self.validate_rxnorm()
        await self.validate_drugbank()
        await self.validate_umls()
        await self.validate_snomed()
        await self.validate_loinc()
        await self.validate_crediblemeds()
        await self.validate_ahrq()
        await self.validate_openfda()
        await self.validate_graphdb()
        
        # Summary
        self.print_validation_summary()
        
        # Return overall status
        all_available = all(
            result['available'] for result in self.validation_results.values()
        )
        
        return all_available
    
    async def validate_rxnorm(self):
        """Validate RxNorm data availability"""
        logger.info("📋 Validating RxNorm data source...")
        
        try:
            # Check if RxNorm RRF files exist locally
            data_dir = Path(settings.DATA_DIR) / "rxnorm" / "rrf"
            required_files = settings.RXNORM_PROCESS_TABLES
            
            missing_files = []
            for rrf_file in required_files:
                file_path = data_dir / rrf_file
                if not file_path.exists():
                    missing_files.append(rrf_file)
            
            if missing_files:
                error_msg = f"Missing RxNorm RRF files: {missing_files}"
                self.validation_results['rxnorm']['errors'].append(error_msg)
                self.validation_results['rxnorm']['errors'].append(
                    f"Download from: {settings.RXNORM_DOWNLOAD_URL}"
                )
                logger.error("❌ RxNorm validation failed", missing_files=missing_files)
            else:
                self.validation_results['rxnorm']['available'] = True
                logger.info("✅ RxNorm data files found")
        
        except Exception as e:
            error_msg = f"RxNorm validation error: {str(e)}"
            self.validation_results['rxnorm']['errors'].append(error_msg)
            logger.error("❌ RxNorm validation failed", error=str(e))
    
    async def validate_drugbank(self):
        """Validate DrugBank data availability"""
        logger.info("💊 Validating DrugBank Academic data source...")
        
        try:
            # Check if DrugBank XML file exists
            data_dir = Path(settings.DATA_DIR) / "drugbank"
            xml_files = [
                "drugbank_full_database.xml",
                "drugbank_all_full_database.xml.zip"
            ]
            
            xml_found = False
            for xml_file in xml_files:
                if (data_dir / xml_file).exists():
                    xml_found = True
                    break
            
            if not xml_found:
                error_msg = "DrugBank XML file not found"
                self.validation_results['drugbank']['errors'].append(error_msg)
                self.validation_results['drugbank']['errors'].append(
                    "Manual download required from: https://go.drugbank.com/releases/latest#open-data"
                )
                self.validation_results['drugbank']['errors'].append(
                    "1. Create free academic account"
                )
                self.validation_results['drugbank']['errors'].append(
                    "2. Download 'All drugs (XML)' file"
                )
                self.validation_results['drugbank']['errors'].append(
                    f"3. Save to: {data_dir}"
                )
                logger.error("❌ DrugBank validation failed - manual download required")
            else:
                self.validation_results['drugbank']['available'] = True
                logger.info("✅ DrugBank XML file found")
        
        except Exception as e:
            error_msg = f"DrugBank validation error: {str(e)}"
            self.validation_results['drugbank']['errors'].append(error_msg)
            logger.error("❌ DrugBank validation failed", error=str(e))

    async def validate_umls(self):
        """Validate UMLS Metathesaurus data availability"""
        logger.info("🏥 Validating UMLS Metathesaurus data source...")

        try:
            # Check if UMLS ZIP file exists
            data_dir = Path(settings.DATA_DIR) / "umls"
            zip_files = [
                "umls-metathesaurus-full.zip",
                "umls-current-metathesaurus-full.zip"
            ]

            zip_found = False
            for zip_file in zip_files:
                if (data_dir / zip_file).exists():
                    zip_found = True
                    break

            if not zip_found:
                error_msg = "UMLS Metathesaurus ZIP file not found"
                self.validation_results['umls']['errors'].append(error_msg)
                self.validation_results['umls']['errors'].append(
                    "UMLS license required from: https://uts.nlm.nih.gov/uts/"
                )
                self.validation_results['umls']['errors'].append(
                    "1. Create UTS account and accept license"
                )
                self.validation_results['umls']['errors'].append(
                    "2. Download UMLS Metathesaurus Files"
                )
                self.validation_results['umls']['errors'].append(
                    f"3. Save to: {data_dir}"
                )
                logger.error("❌ UMLS validation failed - license and download required")
            else:
                self.validation_results['umls']['available'] = True
                logger.info("✅ UMLS Metathesaurus ZIP file found")

        except Exception as e:
            error_msg = f"UMLS validation error: {str(e)}"
            self.validation_results['umls']['errors'].append(error_msg)
            logger.error("❌ UMLS validation failed", error=str(e))

    async def validate_snomed(self):
        """Validate SNOMED CT data availability"""
        logger.info("🩺 Validating SNOMED CT data source...")

        try:
            # Check if SNOMED CT ZIP file exists
            data_dir = Path(settings.DATA_DIR) / "snomed"
            zip_files = [
                "SnomedCT_InternationalRF2_PRODUCTION.zip",
                "SnomedCT_InternationalRF2_PRODUCTION_*.zip"
            ]

            zip_found = False
            for zip_pattern in zip_files:
                if list(data_dir.glob(zip_pattern)):
                    zip_found = True
                    break

            if not zip_found:
                error_msg = "SNOMED CT ZIP file not found"
                self.validation_results['snomed']['errors'].append(error_msg)
                self.validation_results['snomed']['errors'].append(
                    "SNOMED International license required: https://www.snomed.org/snomed-ct/get-snomed"
                )
                self.validation_results['snomed']['errors'].append(
                    "1. Check licensing requirements for your country"
                )
                self.validation_results['snomed']['errors'].append(
                    "2. Download SNOMED CT International Edition RF2"
                )
                self.validation_results['snomed']['errors'].append(
                    f"3. Save to: {data_dir}"
                )
                logger.error("❌ SNOMED CT validation failed - license and download required")
            else:
                self.validation_results['snomed']['available'] = True
                logger.info("✅ SNOMED CT ZIP file found")

        except Exception as e:
            error_msg = f"SNOMED CT validation error: {str(e)}"
            self.validation_results['snomed']['errors'].append(error_msg)
            logger.error("❌ SNOMED CT validation failed", error=str(e))

    async def validate_loinc(self):
        """Validate LOINC data availability"""
        logger.info("🧪 Validating LOINC data source...")

        try:
            # Check if LOINC ZIP file exists
            data_dir = Path(settings.DATA_DIR) / "loinc"
            zip_files = [
                "Loinc_current.zip",
                "Loinc_*.zip"
            ]

            zip_found = False
            for zip_pattern in zip_files:
                if list(data_dir.glob(zip_pattern)):
                    zip_found = True
                    break

            if not zip_found:
                error_msg = "LOINC ZIP file not found"
                self.validation_results['loinc']['errors'].append(error_msg)
                self.validation_results['loinc']['errors'].append(
                    "LOINC license agreement required: https://loinc.org/license/"
                )
                self.validation_results['loinc']['errors'].append(
                    "1. Create free LOINC account"
                )
                self.validation_results['loinc']['errors'].append(
                    "2. Accept LOINC License Agreement"
                )
                self.validation_results['loinc']['errors'].append(
                    "3. Download LOINC Table File (CSV)"
                )
                self.validation_results['loinc']['errors'].append(
                    f"4. Save to: {data_dir}"
                )
                logger.error("❌ LOINC validation failed - license and download required")
            else:
                self.validation_results['loinc']['available'] = True
                logger.info("✅ LOINC ZIP file found")

        except Exception as e:
            error_msg = f"LOINC validation error: {str(e)}"
            self.validation_results['loinc']['errors'].append(error_msg)
            logger.error("❌ LOINC validation failed", error=str(e))
    
    async def validate_crediblemeds(self):
        """Validate CredibleMeds data availability"""
        logger.info("⚡ Validating CredibleMeds data source...")
        
        try:
            # Test CredibleMeds website accessibility
            timeout = aiohttp.ClientTimeout(total=10)
            async with aiohttp.ClientSession(timeout=timeout) as session:
                try:
                    async with session.get("https://www.crediblemeds.org") as response:
                        if response.status == 200:
                            self.validation_results['crediblemeds']['available'] = True
                            logger.info("✅ CredibleMeds website accessible")
                        else:
                            error_msg = f"CredibleMeds website returned status {response.status}"
                            self.validation_results['crediblemeds']['errors'].append(error_msg)
                            logger.error("❌ CredibleMeds validation failed", status=response.status)
                
                except aiohttp.ClientError as e:
                    error_msg = f"Cannot access CredibleMeds website: {str(e)}"
                    self.validation_results['crediblemeds']['errors'].append(error_msg)
                    logger.error("❌ CredibleMeds validation failed", error=str(e))
        
        except Exception as e:
            error_msg = f"CredibleMeds validation error: {str(e)}"
            self.validation_results['crediblemeds']['errors'].append(error_msg)
            logger.error("❌ CredibleMeds validation failed", error=str(e))
    
    async def validate_ahrq(self):
        """Validate AHRQ CDS Connect data availability"""
        logger.info("🏥 Validating AHRQ CDS Connect data source...")
        
        try:
            # Test AHRQ API accessibility
            timeout = aiohttp.ClientTimeout(total=10)
            async with aiohttp.ClientSession(timeout=timeout) as session:
                try:
                    api_url = "https://cds.ahrq.gov/cdsconnect/api/artifacts"
                    async with session.get(api_url) as response:
                        if response.status == 200:
                            data = await response.json()
                            if 'artifacts' in data or isinstance(data, list):
                                self.validation_results['ahrq']['available'] = True
                                logger.info("✅ AHRQ CDS Connect API accessible")
                            else:
                                error_msg = "AHRQ API returned unexpected format"
                                self.validation_results['ahrq']['errors'].append(error_msg)
                                logger.error("❌ AHRQ validation failed - unexpected format")
                        else:
                            error_msg = f"AHRQ API returned status {response.status}"
                            self.validation_results['ahrq']['errors'].append(error_msg)
                            logger.error("❌ AHRQ validation failed", status=response.status)
                
                except aiohttp.ClientError as e:
                    error_msg = f"Cannot access AHRQ API: {str(e)}"
                    self.validation_results['ahrq']['errors'].append(error_msg)
                    logger.error("❌ AHRQ validation failed", error=str(e))
        
        except Exception as e:
            error_msg = f"AHRQ validation error: {str(e)}"
            self.validation_results['ahrq']['errors'].append(error_msg)
            logger.error("❌ AHRQ validation failed", error=str(e))

    async def validate_openfda(self):
        """Validate OpenFDA API availability"""
        logger.info("💊 Validating OpenFDA API data source...")

        try:
            # Test OpenFDA API accessibility
            timeout = aiohttp.ClientTimeout(total=10)
            async with aiohttp.ClientSession(timeout=timeout) as session:
                try:
                    # Test with a simple query
                    test_url = "https://api.fda.gov/drug/event.json?search=receivedate:[20230101+TO+20230102]&limit=1"
                    async with session.get(test_url) as response:
                        if response.status == 200:
                            data = await response.json()
                            if 'results' in data or 'error' in data:
                                self.validation_results['openfda']['available'] = True
                                logger.info("✅ OpenFDA API accessible")
                            else:
                                error_msg = "OpenFDA API returned unexpected format"
                                self.validation_results['openfda']['errors'].append(error_msg)
                                logger.error("❌ OpenFDA validation failed - unexpected format")
                        elif response.status == 429:
                            # Rate limited but API is working
                            self.validation_results['openfda']['available'] = True
                            logger.info("✅ OpenFDA API accessible (rate limited)")
                        else:
                            error_msg = f"OpenFDA API returned status {response.status}"
                            self.validation_results['openfda']['errors'].append(error_msg)
                            self.validation_results['openfda']['errors'].append(
                                "Consider getting API key: https://open.fda.gov/apis/authentication/"
                            )
                            logger.error("❌ OpenFDA validation failed", status=response.status)

                except aiohttp.ClientError as e:
                    error_msg = f"Cannot access OpenFDA API: {str(e)}"
                    self.validation_results['openfda']['errors'].append(error_msg)
                    self.validation_results['openfda']['errors'].append(
                        "Check internet connectivity and firewall settings"
                    )
                    logger.error("❌ OpenFDA validation failed", error=str(e))

        except Exception as e:
            error_msg = f"OpenFDA validation error: {str(e)}"
            self.validation_results['openfda']['errors'].append(error_msg)
            logger.error("❌ OpenFDA validation failed", error=str(e))

    async def validate_graphdb(self):
        """Validate database connectivity (GraphDB or Neo4j Cloud)"""
        from core.config import settings

        database_type = getattr(settings, 'DATABASE_TYPE', 'neo4j').lower()

        if database_type == 'neo4j':
            logger.info("🌐 Validating Neo4j Cloud connectivity...")
            await self._validate_neo4j_cloud()
        else:
            logger.info("🗄️ Validating GraphDB connectivity...")
            await self._validate_graphdb_local()

    async def _validate_neo4j_cloud(self):
        """Validate Neo4j Cloud connectivity"""
        try:
            from core.database_factory import validate_database_connection

            result = await validate_database_connection()

            if result["status"] == "connected":
                self.validation_results['graphdb']['available'] = True
                logger.info("✅ Neo4j Cloud connection successful")

                # Log database info
                db_info = result.get("database_info", {})
                logger.info("Neo4j Cloud instance details",
                          uri=db_info.get("uri", "unknown"),
                          database=db_info.get("database", "unknown"))
            else:
                error_msg = result.get("error", "Connection failed")
                self.validation_results['graphdb']['errors'].append(f"Neo4j Cloud: {error_msg}")
                logger.error("❌ Neo4j Cloud connection failed", error=error_msg)

        except Exception as e:
            error_msg = f"Neo4j Cloud validation error: {str(e)}"
            self.validation_results['graphdb']['errors'].append(error_msg)
            logger.error("❌ Neo4j Cloud validation failed", error=str(e))

    async def _validate_graphdb_local(self):
        """Validate local GraphDB connectivity"""
        try:
            from core.graphdb_client import GraphDBClient

            client = GraphDBClient()
            await client.connect()

            # Test connection
            connected = await client.test_connection()

            if connected:
                self.validation_results['graphdb']['available'] = True
                logger.info("✅ GraphDB connection successful")
            else:
                error_msg = "GraphDB connection test failed"
                self.validation_results['graphdb']['errors'].append(error_msg)
                logger.error("❌ GraphDB validation failed")
            
            await client.disconnect()
        
        except Exception as e:
            error_msg = f"GraphDB validation error: {str(e)}"
            self.validation_results['graphdb']['errors'].append(error_msg)
            logger.error("❌ GraphDB validation failed", error=str(e))
    
    def print_validation_summary(self):
        """Print comprehensive validation summary"""
        logger.info("📊 VALIDATION SUMMARY - REAL DATA SOURCES ONLY")
        
        total_sources = len(self.validation_results)
        available_sources = sum(1 for result in self.validation_results.values() if result['available'])
        
        print("\n" + "="*80)
        print("🔍 CLINICAL DATA SOURCE VALIDATION RESULTS")
        print("="*80)
        print(f"📈 Overall Status: {available_sources}/{total_sources} sources available")
        print()
        
        for source_name, result in self.validation_results.items():
            status_icon = "✅" if result['available'] else "❌"
            status_text = "AVAILABLE" if result['available'] else "UNAVAILABLE"
            
            print(f"{status_icon} {source_name.upper()}: {status_text}")
            
            if result['errors']:
                for error in result['errors']:
                    print(f"   ⚠️  {error}")
            print()
        
        if available_sources == total_sources:
            print("🎉 ALL DATA SOURCES VALIDATED - PIPELINE READY TO RUN")
        else:
            print("🚨 PIPELINE CANNOT RUN - MISSING REQUIRED DATA SOURCES")
            print("   📋 Action Required:")
            print("   1. Download/fix missing data sources")
            print("   2. Re-run validation")
            print("   3. NO FALLBACK DATA WILL BE USED")
        
        print("="*80)

    def print_specific_validation_summary(self, sources: list):
        """Print validation summary for specific sources only"""
        logger.info("📊 VALIDATION SUMMARY - SPECIFIC SOURCES ONLY")

        # Filter results to only requested sources + GraphDB
        requested_sources = sources + ['graphdb']
        filtered_results = {k: v for k, v in self.validation_results.items()
                          if k in requested_sources}

        total_sources = len(filtered_results)
        available_sources = sum(1 for result in filtered_results.values() if result['available'])

        print("\n" + "="*80)
        print("🔍 CLINICAL DATA SOURCE VALIDATION RESULTS")
        print("="*80)
        print(f"📈 Overall Status: {available_sources}/{total_sources} sources available")
        print()

        for source_name, result in filtered_results.items():
            status_icon = "✅" if result['available'] else "❌"
            status_text = "AVAILABLE" if result['available'] else "UNAVAILABLE"

            print(f"{status_icon} {source_name.upper()}: {status_text}")

            if result['errors']:
                for error in result['errors']:
                    print(f"   ⚠️  {error}")
            print()

        if available_sources == total_sources:
            print("🎉 ALL REQUESTED DATA SOURCES VALIDATED - PIPELINE READY TO RUN")
        else:
            print("🚨 PIPELINE CANNOT RUN - MISSING REQUIRED DATA SOURCES")
            print("   📋 Action Required:")
            print("   1. Download/fix missing data sources")
            print("   2. Re-run validation")
            print("   3. NO FALLBACK DATA WILL BE USED")

        print("="*80)

    def print_specific_validation_summary(self, sources: list):
        """Print validation summary for specific sources only"""
        logger.info("📊 VALIDATION SUMMARY - SPECIFIC SOURCES ONLY")

        # Filter results to only requested sources + GraphDB
        requested_sources = sources + ['graphdb']
        filtered_results = {k: v for k, v in self.validation_results.items()
                          if k in requested_sources}

        total_sources = len(filtered_results)
        available_sources = sum(1 for result in filtered_results.values() if result['available'])

        print("\n" + "="*80)
        print("🔍 CLINICAL DATA SOURCE VALIDATION RESULTS")
        print("="*80)
        print(f"📈 Overall Status: {available_sources}/{total_sources} sources available")
        print()

        for source_name, result in filtered_results.items():
            status_icon = "✅" if result['available'] else "❌"
            status_text = "AVAILABLE" if result['available'] else "UNAVAILABLE"

            print(f"{status_icon} {source_name.upper()}: {status_text}")

            if result['errors']:
                for error in result['errors']:
                    print(f"   ⚠️  {error}")
            print()

        if available_sources == total_sources:
            print("🎉 ALL REQUESTED DATA SOURCES VALIDATED - PIPELINE READY TO RUN")
        else:
            print("🚨 PIPELINE CANNOT RUN - MISSING REQUIRED DATA SOURCES")
            print("   📋 Action Required:")
            print("   1. Download/fix missing data sources")
            print("   2. Re-run validation")
            print("   3. NO FALLBACK DATA WILL BE USED")

        print("="*80)


async def main():
    """Main validation function"""
    validator = DataSourceValidator()
    
    try:
        all_valid = await validator.validate_all_sources()
        
        if all_valid:
            logger.info("🎉 All data sources validated successfully")
            sys.exit(0)
        else:
            logger.error("🚨 Data source validation failed")
            sys.exit(1)
    
    except Exception as e:
        logger.error("💥 Validation script failed", error=str(e))
        sys.exit(1)


if __name__ == "__main__":
    asyncio.run(main())
