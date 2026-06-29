import { useState } from 'react';
import { SectionCard } from '../primitives/SectionCard';
import { DataBlock } from '../primitives/DataBlock';
import { getServerKey, getIdentity } from '../../../api/handlers';

export const Identity = () => {
  const [lookupAddr, setLookupAddr] = useState('');
  const [lookupResult, setLookupResult] = useState('// Enter an address and click Lookup');

  const handleRefreshKey = async () => {
    const data = await getServerKey();
    if (data) {
      // Update display with real key
      // For now, just show it was fetched
      console.log('Server key fetched:', data);
    }
  };

  const handleLookup = async () => {
    if (!lookupAddr) return;
    try {
      const data = await getIdentity(lookupAddr);
      setLookupResult(JSON.stringify(data, null, 2));
    } catch (error) {
      setLookupResult(`// Error: ${String(error)}`);
    }
  };

  return (
    <div className="fade-in space-y-3">
      <div className="grid grid-cols-2 gap-3 mb-3">
        {/* Server Key */}
        <SectionCard title="Server Key" subtitle="Ed25519 public key for this server">
          <DataBlock
            label="domain"
            value="localhost:5150"
            valueColor="#6366F1"
            mono
          />
          <DataBlock
            label="key (Ed25519, base64)"
            value="MCowBQYDK2VdAyEA...\n(mock — connect server to see real key)"
            mono
          />
          <button
            onClick={handleRefreshKey}
            className="mt-3 px-3 py-1.5 bg-[#18181B] border border-[#1E1E22] rounded-md text-[#A1A1AA] text-[11px] hover:border-[#6366F1] transition-colors"
          >
            Refresh from server
          </button>
        </SectionCard>

        {/* Resolve Identity */}
        <SectionCard title="Resolve Identity" subtitle="Look up any UCP address">
          <div className="flex gap-[7px] mb-2.5">
            <input
              type="text"
              value={lookupAddr}
              onChange={(e) => setLookupAddr(e.target.value)}
              placeholder="alice@example.com"
              className="flex-1 px-2.5 py-1.5 bg-[#09090B] border border-[#1E1E22] rounded-md text-[#FAFAFA] font-mono text-[11px] outline-none focus:border-[#6366F1] transition-colors"
            />
            <button
              onClick={handleLookup}
              className="px-3.5 py-1.5 bg-[#6366F1] border-0 rounded-md text-white text-[12px] font-semibold hover:bg-[#5558E3] transition-colors"
            >
              Lookup
            </button>
          </div>
          <pre className="m-0 bg-[#09090B] border border-[#1E1E22] rounded-md px-2.5 py-2.5 font-mono text-[10px] text-[#A1A1AA] whitespace-pre-wrap min-h-[130px] max-h-[220px] overflow-y-auto leading-[1.6]">
            {lookupResult}
          </pre>
        </SectionCard>
      </div>

      {/* Key Infrastructure */}
      <SectionCard title="Key Infrastructure">
        <div className="grid grid-cols-3 gap-2.5">
          {[
            {
              label: 'Identity Key',
              title: 'Ed25519 · primary',
              body: 'DNS-anchored, long-lived. Signs all signing keys.',
            },
            {
              label: 'Signing Keys',
              title: 'Ed25519 · rotating',
              body: 'active / grace / expired lifecycle. Signs messages.',
            },
            {
              label: 'Revocation Key',
              title: 'Ed25519 · offline',
              body: 'Immediate revocation + offline recovery path.',
            },
          ].map((item, idx) => (
            <div
              key={idx}
              className="bg-[#09090B] border border-[#1E1E22] rounded-md p-3.5"
            >
              <div className="text-[9px] text-[#52525B] uppercase tracking-[0.07em] mb-[7px]">
                {item.label}
              </div>
              <div className="text-[12px] text-[#FAFAFA] font-medium mb-1.5">{item.title}</div>
              <div className="text-[11px] text-[#71717A] leading-[1.5]">{item.body}</div>
            </div>
          ))}
        </div>
      </SectionCard>
    </div>
  );
};
