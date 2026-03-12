# Clinical Assertion Engine (CAE) Architecture

## **CAE-Specific Data Infrastructure**

The CAE has its own dedicated data infrastructure separate from the main microservices:

```
┌─────────────────────────────────────────────────────────────┐
│                Clinical Assertion Engine (CAE)             │
│                        Port: 8027                          │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐ │
│  │   gRPC Server   │  │  GraphQL API    │  │  REST API    │ │
│  │   (Clinical)    │  │  (Federation)   │  │  (Health)    │ │
│  └─────────────────┘  └─────────────────┘  └──────────────┘ │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐ │
│  │ Medication      │  │ Dosing          │  │Contraindication│ │
│  │ Interaction     │  │ Calculator      │  │   Reasoner   │ │
│  │ Reasoner        │  │                 │  │              │ │
│  └─────────────────┘  └─────────────────┘  └──────────────┘ │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐                   │
│  │ Patient Context │  │ Clinical        │                   │
│  │ Assembler       │  │ Knowledge       │                   │
│  │ (GraphDB)       │  │ Cache (Redis)   │                   │
│  └─────────────────┘  └─────────────────┘                   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    CAE Data Layer                          │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐ │
│  │   CAE GraphDB   │  │   CAE Redis     │  │ Clinical     │ │
│  │   Port: 7201    │  │   Port: 6380    │  │ Knowledge    │ │
│  │                 │  │                 │  │ Files        │ │
│  │ Patient Context │  │ • Context Cache │  │ • Drug DB    │ │
│  │ • Demographics  │  │ • Knowledge     │  │ • Guidelines │ │
│  │ • Conditions    │  │ • Session Data  │  │ • Evidence   │ │
│  │ • Medications   │  │ • Rate Limits   │  │              │ │
│  │ • Allergies     │  │                 │  │              │ │
│  │ • Lab Results   │  │                 │  │              │ │
│  │ • Timeline      │  │                 │  │              │ │
│  └─────────────────┘  └─────────────────┘  └──────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

## **Why CAE Needs Its Own GraphDB & Redis**

### **1. CAE GraphDB (Port: 7201)**
- **Purpose**: Store and query patient clinical context for reasoning
- **Data Model**: Clinical ontologies, patient relationships, temporal data
- **Queries**: `DESCRIBE :Patient123` to get complete clinical picture
- **Content**:
  - Patient demographics and clinical history
  - Active conditions and diagnoses
  - Current medications and dosing history
  - Allergies and adverse reactions
  - Laboratory results and trends
  - Clinical timeline and events

### **2. CAE Redis (Port: 6380)**
- **Purpose**: High-performance caching for clinical reasoning
- **Use Cases**:
  - **Patient Context Cache**: Cache assembled patient contexts (TTL: 5 minutes)
  - **Clinical Knowledge Cache**: Cache drug interaction rules, dosing guidelines
  - **Session Management**: Track reasoning sessions and correlation IDs
  - **Rate Limiting**: Prevent abuse of clinical reasoning APIs
  - **Performance Optimization**: Cache expensive SPARQL query results

## **Data Flow Architecture**

```
1. gRPC Request → CAE Service
2. CAE → Check Redis Cache for Patient Context
3. If Cache Miss → CAE → Query GraphDB (SPARQL)
4. GraphDB → Return Patient Clinical Context
5. CAE → Cache Context in Redis
6. CAE → Apply Clinical Reasoning Logic
7. CAE → Return Clinical Assertions
```

## **Integration with Microservices**

The CAE **does NOT directly connect** to other microservices' databases. Instead:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ Patient Service │    │ Medication Svc  │    │ Condition Svc   │
│ (Port: 8003)    │    │ (Port: 8009)    │    │ (Port: 8010)    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────────────────────────────────────────────────┐
│              Data Synchronization Layer                    │
│              (ETL Pipeline / Event Streaming)              │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    CAE GraphDB                             │
│              (Clinical Context Store)                      │
└─────────────────────────────────────────────────────────────┘
```

## **Why This Architecture?**

### **Separation of Concerns**
- **CAE GraphDB**: Optimized for clinical reasoning queries
- **Main Services**: Optimized for CRUD operations
- **No Direct DB Access**: CAE doesn't query other services' databases

### **Performance**
- **Redis Caching**: Sub-millisecond context retrieval
- **GraphDB**: Optimized SPARQL queries for clinical relationships
- **Dedicated Resources**: CAE has its own data infrastructure

### **Clinical Safety**
- **Immutable Context**: CAE works with snapshots of patient data
- **Audit Trail**: All clinical reasoning decisions are logged
- **Version Control**: Clinical knowledge base versioning

## **Implementation Status**

### ✅ **Completed**
- gRPC Server infrastructure
- Clinical reasoning logic (interactions, dosing, contraindications)
- Basic patient context assembly framework

### ❌ **Missing (Critical)**
- **CAE GraphDB Setup**: Dedicated GraphDB instance for CAE
- **CAE Redis Setup**: Dedicated Redis instance for CAE
- **Data Synchronization**: ETL pipeline to populate CAE GraphDB
- **SPARQL Queries**: Real clinical context queries
- **Redis Caching**: Performance optimization layer

## **Next Implementation Steps**

1. **Set up CAE GraphDB** (Port: 7201)
2. **Set up CAE Redis** (Port: 6380)
3. **Implement data synchronization** from main services to CAE GraphDB
4. **Create real SPARQL queries** for patient context
5. **Implement Redis caching** for performance
6. **Add monitoring and metrics** for CAE data layer
