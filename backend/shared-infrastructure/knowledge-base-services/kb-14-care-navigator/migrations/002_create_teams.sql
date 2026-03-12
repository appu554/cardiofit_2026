-- KB-14 Care Navigator: Teams Table Migration
-- Migration: 002_create_teams
-- Description: Creates teams and team_members tables for care team management

-- Create teams table
CREATE TABLE IF NOT EXISTS teams (
    -- Primary key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    team_id VARCHAR(50) NOT NULL UNIQUE,

    -- Team details
    name VARCHAR(100) NOT NULL,
    type VARCHAR(50) NOT NULL, -- clinical, care_coordination, outreach, administrative
    manager_id UUID,

    -- Panel attribution - PCPs whose patients this team manages
    panel_pcps JSONB DEFAULT '[]'::jsonb,

    -- Settings
    max_tasks_per_member INTEGER DEFAULT 20,
    auto_assign BOOLEAN DEFAULT true,

    -- Status
    active BOOLEAN DEFAULT true,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create team_members table
CREATE TABLE IF NOT EXISTS team_members (
    -- Primary key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id VARCHAR(50) NOT NULL UNIQUE,

    -- User reference
    user_id VARCHAR(50) NOT NULL,
    team_id UUID NOT NULL REFERENCES teams(id) ON DELETE CASCADE,

    -- Member details
    name VARCHAR(100) NOT NULL,
    role VARCHAR(50) NOT NULL, -- Physician, Nurse, Care Coordinator, Pharmacist, etc.
    email VARCHAR(100),
    phone VARCHAR(20),

    -- Workload management
    max_tasks INTEGER DEFAULT 20,
    current_tasks INTEGER DEFAULT 0,
    available_from TIMESTAMPTZ,
    available_to TIMESTAMPTZ,

    -- Skills & capabilities (JSONB arrays)
    skills JSONB DEFAULT '[]'::jsonb,
    languages JSONB DEFAULT '[]'::jsonb,

    -- Supervisor for escalation chain
    supervisor_id UUID,

    -- Status
    active BOOLEAN DEFAULT true,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for teams
CREATE INDEX idx_teams_team_id ON teams(team_id);
CREATE INDEX idx_teams_type ON teams(type);
CREATE INDEX idx_teams_active ON teams(active);
CREATE INDEX idx_teams_manager_id ON teams(manager_id);

-- Create indexes for team_members
CREATE INDEX idx_team_members_member_id ON team_members(member_id);
CREATE INDEX idx_team_members_user_id ON team_members(user_id);
CREATE INDEX idx_team_members_team_id ON team_members(team_id);
CREATE INDEX idx_team_members_role ON team_members(role);
CREATE INDEX idx_team_members_active ON team_members(active);
CREATE INDEX idx_team_members_supervisor_id ON team_members(supervisor_id);

-- Composite index for available member queries
CREATE INDEX idx_team_members_available ON team_members(team_id, role, active, current_tasks)
    WHERE active = true;

-- GIN indexes for JSONB fields
CREATE INDEX idx_teams_panel_pcps ON teams USING GIN (panel_pcps);
CREATE INDEX idx_team_members_skills ON team_members USING GIN (skills);
CREATE INDEX idx_team_members_languages ON team_members USING GIN (languages);

-- Add check constraints
ALTER TABLE team_members ADD CONSTRAINT chk_team_members_tasks
    CHECK (current_tasks >= 0 AND current_tasks <= max_tasks);

-- Add foreign key for supervisor (self-referencing)
ALTER TABLE team_members ADD CONSTRAINT fk_team_members_supervisor
    FOREIGN KEY (supervisor_id) REFERENCES team_members(id) ON DELETE SET NULL;

-- Add foreign key from tasks to teams and team_members
ALTER TABLE tasks ADD CONSTRAINT fk_tasks_team
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE SET NULL;

-- Create triggers for updated_at
CREATE TRIGGER update_teams_updated_at
    BEFORE UPDATE ON teams
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_team_members_updated_at
    BEFORE UPDATE ON team_members
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments
COMMENT ON TABLE teams IS 'Care teams for task assignment and workload management';
COMMENT ON COLUMN teams.type IS 'Team type: clinical, care_coordination, outreach, administrative';
COMMENT ON COLUMN teams.panel_pcps IS 'JSONB array of PCP identifiers whose patients this team manages';
COMMENT ON COLUMN teams.auto_assign IS 'Enable automatic task assignment based on workload balancing';

COMMENT ON TABLE team_members IS 'Members of care teams with roles, skills, and workload tracking';
COMMENT ON COLUMN team_members.role IS 'Role: Physician, Nurse, Care Coordinator, Pharmacist, Scheduler, etc.';
COMMENT ON COLUMN team_members.skills IS 'JSONB array of skill keywords for matching';
COMMENT ON COLUMN team_members.languages IS 'JSONB array of spoken languages for patient matching';
COMMENT ON COLUMN team_members.current_tasks IS 'Current active task count for workload balancing';

-- Insert default teams for common care patterns
INSERT INTO teams (team_id, name, type, max_tasks_per_member, auto_assign) VALUES
    ('TEAM-CLINICAL-001', 'Primary Care Clinical Team', 'clinical', 15, true),
    ('TEAM-COORD-001', 'Care Coordination Team', 'care_coordination', 25, true),
    ('TEAM-OUTREACH-001', 'Patient Outreach Team', 'outreach', 30, true),
    ('TEAM-ADMIN-001', 'Administrative Team', 'administrative', 20, true)
ON CONFLICT (team_id) DO NOTHING;
