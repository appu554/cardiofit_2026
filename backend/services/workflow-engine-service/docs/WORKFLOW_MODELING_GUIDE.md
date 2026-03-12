# Workflow Modeling Guide

## Overview

This guide explains how to create, deploy, and manage workflows in the Clinical Synthesis Hub using BPMN 2.0 and Camunda Cloud integration.

## BPMN 2.0 Basics

### What is BPMN?

Business Process Model and Notation (BPMN) 2.0 is a standard for modeling business processes. It provides a graphical notation that is easy to understand by business users while being precise enough for technical implementation.

### Key BPMN Elements

#### Start Events
- **Start Event**: Triggers the beginning of a workflow
- **Message Start Event**: Triggered by receiving a message
- **Timer Start Event**: Triggered at a specific time or interval

```xml
<bpmn:startEvent id="start" name="Patient Admission Started">
  <bpmn:outgoing>flow1</bpmn:outgoing>
</bpmn:startEvent>
```

#### Tasks
- **User Task**: Requires human interaction
- **Service Task**: Automated task performed by the system
- **Script Task**: Executes a script

```xml
<bpmn:userTask id="review-patient-data" name="Review Patient Data">
  <bpmn:incoming>flow1</bpmn:incoming>
  <bpmn:outgoing>flow2</bpmn:outgoing>
  <bpmn:extensionElements>
    <zeebe:assignmentDefinition assignee="doctor" />
    <zeebe:formDefinition formKey="patient-review-form" />
  </bpmn:extensionElements>
</bpmn:userTask>
```

#### Gateways
- **Exclusive Gateway**: Only one path is taken
- **Parallel Gateway**: Multiple paths are executed simultaneously
- **Inclusive Gateway**: One or more paths are taken based on conditions

```xml
<bpmn:exclusiveGateway id="decision-gateway" name="Admission Decision">
  <bpmn:incoming>flow2</bpmn:incoming>
  <bpmn:outgoing>approve-flow</bpmn:outgoing>
  <bpmn:outgoing>reject-flow</bpmn:outgoing>
</bpmn:exclusiveGateway>
```

#### End Events
- **End Event**: Normal completion of the workflow
- **Error End Event**: Workflow ends with an error
- **Message End Event**: Sends a message when ending

```xml
<bpmn:endEvent id="end" name="Patient Admitted">
  <bpmn:incoming>flow3</bpmn:incoming>
</bpmn:endEvent>
```

## Workflow Design Patterns

### 1. Sequential Workflow
Tasks are executed one after another.

```xml
<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
                  xmlns:zeebe="http://camunda.org/schema/zeebe/1.0">
  <bpmn:process id="sequential-workflow" isExecutable="true">
    <bpmn:startEvent id="start"/>
    <bpmn:userTask id="task1" name="Step 1"/>
    <bpmn:userTask id="task2" name="Step 2"/>
    <bpmn:userTask id="task3" name="Step 3"/>
    <bpmn:endEvent id="end"/>
    
    <bpmn:sequenceFlow sourceRef="start" targetRef="task1"/>
    <bpmn:sequenceFlow sourceRef="task1" targetRef="task2"/>
    <bpmn:sequenceFlow sourceRef="task2" targetRef="task3"/>
    <bpmn:sequenceFlow sourceRef="task3" targetRef="end"/>
  </bpmn:process>
</bpmn:definitions>
```

### 2. Parallel Workflow
Multiple tasks are executed simultaneously.

```xml
<bpmn:process id="parallel-workflow" isExecutable="true">
  <bpmn:startEvent id="start"/>
  <bpmn:parallelGateway id="fork"/>
  <bpmn:userTask id="task1" name="Lab Tests"/>
  <bpmn:userTask id="task2" name="Imaging"/>
  <bpmn:parallelGateway id="join"/>
  <bpmn:endEvent id="end"/>
  
  <bpmn:sequenceFlow sourceRef="start" targetRef="fork"/>
  <bpmn:sequenceFlow sourceRef="fork" targetRef="task1"/>
  <bpmn:sequenceFlow sourceRef="fork" targetRef="task2"/>
  <bpmn:sequenceFlow sourceRef="task1" targetRef="join"/>
  <bpmn:sequenceFlow sourceRef="task2" targetRef="join"/>
  <bpmn:sequenceFlow sourceRef="join" targetRef="end"/>
</bpmn:process>
```

### 3. Conditional Workflow
Different paths based on conditions.

```xml
<bpmn:process id="conditional-workflow" isExecutable="true">
  <bpmn:startEvent id="start"/>
  <bpmn:userTask id="assessment" name="Patient Assessment"/>
  <bpmn:exclusiveGateway id="decision"/>
  <bpmn:userTask id="emergency-care" name="Emergency Care"/>
  <bpmn:userTask id="routine-care" name="Routine Care"/>
  <bpmn:endEvent id="end"/>
  
  <bpmn:sequenceFlow sourceRef="start" targetRef="assessment"/>
  <bpmn:sequenceFlow sourceRef="assessment" targetRef="decision"/>
  <bpmn:sequenceFlow sourceRef="decision" targetRef="emergency-care">
    <bpmn:conditionExpression>priority == "high"</bpmn:conditionExpression>
  </bpmn:sequenceFlow>
  <bpmn:sequenceFlow sourceRef="decision" targetRef="routine-care">
    <bpmn:conditionExpression>priority == "low"</bpmn:conditionExpression>
  </bpmn:sequenceFlow>
  <bpmn:sequenceFlow sourceRef="emergency-care" targetRef="end"/>
  <bpmn:sequenceFlow sourceRef="routine-care" targetRef="end"/>
</bpmn:process>
```

## Clinical Workflow Examples

### Patient Admission Workflow

```xml
<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
                  xmlns:zeebe="http://camunda.org/schema/zeebe/1.0">
  <bpmn:process id="patient-admission" isExecutable="true">
    
    <!-- Start Event -->
    <bpmn:startEvent id="admission-start" name="Patient Admission Request">
      <bpmn:outgoing>to-triage</bpmn:outgoing>
    </bpmn:startEvent>
    
    <!-- Triage Assessment -->
    <bpmn:userTask id="triage-assessment" name="Triage Assessment">
      <bpmn:incoming>to-triage</bpmn:incoming>
      <bpmn:outgoing>to-priority-decision</bpmn:outgoing>
      <bpmn:extensionElements>
        <zeebe:assignmentDefinition assignee="nurse" />
        <zeebe:formDefinition formKey="triage-form" />
      </bpmn:extensionElements>
    </bpmn:userTask>
    
    <!-- Priority Decision Gateway -->
    <bpmn:exclusiveGateway id="priority-decision" name="Priority Level">
      <bpmn:incoming>to-priority-decision</bpmn:incoming>
      <bpmn:outgoing>to-emergency</bpmn:outgoing>
      <bpmn:outgoing>to-routine</bpmn:outgoing>
    </bpmn:exclusiveGateway>
    
    <!-- Emergency Path -->
    <bpmn:userTask id="emergency-assessment" name="Emergency Assessment">
      <bpmn:incoming>to-emergency</bpmn:incoming>
      <bpmn:outgoing>to-room-assignment</bpmn:outgoing>
      <bpmn:extensionElements>
        <zeebe:assignmentDefinition assignee="doctor" />
      </bpmn:extensionElements>
    </bpmn:userTask>
    
    <!-- Routine Path -->
    <bpmn:userTask id="routine-assessment" name="Routine Assessment">
      <bpmn:incoming>to-routine</bpmn:incoming>
      <bpmn:outgoing>to-room-assignment</bpmn:outgoing>
      <bpmn:extensionElements>
        <zeebe:assignmentDefinition assignee="doctor" />
      </bpmn:extensionElements>
    </bpmn:userTask>
    
    <!-- Room Assignment -->
    <bpmn:userTask id="room-assignment" name="Assign Room">
      <bpmn:incoming>to-room-assignment</bpmn:incoming>
      <bpmn:outgoing>to-end</bpmn:outgoing>
      <bpmn:extensionElements>
        <zeebe:assignmentDefinition assignee="admin" />
      </bpmn:extensionElements>
    </bpmn:userTask>
    
    <!-- End Event -->
    <bpmn:endEvent id="admission-complete" name="Patient Admitted">
      <bpmn:incoming>to-end</bpmn:incoming>
    </bpmn:endEvent>
    
    <!-- Sequence Flows -->
    <bpmn:sequenceFlow id="to-triage" sourceRef="admission-start" targetRef="triage-assessment"/>
    <bpmn:sequenceFlow id="to-priority-decision" sourceRef="triage-assessment" targetRef="priority-decision"/>
    <bpmn:sequenceFlow id="to-emergency" sourceRef="priority-decision" targetRef="emergency-assessment">
      <bpmn:conditionExpression>priority == "high"</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="to-routine" sourceRef="priority-decision" targetRef="routine-assessment">
      <bpmn:conditionExpression>priority == "low"</bpmn:conditionExpression>
    </bpmn:sequenceFlow>
    <bpmn:sequenceFlow id="to-room-assignment" sourceRef="emergency-assessment" targetRef="room-assignment"/>
    <bpmn:sequenceFlow id="to-room-assignment-2" sourceRef="routine-assessment" targetRef="room-assignment"/>
    <bpmn:sequenceFlow id="to-end" sourceRef="room-assignment" targetRef="admission-complete"/>
    
  </bpmn:process>
</bpmn:definitions>
```

## Workflow Variables

### Variable Types
- **String**: Text values
- **Number**: Numeric values
- **Boolean**: True/false values
- **Object**: Complex JSON objects
- **Array**: Lists of values

### Variable Scope
- **Global Variables**: Available throughout the workflow
- **Local Variables**: Available only in specific tasks
- **Output Variables**: Results from completed tasks

### Example Variable Usage

```xml
<bpmn:userTask id="patient-review" name="Review Patient">
  <bpmn:extensionElements>
    <zeebe:ioMapping>
      <zeebe:input source="patientId" target="reviewPatientId"/>
      <zeebe:input source="patientName" target="reviewPatientName"/>
      <zeebe:output source="reviewResult" target="patientReviewResult"/>
      <zeebe:output source="notes" target="reviewNotes"/>
    </zeebe:ioMapping>
  </bpmn:extensionElements>
</bpmn:userTask>
```

## Task Assignment

### Assignment Types

#### 1. Role-Based Assignment
```xml
<zeebe:assignmentDefinition assignee="doctor" />
```

#### 2. User-Based Assignment
```xml
<zeebe:assignmentDefinition assignee="user-123" />
```

#### 3. Expression-Based Assignment
```xml
<zeebe:assignmentDefinition assignee="=if(priority == 'high', 'senior-doctor', 'junior-doctor')" />
```

## Forms and User Interfaces

### Form Definition
```xml
<zeebe:formDefinition formKey="patient-review-form" />
```

### Form Schema Example
```json
{
  "formKey": "patient-review-form",
  "title": "Patient Review",
  "fields": [
    {
      "key": "patientName",
      "label": "Patient Name",
      "type": "text",
      "readonly": true
    },
    {
      "key": "reviewResult",
      "label": "Review Result",
      "type": "select",
      "options": ["approved", "rejected", "needs-more-info"]
    },
    {
      "key": "notes",
      "label": "Notes",
      "type": "textarea"
    }
  ]
}
```

## Timers and Deadlines

### Timer Events
```xml
<bpmn:intermediateCatchEvent id="wait-timer" name="Wait 24 Hours">
  <bpmn:timerEventDefinition>
    <bpmn:timeDuration>PT24H</bpmn:timeDuration>
  </bpmn:timerEventDefinition>
</bpmn:intermediateCatchEvent>
```

### Task Deadlines
```xml
<bpmn:userTask id="urgent-review" name="Urgent Review">
  <bpmn:extensionElements>
    <zeebe:assignmentDefinition assignee="doctor" />
    <zeebe:taskDefinition type="user-task" />
    <zeebe:ioMapping>
      <zeebe:input source="=now() + duration('PT2H')" target="dueDate"/>
    </zeebe:ioMapping>
  </bpmn:extensionElements>
</bpmn:userTask>
```

## Error Handling

### Error Events
```xml
<bpmn:boundaryEvent id="timeout-error" attachedToRef="patient-review">
  <bpmn:outgoing>to-escalation</bpmn:outgoing>
  <bpmn:timerEventDefinition>
    <bpmn:timeDuration>PT4H</bpmn:timeDuration>
  </bpmn:timerEventDefinition>
</bpmn:boundaryEvent>
```

### Error End Events
```xml
<bpmn:endEvent id="error-end" name="Process Failed">
  <bpmn:errorEventDefinition errorRef="process-error"/>
</bpmn:endEvent>
```

## Best Practices

### 1. Naming Conventions
- Use descriptive names for all elements
- Follow consistent naming patterns
- Use verb-noun format for tasks (e.g., "Review Patient Data")

### 2. Process Design
- Keep processes simple and focused
- Use subprocesses for complex logic
- Minimize the number of decision points

### 3. Variable Management
- Use meaningful variable names
- Document variable purposes
- Validate input variables

### 4. Error Handling
- Always include error handling paths
- Use boundary events for timeouts
- Provide meaningful error messages

### 5. Testing
- Test all workflow paths
- Validate with different data sets
- Test error scenarios

## Deployment Process

### 1. Create Workflow Definition
```graphql
mutation CreateWorkflowDefinition {
  createWorkflowDefinition(input: {
    name: "Patient Admission Workflow"
    description: "Complete patient admission process"
    bpmnXml: "<?xml version='1.0'...>"
    category: "admission"
  }) {
    id
    version
  }
}
```

### 2. Deploy to Camunda
The workflow is automatically deployed to Camunda Cloud when created.

### 3. Activate Workflow
```graphql
mutation ActivateWorkflow($id: ID!) {
  activateWorkflowDefinition(id: $id) {
    id
    isActive
  }
}
```

### 4. Start Workflow Instance
```graphql
mutation StartWorkflow($definitionId: ID!, $patientId: ID!) {
  startWorkflow(definitionId: $definitionId, patientId: $patientId) {
    id
    status
  }
}
```

## Monitoring and Analytics

### Workflow Metrics
- Instance completion rates
- Average completion time
- Task assignment patterns
- Error frequencies

### Performance Optimization
- Monitor bottlenecks
- Optimize task assignments
- Review timeout settings
- Analyze user patterns

## Integration Points

### FHIR Resources
- Workflows create PlanDefinition resources
- Tasks create Task resources
- Workflow outputs update relevant FHIR resources

### Other Services
- Patient Service: Patient data retrieval
- Observation Service: Lab results
- Medication Service: Prescription management
- Encounter Service: Visit management

## Troubleshooting

### Common Issues
1. **Workflow not starting**: Check workflow definition syntax
2. **Tasks not assigned**: Verify user roles and permissions
3. **Variables not passed**: Check variable mapping
4. **Timeouts**: Review timer configurations

### Debug Tools
- Camunda Operate: Visual workflow monitoring
- Service logs: Detailed execution logs
- GraphQL playground: API testing
- Database queries: Data verification
