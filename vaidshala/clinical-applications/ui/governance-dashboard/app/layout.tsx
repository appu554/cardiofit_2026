import type { Metadata } from 'next';
import { Inter } from 'next/font/google';
import '@/styles/globals.css';
import { Auth0Provider } from '@auth0/nextjs-auth0/client';
import { Providers } from './providers';
import { Sidebar } from '@/components/dashboard/Sidebar';
import { Header } from '@/components/dashboard/Header';

const inter = Inter({ subsets: ['latin'] });

export const metadata: Metadata = {
  title: 'KB-0 Governance Dashboard',
  description: 'Canonical Fact Store Governance Platform for Clinical Knowledge Management',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body className={inter.className}>
        <Auth0Provider>
        <Providers>
          <div className="flex h-screen bg-gray-50">
            {/* Sidebar Navigation */}
            <Sidebar />

            {/* Main Content Area */}
            <div className="flex-1 flex flex-col overflow-hidden">
              {/* Header */}
              <Header />

              {/* Page Content */}
              <main className="flex-1 overflow-y-auto p-6">
                {children}
              </main>
            </div>
          </div>
        </Providers>
        </Auth0Provider>
      </body>
    </html>
  );
}
