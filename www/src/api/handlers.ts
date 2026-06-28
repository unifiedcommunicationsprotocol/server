export interface APIConfig {
  baseUrl: string;
  timeout: number;
}

const defaultConfig: APIConfig = {
  baseUrl: 'http://localhost:5150',
  timeout: 5000,
};

async function fetchWithTimeout(
  url: string,
  options?: RequestInit,
  timeoutMs: number = defaultConfig.timeout
) {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), timeoutMs);

  try {
    const response = await fetch(url, {
      ...options,
      signal: controller.signal,
    });
    return response;
  } finally {
    clearTimeout(timeoutId);
  }
}

export async function getServerKey(config = defaultConfig) {
  try {
    const response = await fetchWithTimeout(
      `${config.baseUrl}/.well-known/ucp/server-key`,
      { method: 'GET' },
      config.timeout
    );

    if (response.ok) {
      return await response.json();
    }
    return null;
  } catch {
    return null;
  }
}

export async function getIdentity(address: string, config = defaultConfig) {
  try {
    const encodedAddr = encodeURIComponent(address);
    const response = await fetchWithTimeout(
      `${config.baseUrl}/.well-known/ucp/identity/${encodedAddr}`,
      { method: 'GET' },
      config.timeout
    );

    if (response.ok) {
      return await response.json();
    }
    return { error: `Failed to resolve ${address}` };
  } catch (error) {
    return { error: String(error) };
  }
}

export async function getKeyPackages(address: string, config = defaultConfig) {
  try {
    const encodedAddr = encodeURIComponent(address);
    const response = await fetchWithTimeout(
      `${config.baseUrl}/.well-known/ucp/keypackages/${encodedAddr}`,
      { method: 'GET' },
      config.timeout
    );

    if (response.ok) {
      return await response.json();
    }
    return null;
  } catch {
    return null;
  }
}

export async function getPrivacy(config = defaultConfig) {
  try {
    const response = await fetchWithTimeout(
      `${config.baseUrl}/.well-known/ucp/privacy`,
      { method: 'GET' },
      config.timeout
    );

    if (response.ok) {
      return await response.json();
    }
    return null;
  } catch {
    return null;
  }
}

export async function getChallenge(address: string, config = defaultConfig) {
  try {
    const response = await fetchWithTimeout(
      `${config.baseUrl}/auth/challenge`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ address }),
      },
      config.timeout
    );

    if (response.ok) {
      return await response.json();
    }
    return { error: 'Failed to get challenge' };
  } catch (error) {
    return { error: String(error) };
  }
}

export async function createSession(
  address: string,
  challenge: string,
  signature: string,
  config = defaultConfig
) {
  try {
    const response = await fetchWithTimeout(
      `${config.baseUrl}/auth/session`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ address, challenge, signature }),
      },
      config.timeout
    );

    if (response.ok) {
      return await response.json();
    }
    return { error: 'Failed to create session' };
  } catch (error) {
    return { error: String(error) };
  }
}

export async function refreshSession(sessionToken: string, config = defaultConfig) {
  try {
    const response = await fetchWithTimeout(
      `${config.baseUrl}/auth/session/refresh`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${sessionToken}`,
        },
        body: JSON.stringify({}),
      },
      config.timeout
    );

    if (response.ok) {
      return await response.json();
    }
    return { error: 'Failed to refresh session' };
  } catch (error) {
    return { error: String(error) };
  }
}

export async function sendMessage(
  envelope: unknown,
  sessionToken: string,
  config = defaultConfig
) {
  try {
    const response = await fetchWithTimeout(
      `${config.baseUrl}/api/message/send`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${sessionToken}`,
        },
        body: JSON.stringify({ envelope }),
      },
      config.timeout
    );

    if (response.ok) {
      return await response.json();
    }
    return { error: `HTTP ${response.status}` };
  } catch (error) {
    return { error: String(error) };
  }
}

export async function getInbox(sessionToken: string, query?: string, config = defaultConfig) {
  try {
    const params = new URLSearchParams();
    if (query) params.append('query', query);

    const response = await fetchWithTimeout(
      `${config.baseUrl}/api/inbox?${params.toString()}`,
      {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${sessionToken}`,
        },
      },
      config.timeout
    );

    if (response.ok) {
      return await response.json();
    }
    return { error: `HTTP ${response.status}`, messages: [] };
  } catch (error) {
    return { error: String(error), messages: [] };
  }
}

export async function uploadContent(
  file: Blob,
  sessionToken: string,
  config = defaultConfig
) {
  try {
    const formData = new FormData();
    formData.append('file', file);

    const response = await fetchWithTimeout(
      `${config.baseUrl}/api/content/upload`,
      {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${sessionToken}`,
        },
        body: formData,
      },
      config.timeout
    );

    if (response.ok) {
      return await response.json();
    }
    return { error: `HTTP ${response.status}` };
  } catch (error) {
    return { error: String(error) };
  }
}

export async function getContent(id: string, sessionToken: string, config = defaultConfig) {
  try {
    const response = await fetchWithTimeout(
      `${config.baseUrl}/api/content/${encodeURIComponent(id)}`,
      {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${sessionToken}`,
        },
      },
      config.timeout
    );

    if (response.ok) {
      return await response.blob();
    }
    return null;
  } catch {
    return null;
  }
}

export async function apiCall(
  method: 'GET' | 'POST',
  path: string,
  body?: unknown,
  sessionToken?: string,
  config = defaultConfig
) {
  try {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };

    if (sessionToken) {
      headers['Authorization'] = `Bearer ${sessionToken}`;
    }

    const options: RequestInit = {
      method,
      headers,
    };

    if (body) {
      options.body = JSON.stringify(body);
    }

    const response = await fetchWithTimeout(
      `${config.baseUrl}${path}`,
      options,
      config.timeout
    );

    const statusText = response.statusText || `HTTP ${response.status}`;
    const data = await response.json().catch(() => ({}));

    return {
      status: `${response.status} ${statusText}`,
      data,
      ok: response.ok,
    };
  } catch (error) {
    return {
      status: 'Error',
      data: { error: String(error) },
      ok: false,
    };
  }
}
