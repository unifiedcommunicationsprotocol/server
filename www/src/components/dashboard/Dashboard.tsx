import { useState, useEffect } from 'react';
import { Sidebar, type Tab } from './Sidebar';
import { Header } from './Header';
import { Overview } from './tabs/Overview';
import { APIExplorer } from './tabs/APIExplorer';
import { Identity } from './tabs/Identity';
import { Sessions } from './tabs/Sessions';
import { Federation } from './tabs/Federation';
import { Bridge } from './tabs/Bridge';
import { ComposeModal } from './ComposeModal';
import { getServerKey } from '../../api/handlers';

export const Dashboard = () => {
  const [activeTab, setActiveTab] = useState<Tab>('overview');
  const [serverStatus, setServerStatus] = useState<'online' | 'offline' | 'checking'>('checking');
  const [sessionToken, setSessionToken] = useState('');
  const [isComposeOpen, setIsComposeOpen] = useState(false);
  const [senderAddress, setSenderAddress] = useState('alice@example.com'); // TODO: from identity

  // Check server status on mount
  useEffect(() => {
    const checkServer = async () => {
      const key = await getServerKey();
      if (key) {
        setServerStatus('online');
      } else {
        setServerStatus('offline');
      }
    };

    checkServer();
  }, []);

  const pageTitles: Record<Tab, string> = {
    overview: 'Overview',
    explorer: 'API Explorer',
    identity: 'Identity & Keys',
    sessions: 'Sessions',
    federation: 'Federation',
    bridge: 'Bridge',
  };

  return (
    <div className="flex h-screen overflow-hidden bg-[#09090B] text-[#FAFAFA] font-['Space_Grotesk']">
      <Sidebar activeTab={activeTab} onTabChange={setActiveTab} serverStatus={serverStatus} />

      <div className="flex-1 flex flex-col overflow-hidden min-w-0">
        <Header
          pageTitle={pageTitles[activeTab]}
          activeTab={activeTab}
          sessionToken={sessionToken}
          onTokenChange={setSessionToken}
          serverStatus={serverStatus}
          onCompose={() => setIsComposeOpen(true)}
        />

        <main className="flex-1 overflow-y-auto px-5 py-5">
          {activeTab === 'overview' && <Overview serverStatus={serverStatus} />}
          {activeTab === 'explorer' && <APIExplorer />}
          {activeTab === 'identity' && <Identity />}
          {activeTab === 'sessions' && <Sessions />}
          {activeTab === 'federation' && <Federation />}
          {activeTab === 'bridge' && <Bridge />}
        </main>
      </div>

      <ComposeModal
        isOpen={isComposeOpen}
        onClose={() => setIsComposeOpen(false)}
        sessionToken={sessionToken}
        senderAddress={senderAddress}
      />
    </div>
  );
};
