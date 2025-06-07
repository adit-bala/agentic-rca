'use client';

import { useEffect, useRef } from 'react';
import Neovis, { NeovisConfig } from 'neovis.js';

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
        },
      },
      relationships: {
        CALLS: {
          value: 'weight',
          arrows: 'to',
        },
      },
      initialCypher: 'MATCH (n)-[r]->(m) RETURN n,r,m',
    };

    const vis = new Neovis(config);
    vis.render();

    return () => {
      vis.clearNetwork();
    };
  }, []);

  return (
    <div className="w-full h-full">
      <div id="viz" ref={visRef} className="w-full h-[800px]" />
    </div>
  );
};

export default Neo4jVisualization; 