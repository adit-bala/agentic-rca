'use client';

import { useState } from 'react';
import dynamic from 'next/dynamic';

const Neo4jVisualization = dynamic(() => import('@/components/Neo4jVisualization'), {
  ssr: false,
});

export default function Home() {
  const [activeTab, setActiveTab] = useState('graph');

  return (
    <main className="flex min-h-screen">
      {/* Sidebar */}
      <div className="w-64 bg-gray-800 text-white p-4">
        <h2 className="text-xl font-bold mb-4">Service Graph</h2>
        <nav>
          <button
            className={`w-full text-left p-2 rounded ${
              activeTab === 'graph' ? 'bg-gray-700' : 'hover:bg-gray-700'
            }`}
            onClick={() => setActiveTab('graph')}
          >
            Graph Visualization
          </button>
        </nav>
      </div>

      {/* Main content */}
      <div className="flex-1 p-8">
        {activeTab === 'graph' && <Neo4jVisualization />}
      </div>
    </main>
  );
}
