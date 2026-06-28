import { StatusPill } from '../primitives/StatusPill';

const MOCK_QUEUE = [
  {
    recipient: 'bob@relay.remote',
    envelopeId: '01JDTQ9X…',
    attempts: 1,
    lastAttempt: '2026-06-27 14:22',
    status: 'delivered',
    statusColor: '#22C55E',
    statusBg: 'rgba(34, 197, 94, 0.1)',
  },
  {
    recipient: 'carol@ucp.dev',
    envelopeId: '01JDTQ8M…',
    attempts: 2,
    lastAttempt: '2026-06-27 14:18',
    status: 'pending',
    statusColor: '#D97706',
    statusBg: 'rgba(217, 119, 6, 0.1)',
  },
];

export const Federation = () => {
  return (
    <div className="fade-in space-y-3.5">
      {/* Stat cards */}
      <div className="grid grid-cols-3 gap-2.5 mb-3.5">
        {[
          { label: 'Connections', value: '3' },
          { label: 'Queue Depth', value: '2' },
          { label: 'Failed', value: '0', color: '#22C55E' },
        ].map((card) => (
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
            {MOCK_QUEUE.map((row) => (
              <tr key={row.envelopeId} className="border-t border-[#1E1E22]">
                <td className="px-4 py-2.5 text-[12px] text-[#FAFAFA]">{row.recipient}</td>
                <td className="px-4 py-2.5 font-mono text-[10px] text-[#A1A1AA]">{row.envelopeId}</td>
                <td className="px-4 py-2.5 text-[12px] text-[#A1A1AA]">{row.attempts}</td>
                <td className="px-4 py-2.5 text-[10px] text-[#52525B] font-mono">{row.lastAttempt}</td>
                <td className="px-4 py-2.5">
                  <StatusPill label={row.status} color={row.statusColor} bgColor={row.statusBg} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};
