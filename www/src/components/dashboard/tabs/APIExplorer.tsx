import { useState } from 'react';
import { MethodBadge } from '../primitives/MethodBadge';
import { apiCall } from '../../../api/handlers';

interface Endpoint {
  id: string;
  group: 'wellKnown' | 'auth' | 'api';
  name: string;
  method: 'GET' | 'POST';
  path: string;
  hasBody?: boolean;
  defaultBody?: string;
}

const ENDPOINTS: Endpoint[] = [
  // Well-Known
  { id: 'server-key', group: 'wellKnown', name: 'Server Key', method: 'GET', path: '/.well-known/ucp/server-key' },
  { id: 'identity', group: 'wellKnown', name: 'Identity', method: 'GET', path: '/.well-known/ucp/identity/{address}' },
  { id: 'keypackages', group: 'wellKnown', name: 'Key Packages', method: 'GET', path: '/.well-known/ucp/keypackages/{address}' },
  { id: 'privacy', group: 'wellKnown', name: 'Privacy Policy', method: 'GET', path: '/.well-known/ucp/privacy' },
  // Auth
  { id: 'challenge', group: 'auth', name: 'Challenge', method: 'POST', path: '/auth/challenge', hasBody: true, defaultBody: '{\n  "address": "alice@example.com"\n}' },
  { id: 'session', group: 'auth', name: 'Create Session', method: 'POST', path: '/auth/session', hasBody: true, defaultBody: '{\n  "address": "alice@example.com",\n  "challenge": "<base64_challenge>",\n  "signature": "<base64_ed25519_sig>"\n}' },
  { id: 'refresh', group: 'auth', name: 'Refresh Session', method: 'POST', path: '/auth/session/refresh', hasBody: true, defaultBody: '{}' },
  // API
  { id: 'send', group: 'api', name: 'Send Message', method: 'POST', path: '/api/message/send', hasBody: true, defaultBody: '{\n  "envelope": {\n    "v": "ucp/1.0",\n    "type": "application",\n    "thread_id": "01JDTQXX..."\n  }\n}' },
  { id: 'inbox', group: 'api', name: 'Inbox', method: 'GET', path: '/api/inbox' },
  { id: 'upload', group: 'api', name: 'Upload Content', method: 'POST', path: '/api/content/upload', hasBody: true },
  { id: 'content', group: 'api', name: 'Get Content', method: 'GET', path: '/api/content/{id}' },
];

const MOCK_RESPONSES: Record<string, string> = {
  'server-key': JSON.stringify({ domain: 'localhost:5150', key: 'MCowBQYDK2VdAyEA...' }, null, 2),
  'identity': JSON.stringify({ address: 'alice@example.com', identity_key: 'MCow...', signing_keys: [{ key: 'MCow...', expires: '2026-07-28', issued: '2026-06-28', status: 'active' }] }, null, 2),
  'keypackages': JSON.stringify({ keypackages: [{ version: 'mls10', cipher_suite: 'MLS_128_DHKEMX25519_AES128GCM_SHA256_Ed25519' }] }, null, 2),
  'privacy': JSON.stringify({ enabled: false, scopes: [] }, null, 2),
  'challenge': JSON.stringify({ challenge: 'dGhpcyBpcyBhIDMyLWJ5dGUgcmFuZG9tIGNoYWxsZW5nZQ==' }, null, 2),
  'session': JSON.stringify({ session_token: 'ucp_sess_xxx...', expires_at: Math.floor(Date.now() / 1000) + 86400 }, null, 2),
  'refresh': JSON.stringify({ session_token: 'ucp_sess_yyy...', expires_at: Math.floor(Date.now() / 1000) + 86400 }, null, 2),
  'send': JSON.stringify({ envelope_id: '01JDTQXXXXXXXXXXXXXXXX' }, null, 2),
  'inbox': JSON.stringify({ messages: [{ id: '01JDTQ...', from: 'alice@example.com', thread_id: '01JDTQ...', server_ts: 1234567890 }] }, null, 2),
  'upload': JSON.stringify({ id: 'attach_xxx...', sha256: '4af...' }, null, 2),
  'content': '(binary response)',
};

export const APIExplorer = () => {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [requestBody, setRequestBody] = useState('');
  const [responseBody, setResponseBody] = useState('// Select an endpoint and click Send');
  const [responseStatus, setResponseStatus] = useState('');
  const [loading, setLoading] = useState(false);

  const selectedEndpoint = ENDPOINTS.find((ep) => ep.id === selectedId);

  const handleSelectEndpoint = (id: string) => {
    setSelectedId(id);
    const ep = ENDPOINTS.find((e) => e.id === id);
    setRequestBody(ep?.defaultBody || '');
    setResponseBody('// Select an endpoint and click Send');
    setResponseStatus('');
  };

  const handleSend = async () => {
    if (!selectedEndpoint) return;

    setLoading(true);
    try {
      let body: unknown;
      try {
        body = requestBody ? JSON.parse(requestBody) : undefined;
      } catch {
        setResponseBody('// Error: Invalid JSON in request body');
        setResponseStatus('400 Bad Request');
        setLoading(false);
        return;
      }

      const result = await apiCall(
        selectedEndpoint.method,
        selectedEndpoint.path.replace('{address}', 'alice@example.com').replace('{id}', 'test_attach_id'),
        body,
        selectedEndpoint.group === 'api' ? undefined : undefined,
        { baseUrl: 'http://localhost:5150', timeout: 3000 }
      );

      setResponseBody(JSON.stringify(result.data, null, 2));
      setResponseStatus(result.status + (result.ok ? '' : ' (server error)'));
    } catch (error) {
      // Fall back to mock if server unreachable
      setResponseBody(
        MOCK_RESPONSES[selectedEndpoint.id] ||
          JSON.stringify({ message: 'Request sent' }, null, 2)
      );
      setResponseStatus('200 OK (mock)');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fade-in h-[calc(100vh-110px)] flex overflow-hidden">
      <div className="bg-[#111113] border border-[#1E1E22] rounded-lg overflow-hidden flex w-full">
        {/* Left panel */}
        <div className="w-[256px] border-r border-[#1E1E22] overflow-y-auto flex flex-col">
          {/* Well-Known section */}
          <div className="px-3 py-1.5 text-[9px] uppercase tracking-[0.09em] text-[#52525B] bg-[#18181B] border-b border-[#1E1E22]">
            Well-Known
          </div>
          {ENDPOINTS.filter((e) => e.group === 'wellKnown').map((ep) => (
            <button
              key={ep.id}
              onClick={() => handleSelectEndpoint(ep.id)}
              className={`px-3 py-2 border-b border-[#1E1E22] text-left transition-colors ${
                selectedId === ep.id
                  ? 'bg-[rgba(99,102,241,0.09)]'
                  : 'bg-transparent hover:bg-[rgba(99,102,241,0.03)]'
              }`}
            >
              <div className="flex items-center gap-[7px] mb-0.5">
                <MethodBadge method={ep.method} />
                <span className="text-[12px] text-[#FAFAFA] font-medium">{ep.name}</span>
              </div>
              <div className="text-[10px] text-[#52525B] font-mono">{ep.path}</div>
            </button>
          ))}

          {/* Auth section */}
          <div className="px-3 py-1.5 text-[9px] uppercase tracking-[0.09em] text-[#52525B] bg-[#18181B] border-b border-[#1E1E22] border-t border-[#1E1E22]">
            Auth
          </div>
          {ENDPOINTS.filter((e) => e.group === 'auth').map((ep) => (
            <button
              key={ep.id}
              onClick={() => handleSelectEndpoint(ep.id)}
              className={`px-3 py-2 border-b border-[#1E1E22] text-left transition-colors ${
                selectedId === ep.id
                  ? 'bg-[rgba(99,102,241,0.09)]'
                  : 'bg-transparent hover:bg-[rgba(99,102,241,0.03)]'
              }`}
            >
              <div className="flex items-center gap-[7px] mb-0.5">
                <MethodBadge method={ep.method} />
                <span className="text-[12px] text-[#FAFAFA] font-medium">{ep.name}</span>
              </div>
              <div className="text-[10px] text-[#52525B] font-mono">{ep.path}</div>
            </button>
          ))}

          {/* API section */}
          <div className="px-3 py-1.5 text-[9px] uppercase tracking-[0.09em] text-[#52525B] bg-[#18181B] border-b border-[#1E1E22] border-t border-[#1E1E22]">
            API
          </div>
          {ENDPOINTS.filter((e) => e.group === 'api').map((ep) => (
            <button
              key={ep.id}
              onClick={() => handleSelectEndpoint(ep.id)}
              className={`px-3 py-2 border-b border-[#1E1E22] text-left transition-colors ${
                selectedId === ep.id
                  ? 'bg-[rgba(99,102,241,0.09)]'
                  : 'bg-transparent hover:bg-[rgba(99,102,241,0.03)]'
              }`}
            >
              <div className="flex items-center gap-[7px] mb-0.5">
                <MethodBadge method={ep.method} />
                <span className="text-[12px] text-[#FAFAFA] font-medium">{ep.name}</span>
              </div>
              <div className="text-[10px] text-[#52525B] font-mono">{ep.path}</div>
            </button>
          ))}
        </div>

        {/* Right panel */}
        <div className="flex-1 flex flex-col overflow-hidden">
          {/* Request bar */}
          <div className="px-4 py-3.5 border-b border-[#1E1E22]">
            <div className="flex gap-2 items-center mb-2.5">
              {selectedEndpoint && <MethodBadge method={selectedEndpoint.method} />}
              <input
                type="text"
                value={selectedEndpoint?.path || ''}
                readOnly
                placeholder="← select an endpoint"
                className="flex-1 px-2.5 py-1.5 bg-[#09090B] border border-[#1E1E22] rounded-md text-[11px] text-[#A1A1AA] font-mono outline-none"
              />
              <button
                onClick={handleSend}
                disabled={!selectedEndpoint || loading}
                className="px-4 py-1.5 bg-[#6366F1] border-0 rounded-md text-white text-[12px] font-semibold disabled:opacity-50"
              >
                {loading ? 'Sending…' : 'Send'}
              </button>
            </div>

            {selectedEndpoint?.hasBody && (
              <>
                <div className="text-[9px] text-[#52525B] uppercase tracking-[0.07em] mb-1.5">
                  Request Body
                </div>
                <textarea
                  value={requestBody}
                  onChange={(e) => setRequestBody(e.target.value)}
                  className="w-full h-[100px] bg-[#09090B] border border-[#1E1E22] rounded-md text-[#FAFAFA] font-mono text-[11px] px-2 py-2 resize-none outline-none focus:border-[#6366F1] transition-colors"
                />
              </>
            )}
          </div>

          {/* Response area */}
          <div className="flex-1 overflow-y-auto px-4 py-3.5">
            <div className="flex justify-between items-center mb-[7px]">
              <div className="text-[9px] text-[#52525B] uppercase tracking-[0.07em]">Response</div>
              {responseStatus && (
                <span
                  className="text-[10px] font-mono"
                  style={{
                    color: responseStatus.includes('200')
                      ? '#22C55E'
                      : responseStatus.includes('4') || responseStatus.includes('5')
                        ? '#EF4444'
                        : '#52525B',
                  }}
                >
                  {responseStatus}
                </span>
              )}
            </div>
            <pre className="m-0 bg-[#09090B] border border-[#1E1E22] rounded-md px-3 py-3 font-mono text-[11px] text-[#A1A1AA] whitespace-pre-wrap break-all min-h-[100px] leading-[1.6]">
              {responseBody}
            </pre>
          </div>
        </div>
      </div>
    </div>
  );
};
