import { Hono } from 'hono';
import { serveStatic } from 'hono/bun';

const app = new Hono();

// Serve static files (public directory)
app.use('/public/*', serveStatic({ root: './' }));

// Development: check if dist exists, otherwise show helpful message
// Production: serve from dist (after build)
app.get('/*', async (c) => {
  const indexPath = './dist/index.html';
  try {
    const file = Bun.file(indexPath);
    if (await file.exists()) {
      return c.html(await file.text());
    }
  } catch {
    // Ignore
  }

  // Fallback message for development
  return c.html(`
    <!DOCTYPE html>
    <html>
      <head>
        <title>UCP Admin Dashboard</title>
        <style>
          body { font-family: -apple-system, sans-serif; max-width: 600px; margin: 50px auto; padding: 20px; color: #333; }
          h1 { color: #6366F1; }
          p { line-height: 1.6; }
          code { background: #f0f0f0; padding: 2px 6px; border-radius: 3px; }
        </style>
      </head>
      <body>
        <h1>🚀 UCP Admin Dashboard</h1>
        <p>Development mode: React app not yet built.</p>
        <p>To get started:</p>
        <ol>
          <li>Install dependencies: <code>bun install</code></li>
          <li>Build the app: <code>bun run build</code></li>
          <li>Start the dev server: <code>bun run dev</code></li>
        </ol>
        <p>After building, reload this page.</p>
      </body>
    </html>
  `);
});

const port = parseInt(process.env.PORT || '6002');
console.log(`🚀 Admin Dashboard listening on http://localhost:${port}`);

export default {
  port,
  fetch: app.fetch,
};
