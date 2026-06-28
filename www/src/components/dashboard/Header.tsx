import type { Tab } from './Sidebar';

export interface HeaderProps {
  pageTitle: string;
  activeTab: Tab;
  sessionToken: string;
  onTokenChange: (token: string) => void;
  serverStatus: 'online' | 'offline' | 'checking';
}

export const Header = ({
  pageTitle,
  activeTab,
  sessionToken,
  onTokenChange,
  serverStatus,
}: HeaderProps) => {
  const showTokenInput = activeTab === 'explorer';

  const statusLabel =
    serverStatus === 'online'
      ? '● online'
      : serverStatus === 'offline'
        ? '● offline'
        : '◌ checking…';

  return (
    <header className="h-[50px] min-h-[50px] bg-[#111113] border-b border-[#1E1E22] flex items-center justify-between px-5">
      <h1 className="m-0 text-[14px] font-semibold text-[#FAFAFA]">{pageTitle}</h1>
      <div className="flex items-center gap-2.5">
        {showTokenInput && (
          <input
            type="text"
            value={sessionToken}
            onChange={(e) => onTokenChange(e.target.value)}
            placeholder="Bearer token (for auth endpoints)…"
            className="w-[240px] px-2.5 py-1.5 bg-[#18181B] border border-[#1E1E22] rounded-md text-[#FAFAFA] font-mono text-[10px] outline-none focus:border-[#6366F1] transition-colors"
          />
        )}
        <div className="text-[10px] text-[#3F3F46] font-mono">{statusLabel}</div>
      </div>
    </header>
  );
};
