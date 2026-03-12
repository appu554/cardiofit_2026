#!/usr/bin/env python3
"""
Vaidshala Phase 5: Atomiser Registry

Tracks existing CQL content to determine what needs Atomiser extraction.
Only invoke Atomiser when no existing CQL covers the clinical topic.

Usage:
    from atomiser import AtomiserRegistry

    registry = AtomiserRegistry()
    if registry.needs_atomiser("beta-blocker titration HFrEF"):
        # Use Atomiser
    else:
        cql_path = registry.get_existing_cql("beta-blocker titration HFrEF")
"""

import json
import re
from pathlib import Path
from typing import Dict, List, Optional, Set
from dataclasses import dataclass, field
from datetime import datetime


@dataclass
class CQLEntry:
    """Registry entry for a CQL library."""
    library_name: str
    file_path: str
    topics: List[str]
    conditions: List[str]
    medications: List[str]
    source_guideline: str
    version: str = "1.0.0"
    last_updated: str = field(default_factory=lambda: datetime.utcnow().isoformat())


class AtomiserRegistry:
    """
    Registry of existing CQL content.

    Purpose: Prevent duplicate extraction by tracking what CQL already exists.

    RULES:
    1. Always check registry before invoking Atomiser
    2. If matching CQL exists, use it instead of Atomiser
    3. Register new extractions after SME approval
    """

    def __init__(self, cql_base_path: str = None, registry_file: str = None):
        """
        Initialize registry.

        Args:
            cql_base_path: Base path to CQL files
            registry_file: Path to registry JSON file (auto-generated if not exists)
        """
        self.cql_base_path = Path(cql_base_path) if cql_base_path else self._find_cql_base()
        self.registry_file = Path(registry_file) if registry_file else self.cql_base_path / ".cql_registry.json"

        self.entries: Dict[str, CQLEntry] = {}
        self.topic_index: Dict[str, Set[str]] = {}  # topic -> set of library names
        self.condition_index: Dict[str, Set[str]] = {}  # condition -> set of library names
        self.medication_index: Dict[str, Set[str]] = {}  # medication -> set of library names

        self._load_or_build_registry()

    def _find_cql_base(self) -> Path:
        """Find the CQL base path."""
        # Try common locations
        candidates = [
            Path(__file__).parent.parent.parent / "clinical-knowledge-core",
            Path.cwd() / "vaidshala" / "clinical-knowledge-core",
            Path.home() / "Downloads" / "cardiofit" / "vaidshala" / "clinical-knowledge-core"
        ]

        for path in candidates:
            if path.exists():
                return path

        return Path.cwd()

    def _load_or_build_registry(self):
        """Load registry from file or build from CQL scan."""
        if self.registry_file.exists():
            self._load_registry()
        else:
            self._build_registry()
            self._save_registry()

    def _load_registry(self):
        """Load registry from JSON file."""
        with open(self.registry_file) as f:
            data = json.load(f)

        for name, entry_data in data.get("entries", {}).items():
            entry = CQLEntry(**entry_data)
            self.entries[name] = entry
            self._index_entry(entry)

    def _save_registry(self):
        """Save registry to JSON file."""
        data = {
            "version": "1.0",
            "last_updated": datetime.utcnow().isoformat(),
            "entry_count": len(self.entries),
            "entries": {name: self._entry_to_dict(entry) for name, entry in self.entries.items()}
        }

        self.registry_file.parent.mkdir(parents=True, exist_ok=True)
        with open(self.registry_file, 'w') as f:
            json.dump(data, f, indent=2)

    def _entry_to_dict(self, entry: CQLEntry) -> Dict:
        """Convert CQLEntry to dictionary."""
        return {
            "library_name": entry.library_name,
            "file_path": entry.file_path,
            "topics": entry.topics,
            "conditions": entry.conditions,
            "medications": entry.medications,
            "source_guideline": entry.source_guideline,
            "version": entry.version,
            "last_updated": entry.last_updated
        }

    def _build_registry(self):
        """Build registry by scanning CQL files."""
        if not self.cql_base_path.exists():
            return

        for cql_file in self.cql_base_path.rglob("*.cql"):
            entry = self._parse_cql_file(cql_file)
            if entry:
                self.entries[entry.library_name] = entry
                self._index_entry(entry)

    def _parse_cql_file(self, file_path: Path) -> Optional[CQLEntry]:
        """Parse a CQL file to extract metadata."""
        try:
            with open(file_path, 'r', encoding='utf-8') as f:
                content = f.read()
        except Exception:
            return None

        # Extract library name
        lib_match = re.search(r"library\s+([A-Za-z][A-Za-z0-9_]+)", content)
        if not lib_match:
            return None

        library_name = lib_match.group(1)

        # Extract version
        version_match = re.search(r"version\s+'([^']+)'", content)
        version = version_match.group(1) if version_match else "1.0.0"

        # Determine source guideline from path
        source_guideline = self._infer_source(file_path)

        # Extract topics, conditions, medications from content
        topics = self._extract_topics(content, file_path)
        conditions = self._extract_conditions(content)
        medications = self._extract_medications(content)

        return CQLEntry(
            library_name=library_name,
            file_path=str(file_path.relative_to(self.cql_base_path)),
            topics=topics,
            conditions=conditions,
            medications=medications,
            source_guideline=source_guideline,
            version=version
        )

    def _infer_source(self, file_path: Path) -> str:
        """Infer source guideline from file path."""
        path_str = str(file_path).lower()

        if "cdc-opioid" in path_str:
            return "CDC-OPIOID-2022"
        elif "cms-ecqm" in path_str:
            # Extract CMS number
            match = re.search(r"cms(\d+)", path_str)
            return f"CMS-{match.group(1)}" if match else "CMS-ECQM"
        elif "who" in path_str:
            if "anc" in path_str:
                return "WHO-ANC"
            elif "hiv" in path_str:
                return "WHO-HIV"
            elif "immun" in path_str:
                return "WHO-IMMUNIZATION"
            return "WHO"
        elif "sepsis" in path_str:
            return "SSC-2021"
        elif "hf" in path_str or "heart" in path_str:
            return "ACC-AHA-HF-2022"
        elif "vte" in path_str:
            return "CHEST-VTE"
        elif "diabet" in path_str:
            return "ADA-SOC-2024"

        return "UNKNOWN"

    def _extract_topics(self, content: str, file_path: Path) -> List[str]:
        """Extract clinical topics from CQL content."""
        topics = []

        # From file name
        file_name = file_path.stem.lower()
        if "opioid" in file_name:
            topics.append("opioid prescribing")
        if "mme" in file_name or "morphine" in file_name:
            topics.append("MME calculation")
        if "titrat" in file_name:
            topics.append("medication titration")
        if "prophylax" in file_name:
            topics.append("prophylaxis")
        if "screen" in file_name:
            topics.append("screening")
        if "monitor" in file_name:
            topics.append("monitoring")

        # From content keywords
        topic_patterns = [
            (r'sepsis|septic', "sepsis management"),
            (r'heart\s*failure|hfref|hfpef', "heart failure"),
            (r'diabetes|glycemic|a1c|hba1c', "diabetes management"),
            (r'anticoagulat|vte|thromboembol', "anticoagulation"),
            (r'immuniz|vaccin', "immunization"),
            (r'antenatal|pregnan', "antenatal care"),
            (r'hypertension|blood\s*pressure', "hypertension"),
            (r'statin|cholesterol|lipid', "lipid management"),
        ]

        for pattern, topic in topic_patterns:
            if re.search(pattern, content, re.IGNORECASE):
                if topic not in topics:
                    topics.append(topic)

        return topics

    def _extract_conditions(self, content: str) -> List[str]:
        """Extract clinical conditions from CQL content."""
        conditions = []

        # Look for condition references
        condition_patterns = [
            r'\"([^\"]*(?:disorder|disease|syndrome|failure|deficiency)[^\"]*)\"',
            r'valueset\s+\"([^\"]+)\"',
        ]

        for pattern in condition_patterns:
            matches = re.findall(pattern, content, re.IGNORECASE)
            for match in matches:
                # Clean and add
                condition = match.strip()
                if len(condition) > 5 and condition not in conditions:
                    conditions.append(condition[:100])  # Truncate long names

        return conditions[:20]  # Limit to 20

    def _extract_medications(self, content: str) -> List[str]:
        """Extract medications from CQL content."""
        medications = []

        # Common medication patterns in CQL
        med_patterns = [
            r'\"([^\"]*(?:medication|drug|therapy|agent)[^\"]*)\"',
            r'valueset\s+\"([^\"]*(?:oid|rx|med)[^\"]*)\"',
        ]

        for pattern in med_patterns:
            matches = re.findall(pattern, content, re.IGNORECASE)
            for match in matches:
                medication = match.strip()
                if len(medication) > 3 and medication not in medications:
                    medications.append(medication[:100])

        # Also look for specific drug class mentions
        drug_classes = [
            'ARNi', 'ACEi', 'ARB', 'beta-blocker', 'SGLT2i', 'MRA',
            'statin', 'opioid', 'benzodiazepine', 'anticoagulant',
            'antibiotic', 'vasopressor', 'insulin', 'metformin'
        ]

        for drug in drug_classes:
            if re.search(drug, content, re.IGNORECASE):
                if drug not in medications:
                    medications.append(drug)

        return medications[:20]

    def _index_entry(self, entry: CQLEntry):
        """Add entry to search indices."""
        lib_name = entry.library_name

        for topic in entry.topics:
            topic_lower = topic.lower()
            if topic_lower not in self.topic_index:
                self.topic_index[topic_lower] = set()
            self.topic_index[topic_lower].add(lib_name)

        for condition in entry.conditions:
            cond_lower = condition.lower()
            if cond_lower not in self.condition_index:
                self.condition_index[cond_lower] = set()
            self.condition_index[cond_lower].add(lib_name)

        for medication in entry.medications:
            med_lower = medication.lower()
            if med_lower not in self.medication_index:
                self.medication_index[med_lower] = set()
            self.medication_index[med_lower].add(lib_name)

    def needs_atomiser(self, query: str, threshold: float = 0.5) -> bool:
        """
        Check if a topic needs Atomiser (no existing CQL).

        Args:
            query: Natural language query for clinical topic
            threshold: Match confidence threshold

        Returns:
            True if Atomiser needed (no matching CQL found)
        """
        matches = self.search(query)
        if not matches:
            return True

        # Check if any match exceeds threshold
        return all(score < threshold for _, score in matches)

    def search(self, query: str, limit: int = 5) -> List[tuple[str, float]]:
        """
        Search for matching CQL libraries.

        Args:
            query: Search query
            limit: Max results

        Returns:
            List of (library_name, confidence_score) tuples
        """
        query_lower = query.lower()
        scores: Dict[str, float] = {}

        # Search topics
        for topic, libraries in self.topic_index.items():
            if topic in query_lower or query_lower in topic:
                for lib in libraries:
                    scores[lib] = scores.get(lib, 0) + 0.4

        # Search conditions
        for condition, libraries in self.condition_index.items():
            if any(word in condition for word in query_lower.split()):
                for lib in libraries:
                    scores[lib] = scores.get(lib, 0) + 0.3

        # Search medications
        for medication, libraries in self.medication_index.items():
            if medication in query_lower or query_lower in medication:
                for lib in libraries:
                    scores[lib] = scores.get(lib, 0) + 0.3

        # Sort by score
        results = sorted(scores.items(), key=lambda x: x[1], reverse=True)
        return results[:limit]

    def get_existing_cql(self, query: str) -> Optional[str]:
        """
        Get path to existing CQL that matches query.

        Args:
            query: Search query

        Returns:
            File path to best matching CQL, or None
        """
        matches = self.search(query, limit=1)
        if matches and matches[0][1] >= 0.5:
            lib_name = matches[0][0]
            entry = self.entries.get(lib_name)
            if entry:
                return entry.file_path
        return None

    def register_extraction(self, entry: CQLEntry):
        """
        Register a new CQL extraction.

        Called after SME approval of Atomiser output.
        """
        self.entries[entry.library_name] = entry
        self._index_entry(entry)
        self._save_registry()

    def get_stats(self) -> Dict:
        """Get registry statistics."""
        return {
            "total_libraries": len(self.entries),
            "total_topics": len(self.topic_index),
            "total_conditions": len(self.condition_index),
            "total_medications": len(self.medication_index),
            "by_source": self._count_by_source()
        }

    def _count_by_source(self) -> Dict[str, int]:
        """Count libraries by source guideline."""
        counts: Dict[str, int] = {}
        for entry in self.entries.values():
            source = entry.source_guideline
            counts[source] = counts.get(source, 0) + 1
        return counts

    def refresh(self):
        """Rebuild registry from CQL files."""
        self.entries.clear()
        self.topic_index.clear()
        self.condition_index.clear()
        self.medication_index.clear()
        self._build_registry()
        self._save_registry()


def main():
    """Demo the Atomiser Registry."""
    print("=" * 60)
    print("Phase 5: Atomiser Registry Demo")
    print("=" * 60)

    registry = AtomiserRegistry()

    print(f"\n--- Registry Stats ---")
    stats = registry.get_stats()
    print(json.dumps(stats, indent=2))

    # Test queries
    test_queries = [
        "beta-blocker titration heart failure",
        "opioid MME calculation",
        "sepsis fluid resuscitation",
        "diabetes eye exam screening",
        "pregnancy immunization schedule",
        "novel gene therapy protocol"  # Should need atomiser
    ]

    print("\n--- Query Tests ---")
    for query in test_queries:
        needs = registry.needs_atomiser(query)
        matches = registry.search(query, limit=3)

        print(f"\nQuery: '{query}'")
        print(f"  Needs Atomiser: {needs}")
        if matches:
            print(f"  Best matches:")
            for lib, score in matches:
                print(f"    - {lib}: {score:.2f}")


if __name__ == "__main__":
    main()
