'use client';

import { useEffect, useState } from 'react';
import { getAdminFederationConnections, getAdminFederationQueue } from '../../../api/handlers';
import { StatusPill } from '../primitives/StatusPill';

interface FederationConnection {
  remote_domain: string;
  established_at: number;
  last_activity: number;
  retries: number;
}

interface QueueItem {
  recipient: string;
  envelope_id: string;
  attempts: number;
  last_attempt: number;
  status: string;
  next_retry: number;
}

interface QueueData {
  queue_depth: number;
  failed_count: number;
  items: QueueItem[];
}

export const Federation = () => {
  const [connections, setConnections] = useState<FederationConnection[]>([]);
  const [queueData, setQueueData] = useState<QueueData>({
    queue_depth: 0,
    failed_count: 0,
    items: [],
  });
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      setIsLoading(true);
      const [connsData, queueResult] = await Promise.all([
        getAdminFederationConnections(),
        getAdminFederationQueue(),
      ]);
      setConnections(connsData.connections || []);
      setQueueData(queueResult || { queue_depth: 0, failed_count: 0, items: [] });
      setIsLoading(false);
    };

    fetchData();
    // Refresh every 10 seconds
    const interval = setInterval(fetchData, 10000);
    return () => clearInterval(interval);
  }, []);
  const statCards = [
    { label: 'Connections', value: connections.length.toString() },
    { label: 'Queue Depth', value: queueData.queue_depth.toString() },
    { label: 'Failed', value: queueData.failed_count.toString(), color: '#22C55E' },
  ];

  return (
    <div className="fade-in space-y-3.5">
      {/* Stat cards */}
      <div className="grid grid-cols-3 gap-2.5 mb-3.5">
        {statCards.map((card) => (
          <div key={card.label} className="bg-[#111113] border border-[#1E1E22] rounded-lg p-4">
            <div className="text-[10px] text-[#52525B] uppercase tracking-[0.07em] mb-2">
              {card.label}
            </div>
            <div className="text-[26px] font-bold leading-none" style={{ color: card.color || '#FAFAFA' }}>
              {card.value}
            </div>
          </div>
        ))}
      </div>

      {/* Delivery Queue Table */}
      <div className="bg-[#111113] border border-[#1E1E22] rounded-lg overflow-hidden">
        <div className="px-[18px] py-3 border-b border-[#1E1E22]">
          <h3 className="text-[12px] font-semibold text-[#FAFAFA]">Delivery Queue</h3>
        </div>

        <table className="w-full border-collapse">
          <thead>
            <tr className="bg-[#18181B]">
              {['Recipient', 'Envelope ID', 'Attempts', 'Last Attempt', 'Status'].map((col) => (
                <th
                  key={col}
                  className="px-4 py-1.5 text-left text-[9px] uppercase tracking-[0.07em] text-[#52525B] font-medium"
                >
                  {col}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {isLoading ? (
              <tr>
                <td colSpan={5} className="px-4 py-4 text-center text-[12px] text-[#52525B]">
                  Loading...
                </td>
              </tr>
            ) : queueData.items.length === 0 ? (
              <tr>
                <td colSpan={5} className="px-4 py-4 text-center text-[12px] text-[#52525B]">
                  No items in queue
                </td>
              </tr>
            ) : (
              queueData.items.map((row) => (
                <tr key={row.envelope_id} className="border-t border-[#1E1E22]">
                  <td className="px-4 py-2.5 text-[12px] text-[#FAFAFA]">{row.recipient}</td>
                  <td className="px-4 py-2.5 font-mono text-[10px] text-[#A1A1AA]">{row.envelope_id}</td>
                  <td className="px-4 py-2.5 text-[12px] text-[#A1A1AA]">{row.attempts}</td>
                  <td className="px-4 py-2.5 text-[10px] text-[#52525B] font-mono">
                    {new Date(row.last_attempt * 1000).toLocaleString()}
                  </td>
                  <td className="px-4 py-2.5">
                    <StatusPill label={row.status} />
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
};
