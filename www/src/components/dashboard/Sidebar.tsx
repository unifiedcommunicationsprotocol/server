import type { ReactNode } from 'react';

export type Tab = 'overview' | 'explorer' | 'identity' | 'sessions' | 'federation' | 'bridge';

export interface SidebarProps {
  activeTab: Tab;
  onTabChange: (tab: Tab) => void;
  serverStatus: 'online' | 'offline' | 'checking';
}

const tabIcons: Record<Tab, ReactNode> = {
  overview: (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <rect x="3" y="3" width="7" height="7" rx="1" />
      <rect x="14" y="3" width="7" height="7" rx="1" />
      <rect x="14" y="14" width="7" height="7" rx="1" />
      <rect x="3" y="14" width="7" height="7" rx="1" />
    </svg>
  ),
  explorer: (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <polyline points="16 18 22 12 16 6" />
      <polyline points="8 6 2 12 8 18" />
    </svg>
  ),
  identity: (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
      <circle cx="12" cy="7" r="4" />
    </svg>
  ),
  sessions: (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <rect x="3" y="11" width="18" height="11" rx="2" />
      <path d="M7 11V7a5 5 0 0 1 10 0v4" />
    </svg>
  ),
  federation: (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <circle cx="12" cy="12" r="10" />
      <path d="M2 12h20M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10A15.3 15.3 0 0 1 12 2z" />
    </svg>
  ),
  bridge: (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="M4 9h16M4 15h16M10 3 8 21M16 3l-2 18" />
    </svg>
  ),
};

const tabLabels: Record<Tab, string> = {
  overview: 'Overview',
  explorer: 'API Explorer',
  identity: 'Identity',
  sessions: 'Sessions',
  federation: 'Federation',
  bridge: 'Bridge',
};

export const Sidebar = ({ activeTab, onTabChange, serverStatus }: SidebarProps) => {
  const statusColor =
    serverStatus === 'online' ? '#22C55E' : serverStatus === 'offline' ? '#EF4444' : '#D97706';
  const statusAnim = serverStatus === 'checking' ? 'pulse 1.5s ease infinite' : 'none';

  return (
    <nav className="w-[216px] min-w-[216px] bg-[#111113] border-r border-[#1E1E22] flex flex-col">
      {/* Logo */}
      <div className="px-[14px] py-[18px] pb-[14px] border-b border-[#1E1E22]">
        <div className="flex items-center gap-[9px]">
          <div className="w-[30px] h-[30px] rounded-[7px] bg-[#6366F1] flex items-center justify-center flex-shrink-0">
            <svg
              width="15"
              height="15"
              viewBox="0 0 24 24"
              fill="none"
              stroke="#fff"
              strokeWidth="2.5"
            >
              <path d="M12 2L2 7l10 5 10-5-10-5z" />
              <path d="M2 17l10 5 10-5" />
              <path d="M2 12l10 5 10-5" />
            </svg>
          </div>
          <div>
            <div className="text-[13px] font-bold text-[#FAFAFA] leading-tight">UCP</div>
            <div className="text-[10px] text-[#52525B] leading-[1.3]">Admin Dashboard</div>
          </div>
        </div>
      </div>

      {/* Server badge */}
      <div className="px-[10px] py-[8px] border-b border-[#1E1E22]">
        <div className="flex items-center gap-[7px] px-[9px] py-[6px] bg-[#18181B] rounded-md border border-[#1E1E22]">
          <div
            className="w-[6px] h-[6px] rounded-full flex-shrink-0"
            style={{
              backgroundColor: statusColor,
              animation: statusAnim,
            }}
          />
          <span className="text-[10px] text-[#A1A1AA] font-mono overflow-hidden text-ellipsis whitespace-nowrap">
            localhost:5150
          </span>
        </div>
      </div>

      {/* Nav items */}
      <div className="flex-1 px-2 py-2 flex flex-col gap-0.5">
        {(['overview', 'explorer', 'identity', 'sessions', 'federation', 'bridge'] as const).map(
          (tab) => {
            const isActive = activeTab === tab;
            return (
              <button
                key={tab}
                onClick={() => onTabChange(tab)}
                className={`flex items-center gap-[9px] px-[10px] py-2 rounded-md cursor-pointer transition-colors ${
                  isActive
                    ? 'bg-[rgba(99,102,241,0.09)] text-[#FAFAFA]'
                    : 'bg-transparent text-[#71717A] hover:text-[#A1A1AA]'
                }`}
              >
                {tabIcons[tab]}
                <span className="text-[13px] font-medium">{tabLabels[tab]}</span>
              </button>
            );
          }
        )}
      </div>

      {/* Footer */}
      <div className="px-[14px] py-[10px] border-t border-[#1E1E22]">
        <div className="text-[10px] text-[#3F3F46] font-mono">v0.1.0 · ucp/1.0 · Go</div>
      </div>
    </nav>
  );
};
