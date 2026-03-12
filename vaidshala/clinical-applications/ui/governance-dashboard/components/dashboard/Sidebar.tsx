'use client';

import { useState } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import {
  LayoutDashboard,
  ClipboardList,
  FileCheck,
  AlertTriangle,
  History,
  Settings,
  Activity,
  Shield,
  LogOut,
  FileText,
  Columns,
  ChevronsLeft,
  ChevronsRight,
  FlaskConical,
} from 'lucide-react';
import { cn } from '@/lib/utils';
import { useAuth } from '@/hooks/useAuth';
import { getUserInitials, getRoleDisplayName } from '@/lib/auth';

const navigation = [
  { name: 'Dashboard', href: '/', icon: LayoutDashboard },
  { name: 'Review Queue', href: '/queue', icon: ClipboardList },
  { name: 'Active Facts', href: '/facts', icon: FileCheck },
  { name: 'Conflicts', href: '/conflicts', icon: AlertTriangle },
  { name: 'Audit History', href: '/audit', icon: History },
  { name: 'Executor', href: '/executor', icon: Activity },
];

const extractionNav = [
  { name: 'SPL Review', href: '/spl-review', icon: FlaskConical },
  { name: 'Span Review', href: '/pipeline1', icon: FileText },
  { name: 'Compare View', href: '/pipeline1/compare', icon: Columns },
];

const secondaryNav = [
  { name: 'Settings', href: '/settings', icon: Settings },
];

export function Sidebar() {
  const pathname = usePathname();
  const { user, isLoading: authLoading } = useAuth();
  const [collapsed, setCollapsed] = useState(false);

  return (
    <div
      className={cn(
        'flex flex-col bg-white border-r border-gray-200 transition-[width] duration-200 shrink-0',
        collapsed ? 'w-16' : 'w-64'
      )}
    >
      {/* Logo + Collapse Toggle */}
      <div className="flex items-center h-16 px-3 border-b border-gray-200">
        {collapsed ? (
          <button
            onClick={() => setCollapsed(false)}
            className="mx-auto p-1.5 rounded-lg hover:bg-gray-100 transition-colors"
            title="Expand sidebar"
          >
            <Shield className="h-7 w-7 text-blue-600" />
          </button>
        ) : (
          <>
            <Shield className="h-8 w-8 text-blue-600 shrink-0 ml-3" />
            <div className="ml-3 min-w-0">
              <h1 className="text-lg font-bold text-gray-900">KB-0</h1>
              <p className="text-xs text-gray-500">Governance Platform</p>
            </div>
            <button
              onClick={() => setCollapsed(true)}
              className="ml-auto p-1 rounded text-gray-400 hover:text-gray-600 hover:bg-gray-100 transition-colors"
              title="Collapse sidebar"
            >
              <ChevronsLeft className="h-4 w-4" />
            </button>
          </>
        )}
      </div>

      {/* Primary Navigation */}
      <nav className="flex-1 px-2 py-4 space-y-1 overflow-y-auto overflow-x-hidden">
        {!collapsed && (
          <p className="px-3 text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">
            Governance
          </p>
        )}
        {navigation.map((item) => {
          const isActive = pathname === item.href;
          return (
            <Link
              key={item.name}
              href={item.href}
              title={collapsed ? item.name : undefined}
              className={cn(
                'flex items-center text-sm font-medium rounded-lg transition-colors',
                collapsed ? 'justify-center px-2 py-2.5' : 'px-3 py-2',
                isActive
                  ? 'bg-blue-50 text-blue-700'
                  : 'text-gray-700 hover:bg-gray-100'
              )}
            >
              <item.icon
                className={cn(
                  'h-5 w-5 shrink-0',
                  !collapsed && 'mr-3',
                  isActive ? 'text-blue-600' : 'text-gray-400'
                )}
              />
              {!collapsed && item.name}
            </Link>
          );
        })}

        {/* Extraction Pipeline */}
        <div className={collapsed ? 'pt-4' : 'pt-6'}>
          {!collapsed && (
            <p className="px-3 text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">
              Extraction
            </p>
          )}
          {collapsed && (
            <div className="mx-auto w-6 border-t border-gray-200 mb-3" />
          )}
          {extractionNav.map((item) => {
            const basePath = item.href.split('?')[0];
            const isActive = pathname === basePath || pathname?.startsWith(basePath + '/');
            return (
              <Link
                key={item.name}
                href={item.href}
                title={collapsed ? item.name : undefined}
                className={cn(
                  'flex items-center text-sm font-medium rounded-lg transition-colors',
                  collapsed ? 'justify-center px-2 py-2.5' : 'px-3 py-2',
                  isActive
                    ? 'bg-blue-50 text-blue-700'
                    : 'text-gray-700 hover:bg-gray-100'
                )}
              >
                <item.icon
                  className={cn(
                    'h-5 w-5 shrink-0',
                    !collapsed && 'mr-3',
                    isActive ? 'text-blue-600' : 'text-gray-400'
                  )}
                />
                {!collapsed && item.name}
              </Link>
            );
          })}
        </div>

        {/* Secondary Navigation */}
        <div className={collapsed ? 'pt-4' : 'pt-6'}>
          {!collapsed && (
            <p className="px-3 text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">
              System
            </p>
          )}
          {collapsed && (
            <div className="mx-auto w-6 border-t border-gray-200 mb-3" />
          )}
          {secondaryNav.map((item) => {
            const isActive = pathname === item.href;
            return (
              <Link
                key={item.name}
                href={item.href}
                title={collapsed ? item.name : undefined}
                className={cn(
                  'flex items-center text-sm font-medium rounded-lg transition-colors',
                  collapsed ? 'justify-center px-2 py-2.5' : 'px-3 py-2',
                  isActive
                    ? 'bg-blue-50 text-blue-700'
                    : 'text-gray-700 hover:bg-gray-100'
                )}
              >
                <item.icon
                  className={cn(
                    'h-5 w-5 shrink-0',
                    !collapsed && 'mr-3',
                    isActive ? 'text-blue-600' : 'text-gray-400'
                  )}
                />
                {!collapsed && item.name}
              </Link>
            );
          })}
        </div>
      </nav>

      {/* Expand button (collapsed mode) */}
      {collapsed && (
        <div className="px-2 pb-2">
          <button
            onClick={() => setCollapsed(false)}
            className="w-full flex items-center justify-center p-2 rounded-lg text-gray-400 hover:text-gray-600 hover:bg-gray-100 transition-colors"
            title="Expand sidebar"
          >
            <ChevronsRight className="h-4 w-4" />
          </button>
        </div>
      )}

      {/* User Info */}
      <div className="p-3 border-t border-gray-200">
        {authLoading ? (
          <div className="flex items-center justify-center">
            <div className="h-8 w-8 rounded-full bg-gray-200 animate-pulse" />
          </div>
        ) : collapsed ? (
          /* Collapsed: avatar only */
          <div className="flex flex-col items-center gap-2">
            {user?.picture ? (
              <img src={user.picture} alt={user.name} className="h-8 w-8 rounded-full" title={user.name} />
            ) : (
              <div className="h-8 w-8 rounded-full bg-blue-100 flex items-center justify-center" title={user?.name || 'User'}>
                <span className="text-sm font-medium text-blue-600">
                  {getUserInitials(user)}
                </span>
              </div>
            )}
            <a
              href="/api/auth/logout"
              className="p-1.5 rounded-lg text-gray-400 hover:text-gray-600 hover:bg-gray-100 transition-colors"
              title="Sign Out"
            >
              <LogOut className="h-4 w-4" />
            </a>
          </div>
        ) : (
          /* Expanded: full user info */
          <>
            <div className="flex items-center">
              {user?.picture ? (
                <img src={user.picture} alt={user.name} className="h-8 w-8 rounded-full" />
              ) : (
                <div className="h-8 w-8 rounded-full bg-blue-100 flex items-center justify-center">
                  <span className="text-sm font-medium text-blue-600">
                    {getUserInitials(user)}
                  </span>
                </div>
              )}
              <div className="ml-3 min-w-0">
                <p className="text-sm font-medium text-gray-900 truncate">{user?.name || 'User'}</p>
                <p className="text-xs text-gray-500 truncate">{getRoleDisplayName(user)}</p>
              </div>
            </div>
            <a
              href="/api/auth/logout"
              className="mt-3 w-full flex items-center justify-center px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
            >
              <LogOut className="h-4 w-4 mr-2" />
              Sign Out
            </a>
          </>
        )}
      </div>
    </div>
  );
}
