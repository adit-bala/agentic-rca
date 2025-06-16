'use client';

import { useEffect, useRef } from 'react';
import Neovis, { NeovisConfig } from 'neovis.js';

const ACTIVE_COLOR = '#FF6B00';
const NORMAL_COLOR = '#B6E900';
const NODE_LABEL_FONT = {
  color: '#222',
  size: 22,
  face: 'Inter, Arial, sans-serif',
};

const Neo4jVisualization = () => {
  const visRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!visRef.current) return;

    const config: NeovisConfig = {
      containerId: visRef.current.id,
      neo4j: {
        serverUrl: 'bolt://localhost:7687',
        serverUser: 'neo4j',
        serverPassword: 'password',
      },
      labels: {
        Service: {
          label: 'name',
          value: 'pagerank',
          group: 'community',
          title: 'name',
          // Node styling by group (simulate double ring with borderWidth and shadow)
          // You can set group in Neo4j or override in visConfig below
        },
      },
      relationships: {
        CALLS: {
          value: 'weight',
          arrows: 'to',
        },
      },
      initialCypher: 'MATCH (n:Service)-[r:CALLS]->(m:Service) RETURN n,r,m',
      visConfig: {
        nodes: {
          borderWidth: 6,
          borderWidthSelected: 8,
          color: {
            border: NORMAL_COLOR,
            background: '#fff',
            highlight: {
              border: ACTIVE_COLOR,
              background: '#fff',
            },
            hover: {
              border: ACTIVE_COLOR,
              background: '#fff',
            },
          },
          shadow: {
            enabled: true,
            color: '#eee',
            size: 20,
            x: 0,
            y: 0,
          },
          font: NODE_LABEL_FONT,
          shape: 'circle',
        },
        edges: {
          color: {
            color: NORMAL_COLOR,
            highlight: ACTIVE_COLOR,
            hover: ACTIVE_COLOR,
            inherit: false,
          },
          width: 3,
          arrows: {
            to: { enabled: true, scaleFactor: 1.2, type: 'arrow' },
          },
          font: {
            color: '#222',
            size: 16,
            face: 'Inter, Arial, sans-serif',
            strokeWidth: 0,
            align: 'middle',
          },
          smooth: {
            enabled: true,
            type: 'cubicBezier',
            roundness: 0.3,
          },
        },
        layout: {
          hierarchical: false,
        },
        physics: {
          enabled: true,
          barnesHut: {
            gravitationalConstant: -20000,
            springLength: 200,
            springConstant: 0.04,
            damping: 0.09,
            avoidOverlap: 1,
          },
        },
        interaction: {
          hover: true,
          tooltipDelay: 100,
        },
      },
    };

    const vis = new Neovis(config);
    vis.render();

    return () => {
      vis.clearNetwork();
    };
  }, []);

  return (
    <div className="w-full h-full bg-white rounded-lg border border-gray-200 shadow p-4">
      <h3 className="text-xl font-bold mb-2 text-black">Service Dependency Graph</h3>
      <div id="viz" ref={visRef} className="w-full h-[800px]" />
    </div>
  );
};

export default Neo4jVisualization; 