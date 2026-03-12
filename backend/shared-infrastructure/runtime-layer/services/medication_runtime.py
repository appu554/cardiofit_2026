"""
Medication Runtime Service for KB7 Terminology Service
Service-specific runtime that leverages all integrated components
Implements complete medication calculation flow with query routing and caching
"""

import asyncio
from typing import Dict, List, Any, Optional
from datetime import datetime
import json
from loguru import logger


class MedicationRuntime:
    """
    Medication service runtime with all integrations
    Orchestrates complete medication calculation workflows
    """

    def __init__(self, query_router, cache_prefetcher):
        """
        Initialize Medication Runtime

        Args:
            query_router: Query Router instance
            cache_prefetcher: Cache Prefetcher instance
        """
        self.router = query_router
        self.cache = cache_prefetcher

        # Processing metrics
        self.metrics = {
            'calculations_performed': 0,
            'cache_hits': 0,
            'cache_misses': 0,
            'average_response_time': 0,
            'errors': 0,
            'start_time': datetime.utcnow()
        }

        logger.info("Medication Runtime initialized")

    async def calculate_medication_options(self, request: Dict[str, Any]) -> Dict[str, Any]:
        """
        Complete medication calculation flow

        Args:
            request: Medication calculation request

        Returns:
            Complete medication recommendation with scoring and safety analysis
        """
        start_time = datetime.utcnow()

        try:
            # Step 1: Create snapshot for consistency
            from ..query_router.router import QueryRequest, QueryPattern

            snapshot = await self.router.snapshot_manager.create_snapshot(
                service_id="medication",
                context=request
            )

            logger.info(f"Created snapshot {snapshot.id} for medication calculation")

            # Step 2: Get candidates from Neo4j semantic mesh
            candidates_query = QueryRequest(
                service_id="medication",
                pattern=QueryPattern.DRUG_ALTERNATIVES,
                params={
                    'drug_code': request.get('primary_drug', request.get('indication')),
                    'indication': request['indication'],
                    'patient_id': request['patient_id']
                },
                require_snapshot=True,
                context={'snapshot_id': snapshot.id}
            )

            candidates_response = await self.router.route_query(candidates_query)
            candidates = candidates_response.data or []

            # If no alternatives found, use provided candidates
            if not candidates:
                candidates = [{'rxnorm': drug, 'name': f'Drug {drug}'}
                            for drug in request.get('candidate_drugs', [])]

            # Step 3: Score using ClickHouse
            drug_codes = [c.get('rxnorm') for c in candidates if c.get('rxnorm')]

            if drug_codes:
                scores_query = QueryRequest(
                    service_id="medication",
                    pattern=QueryPattern.MEDICATION_SCORING,
                    params={
                        'drugs': drug_codes,
                        'indication': request['indication'],
                        'patient_context': request.get('patient_context', {}),
                        'snapshot_id': snapshot.id
                    },
                    require_snapshot=True,
                    context={'snapshot_id': snapshot.id}
                )

                scores_response = await self.router.route_query(scores_query)
                scores_data = scores_response.data

                if scores_response.cache_hit:
                    self.metrics['cache_hits'] += 1
                else:
                    self.metrics['cache_misses'] += 1

            else:
                scores_data = None

            # Step 4: Check patient-specific safety
            safety_query = QueryRequest(
                service_id="safety",
                pattern=QueryPattern.SAFETY_ANALYTICS,
                params={
                    'patient_id': request['patient_id'],
                    'medications': drug_codes,
                    'conditions': request.get('patient_conditions', [])
                },
                context={'snapshot_id': snapshot.id}
            )

            safety_response = await self.router.route_query(safety_query)

            # Step 5: Get current patient medications for interaction checking
            patient_meds_query = QueryRequest(
                service_id="medication",
                pattern=QueryPattern.PATIENT_MEDICATIONS,
                params={'patient_id': request['patient_id']},
                context={'snapshot_id': snapshot.id}
            )

            patient_meds_response = await self.router.route_query(patient_meds_query)
            current_medications = patient_meds_response.data or []

            # Step 6: Check drug interactions if patient has current medications
            interactions = []
            if current_medications:
                current_drug_codes = [med.get('rxnorm') for med in current_medications
                                    if med.get('rxnorm')]
                all_drug_codes = list(set(drug_codes + current_drug_codes))

                if len(all_drug_codes) > 1:
                    interaction_query = QueryRequest(
                        service_id="safety",
                        pattern=QueryPattern.DRUG_INTERACTIONS,
                        params={'drug_codes': all_drug_codes},
                        context={'snapshot_id': snapshot.id}
                    )

                    interaction_response = await self.router.route_query(interaction_query)
                    interactions = interaction_response.data or []

            # Step 7: Check contraindications
            contraindications = []
            patient_conditions = request.get('patient_conditions', [])

            if patient_conditions and drug_codes:
                for drug_code in drug_codes:
                    contra_query = QueryRequest(
                        service_id="safety",
                        pattern=QueryPattern.CONTRAINDICATIONS,
                        params={
                            'drug_code': drug_code,
                            'condition_codes': patient_conditions
                        },
                        context={'snapshot_id': snapshot.id}
                    )

                    contra_response = await self.router.route_query(contra_query)
                    if contra_response.data:
                        contraindications.extend(contra_response.data)

            # Step 8: Combine results
            result = self._combine_results(
                candidates, scores_data, safety_response.data,
                current_medications, interactions, contraindications,
                snapshot, request
            )

            # Update metrics
            response_time = (datetime.utcnow() - start_time).total_seconds()
            self._update_metrics(response_time)

            logger.info(f"Completed medication calculation for patient {request['patient_id']}")

            return result

        except Exception as e:
            self.metrics['errors'] += 1
            logger.error(f"Error in medication calculation: {e}")
            raise

    def _combine_results(self, candidates: List[Dict], scores_data: Any,
                        safety_data: Dict, current_medications: List[Dict],
                        interactions: List[Dict], contraindications: List[Dict],
                        snapshot, request: Dict) -> Dict[str, Any]:
        """
        Combine all analysis results into final recommendation

        Args:
            candidates: Drug candidates
            scores_data: Scoring data from ClickHouse
            safety_data: Safety analysis data
            current_medications: Patient's current medications
            interactions: Drug interactions
            contraindications: Contraindications
            snapshot: Consistency snapshot
            request: Original request

        Returns:
            Combined medication recommendation
        """
        # Convert scores_data to dict if it's a DataFrame
        scores_dict = {}
        if hasattr(scores_data, 'to_dict'):
            scores_records = scores_data.to_dict('records')
            scores_dict = {record['drug_rxnorm']: record for record in scores_records}

        # Combine candidate data with scores
        recommendations = []
        for candidate in candidates:
            drug_code = candidate.get('rxnorm')
            if not drug_code:
                continue

            # Get scoring data
            score_data = scores_dict.get(drug_code, {})

            # Check for interactions involving this drug
            drug_interactions = [
                interaction for interaction in interactions
                if interaction.get('drug1') == drug_code or interaction.get('drug2') == drug_code
            ]

            # Check for contraindications
            drug_contraindications = [
                contra for contra in contraindications
                if contra.get('drug_code') == drug_code
            ]

            # Calculate risk factors
            risk_factors = self._calculate_risk_factors(
                drug_interactions, drug_contraindications, safety_data
            )

            recommendation = {
                'drug_code': drug_code,
                'drug_name': candidate.get('name', 'Unknown'),
                'composite_score': score_data.get('composite_score', 0),
                'scoring_breakdown': {
                    'guideline_score': score_data.get('guideline_score', 0),
                    'safety_score': score_data.get('safety_score', 0),
                    'efficacy_score': score_data.get('efficacy_score', 0),
                    'cost_score': score_data.get('cost_score', 0),
                    'patient_preference_score': score_data.get('patient_preference_score', 0)
                },
                'risk_assessment': {
                    'overall_risk': risk_factors['overall_risk'],
                    'risk_level': risk_factors['risk_level'],
                    'risk_factors': risk_factors['factors']
                },
                'interactions': drug_interactions,
                'contraindications': drug_contraindications,
                'formulary_tier': score_data.get('formulary_tier'),
                'recommendation_strength': self._determine_recommendation_strength(
                    score_data.get('composite_score', 0), risk_factors['overall_risk']
                )
            }

            recommendations.append(recommendation)

        # Sort by composite score (highest first) and risk (lowest first)
        recommendations.sort(
            key=lambda x: (-x['composite_score'], x['risk_assessment']['overall_risk'])
        )

        return {
            'patient_id': request['patient_id'],
            'indication': request['indication'],
            'recommendations': recommendations,
            'current_medications': current_medications,
            'overall_safety_assessment': safety_data,
            'analysis_metadata': {
                'snapshot_id': snapshot.id,
                'total_candidates': len(candidates),
                'analyzed_candidates': len(recommendations),
                'total_interactions': len(interactions),
                'total_contraindications': len(contraindications),
                'calculation_timestamp': datetime.utcnow().isoformat()
            },
            'top_recommendation': recommendations[0] if recommendations else None
        }

    def _calculate_risk_factors(self, interactions: List[Dict],
                               contraindications: List[Dict],
                               safety_data: Dict) -> Dict[str, Any]:
        """Calculate overall risk factors for a drug"""
        risk_score = 0
        factors = []

        # Interaction risk
        if interactions:
            high_severity = [i for i in interactions if i.get('severity') == 'major']
            moderate_severity = [i for i in interactions if i.get('severity') == 'moderate']

            if high_severity:
                risk_score += 0.4
                factors.append(f"{len(high_severity)} major drug interactions")
            if moderate_severity:
                risk_score += 0.2
                factors.append(f"{len(moderate_severity)} moderate drug interactions")

        # Contraindication risk
        if contraindications:
            high_severity_contra = [c for c in contraindications if c.get('severity') == 'major']
            if high_severity_contra:
                risk_score += 0.5
                factors.append("Major contraindications present")
            else:
                risk_score += 0.3
                factors.append("Contraindications present")

        # Safety data risk
        if safety_data and 'risk_score' in safety_data:
            patient_risk = safety_data['risk_score']
            risk_score += patient_risk * 0.3
            if patient_risk > 0.6:
                factors.append("High patient-specific risk")

        # Determine risk level
        if risk_score < 0.3:
            risk_level = 'low'
        elif risk_score < 0.6:
            risk_level = 'moderate'
        else:
            risk_level = 'high'

        return {
            'overall_risk': min(risk_score, 1.0),  # Cap at 1.0
            'risk_level': risk_level,
            'factors': factors
        }

    def _determine_recommendation_strength(self, composite_score: float,
                                         risk_score: float) -> str:
        """Determine recommendation strength based on score and risk"""
        if composite_score > 0.8 and risk_score < 0.3:
            return 'strong'
        elif composite_score > 0.6 and risk_score < 0.5:
            return 'moderate'
        elif risk_score > 0.7:
            return 'not_recommended'
        else:
            return 'weak'

    def _update_metrics(self, response_time: float) -> None:
        """Update processing metrics"""
        self.metrics['calculations_performed'] += 1

        # Update average response time
        current_avg = self.metrics['average_response_time']
        count = self.metrics['calculations_performed']
        self.metrics['average_response_time'] = (
            (current_avg * (count - 1) + response_time) / count
        )

    async def get_patient_medication_profile(self, patient_id: str) -> Dict[str, Any]:
        """
        Get comprehensive medication profile for a patient

        Args:
            patient_id: Patient identifier

        Returns:
            Complete medication profile with safety analysis
        """
        from ..query_router.router import QueryRequest, QueryPattern

        # Get current medications
        meds_query = QueryRequest(
            service_id="medication",
            pattern=QueryPattern.PATIENT_MEDICATIONS,
            params={'patient_id': patient_id}
        )

        meds_response = await self.router.route_query(meds_query)
        medications = meds_response.data or []

        if not medications:
            return {
                'patient_id': patient_id,
                'medications': [],
                'interactions': [],
                'total_risk_score': 0,
                'risk_level': 'low',
                'recommendations': ['No current medications on file']
            }

        # Get drug codes
        drug_codes = [med.get('rxnorm') for med in medications if med.get('rxnorm')]

        # Check interactions
        interactions = []
        if len(drug_codes) > 1:
            interaction_query = QueryRequest(
                service_id="safety",
                pattern=QueryPattern.DRUG_INTERACTIONS,
                params={'drug_codes': drug_codes}
            )

            interaction_response = await self.router.route_query(interaction_query)
            interactions = interaction_response.data or []

        # Get safety analytics
        safety_query = QueryRequest(
            service_id="safety",
            pattern=QueryPattern.SAFETY_ANALYTICS,
            params={
                'patient_id': patient_id,
                'medications': drug_codes,
                'conditions': []  # Would get from patient record
            }
        )

        safety_response = await self.router.route_query(safety_query)
        safety_data = safety_response.data or {}

        # Generate recommendations
        recommendations = self._generate_profile_recommendations(
            medications, interactions, safety_data
        )

        return {
            'patient_id': patient_id,
            'medications': medications,
            'medication_count': len(medications),
            'interactions': interactions,
            'interaction_count': len(interactions),
            'safety_analysis': safety_data,
            'total_risk_score': safety_data.get('risk_score', 0),
            'risk_level': safety_data.get('risk_level', 'unknown'),
            'recommendations': recommendations,
            'profile_timestamp': datetime.utcnow().isoformat()
        }

    def _generate_profile_recommendations(self, medications: List[Dict],
                                        interactions: List[Dict],
                                        safety_data: Dict) -> List[str]:
        """Generate recommendations based on medication profile"""
        recommendations = []

        # Polypharmacy check
        if len(medications) > 5:
            recommendations.append(
                "Consider medication review - patient is on multiple medications"
            )

        # Interaction warnings
        major_interactions = [i for i in interactions if i.get('severity') == 'major']
        if major_interactions:
            recommendations.append(
                f"ALERT: {len(major_interactions)} major drug interactions detected"
            )

        moderate_interactions = [i for i in interactions
                               if i.get('severity') == 'moderate']
        if moderate_interactions:
            recommendations.append(
                f"Caution: {len(moderate_interactions)} moderate drug interactions"
            )

        # Risk level recommendations
        risk_score = safety_data.get('risk_score', 0)
        if risk_score > 0.7:
            recommendations.append("High risk profile - increase monitoring frequency")
        elif risk_score > 0.4:
            recommendations.append("Moderate risk - standard monitoring recommended")

        if not recommendations:
            recommendations.append("No significant safety concerns identified")

        return recommendations

    async def get_runtime_statistics(self) -> Dict[str, Any]:
        """Get medication runtime statistics"""
        uptime = datetime.utcnow() - self.metrics['start_time']

        cache_hit_rate = 0
        if self.metrics['cache_hits'] + self.metrics['cache_misses'] > 0:
            cache_hit_rate = (
                self.metrics['cache_hits'] /
                (self.metrics['cache_hits'] + self.metrics['cache_misses'])
            )

        return {
            'runtime_metrics': self.metrics,
            'uptime_seconds': uptime.total_seconds(),
            'cache_hit_rate': cache_hit_rate,
            'calculations_per_hour': (
                self.metrics['calculations_performed'] / max(uptime.total_seconds() / 3600, 1)
            ),
            'error_rate': (
                self.metrics['errors'] /
                max(self.metrics['calculations_performed'], 1)
            ),
            'timestamp': datetime.utcnow().isoformat()
        }

    async def test_medication_runtime(self) -> Dict[str, Any]:
        """Test medication runtime with sample data"""
        logger.info("Testing medication runtime with sample data")

        test_request = {
            'patient_id': 'test-patient-001',
            'indication': 'I10',  # Essential hypertension
            'candidate_drugs': ['197361', '197362'],  # Lisinopril, alternative
            'patient_conditions': ['I10', 'E11.9'],  # Hypertension, diabetes
            'patient_context': {
                'age': 65,
                'gender': 'male',
                'weight_kg': 80,
                'creatinine': 1.2,
                'elderly': True
            }
        }

        try:
            result = await self.calculate_medication_options(test_request)

            return {
                'test_status': 'success',
                'patient_id': result.get('patient_id'),
                'recommendations_count': len(result.get('recommendations', [])),
                'top_recommendation': result.get('top_recommendation', {}).get('drug_name'),
                'snapshot_id': result.get('analysis_metadata', {}).get('snapshot_id'),
                'timestamp': datetime.utcnow().isoformat()
            }

        except Exception as e:
            return {
                'test_status': 'error',
                'error_message': str(e),
                'timestamp': datetime.utcnow().isoformat()
            }


# CLI script functionality
if __name__ == "__main__":
    import sys
    import argparse

    async def main():
        parser = argparse.ArgumentParser(description='Medication Runtime Service')
        parser.add_argument('--test', action='store_true',
                          help='Run test with sample data')

        args = parser.parse_args()

        if args.test:
            # Initialize components for testing
            from ..query_router.router import QueryRouter
            from ..cache_warming.cdc_subscriber import CachePrefetcher

            config = {
                'neo4j': {
                    'neo4j_uri': 'bolt://localhost:7687',
                    'neo4j_user': 'neo4j',
                    'neo4j_password': 'kb7password'
                },
                'clickhouse': {
                    'host': 'localhost',
                    'port': 9000,
                    'database': 'kb7_analytics_test'
                }
            }

            query_router = QueryRouter(config)
            await query_router.initialize_clients()

            cache_prefetcher = CachePrefetcher(config)

            # Test medication runtime
            medication_runtime = MedicationRuntime(query_router, cache_prefetcher)
            result = await medication_runtime.test_medication_runtime()

            print(json.dumps(result, indent=2))
        else:
            print("Use --test to run medication runtime tests")

    asyncio.run(main())