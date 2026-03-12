# Supabase Auth Implementation with RBAC

This guide outlines how to implement Role-Based Access Control (RBAC) with Supabase authentication in the Clinical Synthesis Hub application.

## Database Schema

First, create the necessary tables in your Supabase database:

```sql
-- Roles table
CREATE TABLE roles (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  name TEXT NOT NULL UNIQUE,
  description TEXT,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Permissions table
CREATE TABLE permissions (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  name TEXT NOT NULL UNIQUE,
  description TEXT,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Role-Permission mapping table
CREATE TABLE role_permissions (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
  permission_id UUID REFERENCES permissions(id) ON DELETE CASCADE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  UNIQUE(role_id, permission_id)
);

-- User-Role mapping table
CREATE TABLE user_roles (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
  role_id UUID REFERENCES roles(id) ON DELETE CASCADE,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  UNIQUE(user_id, role_id)
);
```

## Row-Level Security (RLS) Policies

Set up RLS policies to secure your tables:

```sql
-- Enable RLS on all tables
ALTER TABLE roles ENABLE ROW LEVEL SECURITY;
ALTER TABLE permissions ENABLE ROW LEVEL SECURITY;
ALTER TABLE role_permissions ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_roles ENABLE ROW LEVEL SECURITY;

-- Policies for roles table
CREATE POLICY "Roles are viewable by authenticated users" 
ON roles FOR SELECT 
TO authenticated 
USING (true);

CREATE POLICY "Roles are editable by users with the admin role" 
ON roles FOR ALL 
TO authenticated 
USING (
  EXISTS (
    SELECT 1 FROM user_roles ur
    JOIN roles r ON ur.role_id = r.id
    WHERE ur.user_id = auth.uid() AND r.name = 'admin'
  )
);

-- Repeat similar policies for other tables
```

## Database Functions

Create a function to get user roles and permissions:

```sql
CREATE OR REPLACE FUNCTION get_user_roles_and_permissions(user_id UUID)
RETURNS TABLE (
  role_name TEXT,
  permissions TEXT[]
) AS $$
BEGIN
  RETURN QUERY
  SELECT 
    r.name AS role_name,
    ARRAY_AGG(p.name) AS permissions
  FROM
    user_roles ur
    JOIN roles r ON ur.role_id = r.id
    LEFT JOIN role_permissions rp ON r.id = rp.role_id
    LEFT JOIN permissions p ON rp.permission_id = p.id
  WHERE
    ur.user_id = get_user_roles_and_permissions.user_id
  GROUP BY
    r.name;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;
```

## JWT Claims in Supabase

Set up Supabase to include role information in JWT tokens:

1. Navigate to your Supabase project settings
2. Go to "API" → "JWT Settings"
3. Add a custom claim template to include roles and permissions:

```json
{
  "user_roles": {
    "roles": "((SELECT array_agg(r.name) FROM user_roles ur JOIN roles r ON ur.role_id = r.id WHERE ur.user_id = auth.uid()))",
    "permissions": "((SELECT array_agg(DISTINCT p.name) FROM user_roles ur JOIN role_permissions rp ON ur.role_id = rp.role_id JOIN permissions p ON rp.permission_id = p.id WHERE ur.user_id = auth.uid()))"
  }
}
```

## Implementation in Auth Service

Update your auth service to verify and extract roles and permissions from the JWT token:

1. In `security.py`, modify the `verify_supabase_token` function to handle the custom claims
2. Use the roles and permissions for authorization decisions

## Apply RBAC in API Gateway

Use the extracted roles and permissions to control access to API endpoints:

1. Update middleware to check roles and permissions
2. Use decorators to protect endpoints

## Example Usage in API Endpoints

```python
@router.get("/patients")
@require_permissions(["read:patients"])
async def get_patients(current_user = Depends(get_current_user)):
    # Only users with "read:patients" permission can access this endpoint
    return {"message": "List of patients"}

@router.post("/admin/settings")
@require_role(["admin"])
async def update_settings(current_user = Depends(get_current_user)):
    # Only users with "admin" role can access this endpoint
    return {"message": "Settings updated"}
```

## User Management

When creating new users or modifying existing ones:

1. Add appropriate entries to the `user_roles` table
2. The custom JWT claims will automatically include the roles and permissions
3. No code changes needed in your application to update the JWT claims