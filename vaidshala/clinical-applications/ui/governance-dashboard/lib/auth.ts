// Auth0 utilities and type definitions for the Governance Dashboard

export type UserRole = 'ADMIN' | 'PHARMACIST' | 'VIEWER';

export interface GovernanceUser {
  sub: string;
  email: string;
  name: string;
  nickname: string;
  picture: string;
  roles: UserRole[];
}

const ROLES_NAMESPACE = 'https://cardiofit.com/roles';

/**
 * Extract roles from Auth0 user claims
 */
export function extractRoles(user: Record<string, unknown>): UserRole[] {
  return (user[ROLES_NAMESPACE] as UserRole[]) || [];
}

/**
 * Check if user has a specific role
 */
export function hasRole(user: GovernanceUser | null, role: UserRole): boolean {
  return user?.roles.includes(role) || false;
}

/**
 * Check if user can approve/reject facts (PHARMACIST or ADMIN)
 */
export function canReviewFacts(user: GovernanceUser | null): boolean {
  return hasRole(user, 'ADMIN') || hasRole(user, 'PHARMACIST');
}

/**
 * Check if user is admin
 */
export function isAdmin(user: GovernanceUser | null): boolean {
  return hasRole(user, 'ADMIN');
}

/**
 * Get user initials for avatar fallback
 */
export function getUserInitials(user: GovernanceUser | null): string {
  if (!user?.name) return 'U';
  const parts = user.name.split(' ');
  if (parts.length >= 2) return parts[0][0] + parts[1][0];
  return user.name[0].toUpperCase();
}

/**
 * Get display name for primary role
 */
export function getRoleDisplayName(user: GovernanceUser | null): string {
  if (!user || user.roles.length === 0) return 'User';
  const map: Record<UserRole, string> = {
    ADMIN: 'Administrator',
    PHARMACIST: 'Clinical Reviewer',
    VIEWER: 'Read-only',
  };
  return map[user.roles[0]] || 'User';
}
