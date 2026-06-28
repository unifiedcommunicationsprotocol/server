import { SectionCard } from '../primitives/SectionCard';
import { StatusPill } from '../primitives/StatusPill';

const MOCK_ACCOUNTS = [
  {
    email: 'bridge@example.com',
    host: 'imap.example.com:993',
    synced: '2 min ago',
  },
  {
    email: 'legacy@corp.com',
    host: 'mail.corp.com:993',
    synced: '5 min ago',
  },
];

const MOCK_THREADING = [
  {
    smtpId: '<abc123@mail.example.com>',
    ucpId: '01JDTQ8MXYZ…',
    gap: 'No',
    gapColor: '#22C55E',
  },
  {
    smtpId: '<def456@corp.com>',
    ucpId: '01JDTQ9NABC…',
    gap: 'Yes',
    gapColor: '#D97706',
  },
  {
    smtpId: '<ghi789@news.co>',
    ucpId: '01JDTQADEF…',
    gap: 'No',
    gapColor: '#22C55E',
  },
];

export const Bridge = () => {
  return (
    <div className="fade-in space-y-3">
      <div className="grid grid-cols-2 gap-3 mb-3">
        {/* IMAP Accounts */}
        <div className="bg-[#111113] border border-[#1E1E22] rounded-lg p-[18px]">
          <div className="flex justify-between items-center mb-3.5">
            <h3 className="text-[12px] font-semibold text-[#FAFAFA]">IMAP Accounts</h3>
            <StatusPill
              label="2 connected"
              color="#22C55E"
              bgColor="rgba(34, 197, 94, 0.1)"
            />
          </div>

          <div className="flex flex-col gap-2">
            {MOCK_ACCOUNTS.map((account) => (
              <div
                key={account.email}
                className="bg-[#09090B] border border-[#1E1E22] rounded-md p-[11px]"
              >
                <div className="flex justify-between items-center mb-0.5">
                  <div className="text-[11px] text-[#FAFAFA] font-mono">{account.email}</div>
                  <div
                    className="w-[6px] h-[6px] rounded-full bg-[#22C55E]"
                  />
                </div>
                <div className="text-[10px] text-[#52525B]">
                  {account.host} · synced {account.synced}
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Bridge Attestation */}
        <div className="bg-[#111113] border border-[#1E1E22] rounded-lg p-[18px]">
          <h3 className="text-[12px] font-semibold text-[#FAFAFA] mb-1">Bridge Attestation</h3>
          <div className="text-[11px] text-[#52525B] mb-[14px]">
            Server-signed proof for SMTP-bridged messages
          </div>

          <div className="flex flex-col gap-[7px]">
            {[
              { label: 'source field', value: 'smtp', fontSize: '11px' },
              {
                label: 'signature covers',
                value: 'source + smtp_from + smtp_message_id + received_at',
                fontSize: '10px',
              },
              { label: 'DKIM', value: 'pass', fontSize: '10px' },
            ].map((item) => (
              <div
                key={item.label}
                className="bg-[#09090B] border border-[#1E1E22] rounded-md p-2.5"
              >
                <div className="text-[9px] text-[#52525B] uppercase tracking-[0.06em] mb-0.5">
                  {item.label}
                </div>
                <div
                  className="font-mono text-[#A1A1AA]"
                  style={{ fontSize: item.fontSize }}
                >
                  {item.value}
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Threading Map */}
      <div className="bg-[#111113] border border-[#1E1E22] rounded-lg overflow-hidden">
        <div className="px-[18px] py-3 border-b border-[#1E1E22]">
          <h3 className="text-[12px] font-semibold text-[#FAFAFA]">Threading Map</h3>
          <div className="text-[10px] text-[#52525B] mt-0.5">
            SMTP Message-ID ↔ UCP Thread ULID
          </div>
        </div>

        <table className="w-full border-collapse">
          <thead>
            <tr className="bg-[#18181B]">
              {['SMTP Message-ID', 'UCP Thread ULID', 'Threading Gap'].map((col) => (
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
            {MOCK_THREADING.map((row) => (
              <tr key={row.smtpId} className="border-t border-[#1E1E22]">
                <td className="px-4 py-2.5 font-mono text-[10px] text-[#A1A1AA]">{row.smtpId}</td>
                <td className="px-4 py-2.5 font-mono text-[10px] text-[#6366F1]">{row.ucpId}</td>
                <td className="px-4 py-2.5 text-[11px]" style={{ color: row.gapColor }}>
                  {row.gap}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};
