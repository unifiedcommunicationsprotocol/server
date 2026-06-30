'use client';

import { useEffect, useState } from 'react';
import { getAdminSessions } from '../../../api/handlers';
import { SectionCard } from '../primitives/SectionCard';
import { StatusPill } from '../primitives/StatusPill';

interface SessionData {
  token: string;
  identity: string;
  issued_at: number;
  expires_at: number;
  status: string;
}

export const Sessions = () => {
  const [sessions, setSessions] = useState<SessionData[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const fetchSessions = async () => {
      setIsLoading(true);
      const data = await getAdminSessions();
      setSessions(data.sessions || []);
      setIsLoading(false);
    };

    fetchSessions();
    // Refresh every 10 seconds
    const interval = setInterval(fetchSessions, 10000);
    return () => clearInterval(interval);
  }, []);
  return (
    <div className="fade-in space-y-3.5">
      {/* Active Sessions Table */}
      <div className="bg-[#111113] border border-[#1E1E22] rounded-lg overflow-hidden">
        <div className="px-[18px] py-3 border-b border-[#1E1E22] flex justify-between items-center">
          <h3 className="text-[12px] font-semibold text-[#FAFAFA]">Active Sessions</h3>
          <div className="text-[10px] text-[#52525B]">24-hour bearer tokens · Ed25519 challenge-response</div>
        </div>

        <table className="w-full border-collapse">
          <thead>
            <tr className="bg-[#18181B]">
              {['Token', 'Identity', 'Issued', 'Expires', 'Status'].map((col) => (
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
                  Loading sessions...
                </td>
              </tr>
            ) : sessions.length === 0 ? (
              <tr>
                <td colSpan={5} className="px-4 py-4 text-center text-[12px] text-[#52525B]">
                  No active sessions
                </td>
              </tr>
            ) : (
              sessions.map((session) => (
                <tr key={session.token} className="border-t border-[#1E1E22]">
                  <td className="px-4 py-2.5 font-mono text-[10px] text-[#A1A1AA]">{session.token}</td>
                  <td className="px-4 py-2.5 text-[12px] text-[#FAFAFA]">{session.identity}</td>
                  <td className="px-4 py-2.5 text-[10px] text-[#52525B] font-mono">
                    {new Date(session.issued_at * 1000).toLocaleString()}
                  </td>
                  <td className="px-4 py-2.5 text-[10px] text-[#52525B] font-mono">
                    {new Date(session.expires_at * 1000).toLocaleString()}
                  </td>
                  <td className="px-4 py-2.5">
                    <StatusPill label={session.status} />
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Auth Flow */}
      <SectionCard title="Auth Flow">
        <div className="grid grid-cols-3 gap-2.5">
          {[
            {
              step: 1,
              name: 'Challenge',
              endpoint: 'POST /auth/challenge',
              code: '{"address":"alice@example.com"}',
              note: '→ 60-second random challenge',
            },
            {
              step: 2,
              name: 'Sign',
              endpoint: 'client-side only',
              code: 'ed25519.sign(challenge, identityKey)',
              note: '→ 64-byte Ed25519 signature',
            },
            {
              step: 3,
              name: 'Session',
              endpoint: 'POST /auth/session',
              code: '{"address":"...","challenge":"...","signature":"..."}',
              note: '→ 24-hr Bearer token',
            },
          ].map((item) => (
            <div
              key={item.step}
              className="bg-[#09090B] border border-[#1E1E22] rounded-md p-3.5"
            >
              <div className="flex items-center gap-2 mb-2.5">
                <div className="w-[18px] h-[18px] rounded-full bg-[#6366F1] flex items-center justify-center text-[9px] font-bold text-white">
                  {item.step}
                </div>
                <div className="text-[12px] font-semibold text-[#FAFAFA]">{item.name}</div>
              </div>

              <div className="text-[10px] font-mono text-[#52525B] mb-2">
                {item.endpoint}
              </div>

              <pre className="m-0 font-mono text-[10px] text-[#52525B] whitespace-pre-wrap leading-[1.5]">
                {item.code}
              </pre>

              <div className="mt-2 text-[10px] text-[#52525B]">{item.note}</div>
            </div>
          ))}
        </div>
      </SectionCard>
    </div>
  );
};
