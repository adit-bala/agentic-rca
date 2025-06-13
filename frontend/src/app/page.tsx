'use client';

import dynamic from 'next/dynamic';

const Neo4jVisualization = dynamic(() => import('@/components/Neo4jVisualization'), {
  ssr: false,
});

export default function Home() {
  return (
    <main className="p-8 min-h-screen" style={{ background: '#F3FF3D' }}>
      <div className="max-w-7xl mx-auto">
        <div className="bg-white rounded-lg shadow-lg p-6">
          <h2 className="text-2xl font-bold mb-6 text-black">Service Graph</h2>
          <Neo4jVisualization />
        </div>
      </div>
    </main>
  );
}
