'use client';

import { useUser } from '@auth0/nextjs-auth0/client';
import { extractRoles } from '@/lib/auth';
import type { UserRole, GovernanceUser } from '@/lib/auth';

export interface UseAuthReturn {
  user: GovernanceUser | null;
  isLoading: boolean;
  error?: Error;
  hasRole: (role: UserRole) => boolean;
  canReviewFacts: boolean;
  isAdmin: boolean;
}

export function useAuth(): UseAuthReturn {
  const { user: auth0User, error, isLoading } = useUser();

  if (!auth0User || isLoading) {
    return {
      user: null,
      isLoading,
      error: error as Error | undefined,
      hasRole: () => false,
      canReviewFacts: false,
      isAdmin: false,
    };
  }

  const roles = extractRoles(auth0User as unknown as Record<string, unknown>);

  const user: GovernanceUser = {
    sub: auth0User.sub || '',
    email: auth0User.email || '',
    name: auth0User.name || '',
    nickname: auth0User.nickname || '',
    picture: auth0User.picture || '',
    roles,
  };

  const canReview = roles.includes('ADMIN') || roles.includes('PHARMACIST');
  const admin = roles.includes('ADMIN');

  return {
    user,
    isLoading: false,
    error: undefined,
    hasRole: (role: UserRole) => roles.includes(role),
    canReviewFacts: canReview,
    isAdmin: admin,
  };
}
