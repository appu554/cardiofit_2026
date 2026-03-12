import { NextRequest, NextResponse } from 'next/server';
import { auth0 } from '@/lib/auth0';

export async function middleware(req: NextRequest) {
  // Let the Auth0 SDK handle its own API auth routes
  const authRes = await auth0.middleware(req);

  // For API auth routes, return the SDK response directly
  if (req.nextUrl.pathname.startsWith('/api/auth')) {
    return authRes;
  }

  // Let KB-0 API proxy rewrites pass through without auth redirect
  if (req.nextUrl.pathname.startsWith('/api/v2/')) {
    return NextResponse.next();
  }

  // DEV ONLY: bypass auth for pipeline1 pages during local testing
  if (process.env.NODE_ENV === 'development' && req.nextUrl.pathname.startsWith('/pipeline1')) {
    return NextResponse.next();
  }

  // For all other routes, check if user is authenticated
  const session = await auth0.getSession(req);
  if (!session) {
    // Redirect unauthenticated users to Auth0 login
    return NextResponse.redirect(
      new URL('/api/auth/login?returnTo=' + encodeURIComponent(req.nextUrl.pathname), req.url)
    );
  }

  return authRes;
}

export const config = {
  matcher: [
    '/((?!_next/static|_next/image|favicon.ico).*)',
  ],
};
