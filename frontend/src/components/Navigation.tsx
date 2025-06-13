'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';

export default function Navigation() {
  const pathname = usePathname();

  return (
    <nav className="fixed left-0 top-0 h-screen w-64 bg-white text-black shadow-lg border-r border-gray-200">
      <div className="flex flex-col h-full">
        <div className="p-6">
          <Link href="/" className="text-2xl font-bold text-black hover:text-[#B6E900] transition-colors">
            Agentic RCA
          </Link>
        </div>
        
        <div className="flex-1 px-4">
          <div className="space-y-2">
            <Link
              href="/"
              className={`flex items-center px-4 py-3 rounded-lg transition-colors font-semibold ${
                pathname === '/'
                  ? 'bg-[#B6E900] text-black'
                  : 'text-gray-700 hover:bg-[#F3FF3D] hover:text-black'
              }`}
            >
              <span className="text-sm font-medium">Dashboard</span>
            </Link>
            <Link
              href="/alerts"
              className={`flex items-center px-4 py-3 rounded-lg transition-colors font-semibold ${
                pathname === '/alerts'
                  ? 'bg-[#B6E900] text-black'
                  : 'text-gray-700 hover:bg-[#F3FF3D] hover:text-black'
              }`}
            >
              <span className="text-sm font-medium">Alerts</span>
            </Link>
          </div>
        </div>
      </div>
    </nav>
  );
} 