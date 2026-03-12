"""
Cypher Query Builder for Neo4j Graph Projector
Builds Cypher queries from GraphMutation objects
"""
import sys
from pathlib import Path
from typing import Dict, Any, List

import structlog

# Add shared module to path
shared_module_path = Path(__file__).parent.parent.parent.parent / "module8-shared"
sys.path.insert(0, str(shared_module_path))

from module8_shared.models.events import GraphMutation, Relationship

logger = structlog.get_logger(__name__)


class CypherQueryBuilder:
    """
    Builds Cypher queries for graph mutations

    Supports:
    - MERGE operations (upsert nodes)
    - CREATE operations (create new nodes)
    - Relationship creation between nodes
    """

    @staticmethod
    def build_merge_node(mutation: GraphMutation) -> tuple[str, Dict[str, Any]]:
        """
        Build MERGE query for node upsert

        Args:
            mutation: GraphMutation with node data

        Returns:
            Tuple of (cypher_query, parameters)
        """
        node_type = mutation.node_type
        node_id = mutation.node_id
        properties = mutation.node_properties

        # Build property assignments (excluding nodeId which is in MERGE clause)
        prop_assignments = []
        parameters = {"nodeId": node_id}

        for key, value in properties.items():
            param_name = f"prop_{key}"
            prop_assignments.append(f"n.{key} = ${param_name}")
            parameters[param_name] = value

        # Add timestamp
        parameters["timestamp"] = mutation.timestamp
        prop_assignments.append("n.lastUpdated = $timestamp")

        # Build query
        query = f"""
            MERGE (n:{node_type} {{nodeId: $nodeId}})
            SET {', '.join(prop_assignments)}
            RETURN n
        """

        logger.debug(
            "Built MERGE node query",
            node_type=node_type,
            node_id=node_id,
            properties_count=len(properties)
        )

        return query, parameters

    @staticmethod
    def build_create_node(mutation: GraphMutation) -> tuple[str, Dict[str, Any]]:
        """
        Build CREATE query for new node

        Args:
            mutation: GraphMutation with node data

        Returns:
            Tuple of (cypher_query, parameters)
        """
        node_type = mutation.node_type
        node_id = mutation.node_id
        properties = mutation.node_properties

        # Build property map
        parameters = {"nodeId": node_id, "timestamp": mutation.timestamp}

        for key, value in properties.items():
            param_name = f"prop_{key}"
            parameters[param_name] = value

        # Build properties clause
        prop_list = ["nodeId: $nodeId", "lastUpdated: $timestamp"]
        for key in properties.keys():
            prop_list.append(f"{key}: $prop_{key}")

        query = f"""
            CREATE (n:{node_type} {{{', '.join(prop_list)}}})
            RETURN n
        """

        logger.debug(
            "Built CREATE node query",
            node_type=node_type,
            node_id=node_id,
            properties_count=len(properties)
        )

        return query, parameters

    @staticmethod
    def build_relationship(
        mutation: GraphMutation,
        rel: Relationship
    ) -> tuple[str, Dict[str, Any]]:
        """
        Build MERGE query for relationship

        Args:
            mutation: GraphMutation with source node data
            rel: Relationship specification

        Returns:
            Tuple of (cypher_query, parameters)
        """
        source_type = mutation.node_type
        source_id = mutation.node_id
        rel_type = rel.relation_type
        target_type = rel.target_node_type
        target_id = rel.target_node_id
        rel_props = rel.relationship_properties

        # Build parameters
        parameters = {
            "sourceId": source_id,
            "targetId": target_id,
            "timestamp": mutation.timestamp,
        }

        # Build relationship properties
        prop_assignments = ["r.lastUpdated = $timestamp"]
        for key, value in rel_props.items():
            param_name = f"rel_{key}"
            prop_assignments.append(f"r.{key} = ${param_name}")
            parameters[param_name] = value

        # Build query
        query = f"""
            MATCH (source:{source_type} {{nodeId: $sourceId}})
            MATCH (target:{target_type} {{nodeId: $targetId}})
            MERGE (source)-[r:{rel_type}]->(target)
            SET {', '.join(prop_assignments)}
            RETURN r
        """

        logger.debug(
            "Built relationship query",
            relation_type=rel_type,
            source=f"{source_type}:{source_id}",
            target=f"{target_type}:{target_id}",
            properties_count=len(rel_props)
        )

        return query, parameters

    @staticmethod
    def build_batch_merge_nodes(
        mutations: List[GraphMutation]
    ) -> tuple[str, Dict[str, Any]]:
        """
        Build batch MERGE query for multiple nodes of the same type

        Args:
            mutations: List of GraphMutation objects

        Returns:
            Tuple of (cypher_query, parameters)
        """
        if not mutations:
            return "", {}

        # Group by node type
        by_type: Dict[str, List[GraphMutation]] = {}
        for mutation in mutations:
            node_type = mutation.node_type
            if node_type not in by_type:
                by_type[node_type] = []
            by_type[node_type].append(mutation)

        # Build UNWIND query for each type
        queries = []
        parameters = {}

        for node_type, type_mutations in by_type.items():
            # Build list of node data
            nodes_data = []
            for mutation in type_mutations:
                node_dict = {
                    "nodeId": mutation.node_id,
                    "timestamp": mutation.timestamp,
                    **mutation.node_properties
                }
                nodes_data.append(node_dict)

            param_name = f"nodes_{node_type}"
            parameters[param_name] = nodes_data

            # Build UNWIND query
            query = f"""
                UNWIND ${param_name} AS nodeData
                MERGE (n:{node_type} {{nodeId: nodeData.nodeId}})
                SET n += nodeData,
                    n.lastUpdated = nodeData.timestamp
            """
            queries.append(query)

        combined_query = "\n".join(queries)

        logger.debug(
            "Built batch MERGE query",
            node_types=list(by_type.keys()),
            total_nodes=len(mutations)
        )

        return combined_query, parameters

    @staticmethod
    def get_constraint_queries() -> List[str]:
        """
        Get list of constraint creation queries for graph schema

        Returns:
            List of Cypher constraint queries
        """
        return [
            # Unique constraints on nodeId
            "CREATE CONSTRAINT patient_id IF NOT EXISTS FOR (p:Patient) REQUIRE p.nodeId IS UNIQUE",
            "CREATE CONSTRAINT event_id IF NOT EXISTS FOR (e:ClinicalEvent) REQUIRE e.nodeId IS UNIQUE",
            "CREATE CONSTRAINT condition_id IF NOT EXISTS FOR (c:Condition) REQUIRE c.nodeId IS UNIQUE",
            "CREATE CONSTRAINT medication_id IF NOT EXISTS FOR (m:Medication) REQUIRE m.nodeId IS UNIQUE",
            "CREATE CONSTRAINT procedure_id IF NOT EXISTS FOR (p:Procedure) REQUIRE p.nodeId IS UNIQUE",
            "CREATE CONSTRAINT department_id IF NOT EXISTS FOR (d:Department) REQUIRE d.nodeId IS UNIQUE",
            "CREATE CONSTRAINT device_id IF NOT EXISTS FOR (d:Device) REQUIRE d.nodeId IS UNIQUE",

            # Indexes for performance
            "CREATE INDEX patient_last_updated IF NOT EXISTS FOR (p:Patient) ON (p.lastUpdated)",
            "CREATE INDEX event_timestamp IF NOT EXISTS FOR (e:ClinicalEvent) ON (e.timestamp)",
            "CREATE INDEX event_patient IF NOT EXISTS FOR (e:ClinicalEvent) ON (e.patientId)",
            "CREATE INDEX condition_patient IF NOT EXISTS FOR (c:Condition) ON (c.patientId)",
            "CREATE INDEX medication_patient IF NOT EXISTS FOR (m:Medication) ON (m.patientId)",
        ]

    @staticmethod
    def get_example_queries() -> Dict[str, str]:
        """
        Get example queries for common graph operations

        Returns:
            Dictionary of query name to Cypher query
        """
        return {
            "patient_journey": """
                MATCH (p:Patient {nodeId: $patientId})-[:HAS_EVENT]->(e:ClinicalEvent)
                RETURN p, e
                ORDER BY e.timestamp
            """,

            "temporal_sequence": """
                MATCH path = (e1:ClinicalEvent)-[:NEXT_EVENT*]->(e2:ClinicalEvent)
                WHERE e1.patientId = $patientId
                RETURN path
                LIMIT 100
            """,

            "clinical_pathway": """
                MATCH (p:Patient)-[:HAS_CONDITION]->(c:Condition),
                      (p)-[:HAS_EVENT]->(e:ClinicalEvent)-[:TRIGGERED_BY]->(c)
                WHERE p.nodeId = $patientId
                RETURN p, c, e
                ORDER BY e.timestamp
            """,

            "patient_medications": """
                MATCH (p:Patient {nodeId: $patientId})-[:PRESCRIBED]->(m:Medication)
                RETURN p, m
                ORDER BY m.startDate DESC
            """,

            "department_events": """
                MATCH (d:Department {nodeId: $departmentId})<-[:LOCATED_IN]-(p:Patient)-[:HAS_EVENT]->(e:ClinicalEvent)
                RETURN d, p, e
                ORDER BY e.timestamp DESC
                LIMIT 50
            """,

            "device_measurements": """
                MATCH (dev:Device {nodeId: $deviceId})<-[:MEASURED_BY]-(e:ClinicalEvent)
                RETURN dev, e
                ORDER BY e.timestamp DESC
                LIMIT 100
            """,

            "patient_summary": """
                MATCH (p:Patient {nodeId: $patientId})
                OPTIONAL MATCH (p)-[:HAS_CONDITION]->(c:Condition)
                OPTIONAL MATCH (p)-[:PRESCRIBED]->(m:Medication)
                OPTIONAL MATCH (p)-[:HAS_EVENT]->(e:ClinicalEvent)
                RETURN p,
                       collect(DISTINCT c) as conditions,
                       collect(DISTINCT m) as medications,
                       count(DISTINCT e) as event_count
            """,
        }
