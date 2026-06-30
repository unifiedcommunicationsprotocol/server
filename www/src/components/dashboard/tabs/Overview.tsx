'use client';

import { useState } from 'react';
import { SectionCard } from '../primitives/SectionCard';
import { MethodBadge } from '../primitives/MethodBadge';

export interface OverviewProps {
  serverStatus: 'online' | 'offline' | 'checking';
}

export const Overview = ({ serverStatus }: OverviewProps) => {
  const [searchQuery, setSearchQuery] = useState('');
  const [isSearching, setIsSearching] = useState(false);
  const statusColor =
    serverStatus === 'online' ? '#22C55E' : serverStatus === 'offline' ? '#EF4444' : '#D97706';

  const serverStatusBig =
    serverStatus === 'online' ? 'Online' : serverStatus === 'offline' ? 'Offline' : '…';
  const serverStatusSub =
    serverStatus === 'online' ? ':5150 responding' : serverStatus === 'offline' ? 'not reachable' : 'checking…';

  return (
    <div className="fade-in">
      {/* Stat cards */}
      <div className="grid grid-cols-4 gap-2.5 mb-4">
        <div className="bg-[#111113] border border-[#1E1E22] rounded-lg p-4">
          <div className="text-[10px] text-[#52525B] uppercase tracking-[0.07em] mb-2">
            Endpoints
          </div>
          <div className="text-[26px] font-bold text-[#FAFAFA] leading-none">11</div>
          <div className="text-[11px] text-[#52525B] mt-1.5">4 public · 7 auth</div>
        </div>

        <div className="bg-[#111113] border border-[#1E1E22] rounded-lg p-4">
          <div className="text-[10px] text-[#52525B] uppercase tracking-[0.07em] mb-2">Tests</div>
          <div className="text-[26px] font-bold text-[#22C55E] leading-none">123</div>
          <div className="text-[11px] text-[#52525B] mt-1.5">all passing</div>
        </div>

        <div className="bg-[#111113] border border-[#1E1E22] rounded-lg p-4">
          <div className="text-[10px] text-[#52525B] uppercase tracking-[0.07em] mb-2">
            DB Tables
          </div>
          <div className="text-[26px] font-bold text-[#FAFAFA] leading-none">14</div>
          <div className="text-[11px] text-[#52525B] mt-1.5">PostgreSQL 18+</div>
        </div>

        <div className="bg-[#111113] border border-[#1E1E22] rounded-lg p-4">
          <div className="text-[10px] text-[#52525B] uppercase tracking-[0.07em] mb-2">
            Server
          </div>
          <div className="text-[26px] font-bold leading-none" style={{ color: statusColor }}>
            {serverStatusBig}
          </div>
          <div className="text-[11px] text-[#52525B] mt-1.5">{serverStatusSub}</div>
        </div>
      </div>

      {/* Search (Phase 2f) */}
      <div className="mb-4 bg-[#111113] border border-[#1E1E22] rounded-lg p-4">
        <div className="text-[12px] font-semibold text-[#FAFAFA] mb-3">Full-Text Search (Phase 2f)</div>
        <div className="flex gap-2">
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search messages by content…"
            className="flex-1 px-3 py-2 bg-[#18181B] border border-[#1E1E22] rounded text-[12px] text-[#FAFAFA] placeholder-[#52525B] focus:outline-none focus:border-[#6366F1]"
          />
          <button
            onClick={async () => {
              setIsSearching(true);
              // TODO: Call /api/search endpoint
              setTimeout(() => setIsSearching(false), 1000);
            }}
            disabled={!searchQuery.trim() || isSearching}
            className="px-4 py-2 bg-[#6366F1] text-white text-[11px] font-semibold rounded hover:bg-[#4F46E5] disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isSearching ? '⟳' : 'Search'}
          </button>
        </div>
        <div className="mt-3 text-[11px] text-[#52525B]">
          ○ Search endpoint not yet implemented (requires database FTS index)
        </div>
      </div>

      {/* Two-column grid */}
      <div className="grid grid-cols-2 gap-3">
        {/* Implementation Status */}
        <SectionCard title="Implementation Status">
          <div className="flex flex-col gap-[7px]">
            {[
              { icon: '✓', color: '#22C55E', text: 'Ed25519 auth + session tokens' },
              { icon: '✓', color: '#22C55E', text: 'Message routing (local + federated)' },
              { icon: '✓', color: '#22C55E', text: 'WebSocket + WebTransport' },
              { icon: '✓', color: '#22C55E', text: 'IMAP/SMTP bridge + attestation' },
              { icon: '✓', color: '#22C55E', text: 'AI metadata surfaces' },
              { icon: '✓', color: '#22C55E', text: 'Federation bundle idempotency' },
              { icon: '~', color: '#D97706', text: 'MLS encryption (RFC 9420 framework)' },
              { icon: '○', color: '#52525B', text: 'IANA UCPWelcomeExtension (blocking)', textMuted: true },
            ].map((item, idx) => (
              <div key={idx} className="flex items-center gap-2.5">
                <span className="text-[12px] leading-none" style={{ color: item.color }}>
                  {item.icon}
                </span>
                <span className={`text-[12px] ${item.textMuted ? 'text-[#52525B]' : 'text-[#A1A1AA]'}`}>
                  {item.text}
                </span>
              </div>
            ))}
          </div>
        </SectionCard>

        {/* API Surface */}
        <SectionCard title="API Surface">
          {/* Well-Known */}
          <div className="mb-3">
            <div className="text-[10px] uppercase tracking-[0.08em] text-[#52525B] mb-1.5">
              Well-Known
            </div>
            <div className="flex flex-col gap-1.5 mb-3">
              {[
                { path: '/.well-known/ucp/server-key' },
                { path: '/.well-known/ucp/identity/{addr}' },
                { path: '/.well-known/ucp/keypackages/{addr}' },
              ].map((ep, idx) => (
                <div key={idx} className="flex gap-[7px] items-baseline">
                  <MethodBadge method="GET" />
                  <span className="text-[11px] text-[#71717A] font-mono">{ep.path}</span>
                </div>
              ))}
            </div>
          </div>

          {/* Auth */}
          <div className="mb-3">
            <div className="text-[10px] uppercase tracking-[0.08em] text-[#52525B] mb-1.5">
              Auth
            </div>
            <div className="flex flex-col gap-1.5 mb-3">
              {['/auth/challenge', '/auth/session', '/auth/session/refresh'].map((path, idx) => (
                <div key={idx} className="flex gap-[7px] items-baseline">
                  <MethodBadge method="POST" />
                  <span className="text-[11px] text-[#71717A] font-mono">{path}</span>
                </div>
              ))}
            </div>
          </div>

          {/* API */}
          <div>
            <div className="text-[10px] uppercase tracking-[0.08em] text-[#52525B] mb-1.5">
              API (auth required)
            </div>
            <div className="flex flex-col gap-1.5">
              {[
                { method: 'POST', path: '/api/message/send' },
                { method: 'GET', path: '/api/inbox' },
                { method: 'POST', path: '/api/content/upload' },
                { method: 'GET', path: '/api/content/{id}' },
              ].map((ep, idx) => (
                <div key={idx} className="flex gap-[7px] items-baseline">
                  <MethodBadge method={ep.method as 'GET' | 'POST'} />
                  <span className="text-[11px] text-[#71717A] font-mono">{ep.path}</span>
                </div>
              ))}
            </div>
          </div>
        </SectionCard>
      </div>
    </div>
  );
};
