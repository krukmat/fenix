// BFF-ADMIN-01: shared HTMX chrome for the admin surface
const HTMX_CDN = 'https://unpkg.com/htmx.org@2.0.4';

const NAV_LINKS = [
  { href: '/bff/admin/workflows',  label: 'Workflows' },
  { href: '/bff/admin/agent-runs', label: 'Agent Runs' },
  { href: '/bff/admin/approvals',  label: 'Approvals' },
  { href: '/bff/admin/audit',      label: 'Audit' },
  { href: '/bff/admin/policy',     label: 'Policy' },
  { href: '/bff/admin/tools',      label: 'Tools' },
  { href: '/bff/admin/metrics',    label: 'Metrics' },
] as const;

const navLinksHtml = NAV_LINKS.map(
  ({ href, label }) => `<a class="nav-link" href="${href}">${label}</a>`,
).join('\n      ');

const STYLES = `
  :root {
    color-scheme: light;
    --bg: #f6f7f9;
    --panel: #ffffff;
    --text: #172033;
    --muted: #5c667a;
    --line: #d9dee8;
    --accent: #1868db;
    --accent-dark: #0f4fa8;
    --nav-width: 200px;
  }
  * { box-sizing: border-box; }
  body {
    margin: 0;
    min-height: 100vh;
    background: var(--bg);
    color: var(--text);
    font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    display: grid;
    grid-template-rows: 56px 1fr;
    grid-template-columns: var(--nav-width) 1fr;
    grid-template-areas: "header header" "sidebar main";
  }
  header {
    grid-area: header;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    padding: 0 20px;
    border-bottom: 1px solid var(--line);
    background: var(--panel);
  }
  .header-left { display: flex; align-items: center; gap: 12px; }
  h1 { margin: 0; font-size: 18px; font-weight: 700; }
  .workspace-badge {
    display: inline-flex;
    align-items: center;
    padding: 2px 10px;
    border-radius: 999px;
    background: var(--bg);
    border: 1px solid var(--line);
    color: var(--muted);
    font-size: 12px;
    font-weight: 600;
  }
  .header-right { display: flex; align-items: center; gap: 8px; }
  .auth-bar { display: flex; align-items: center; gap: 8px; }
  .auth-bar input {
    width: 220px;
    height: 32px;
    border: 1px solid var(--line);
    border-radius: 6px;
    padding: 0 10px;
    color: var(--text);
    font-size: 13px;
  }
  .auth-bar button {
    height: 32px;
    border: 0;
    border-radius: 6px;
    padding: 0 12px;
    background: var(--accent);
    color: #fff;
    font-size: 13px;
    font-weight: 700;
    cursor: pointer;
  }
  .auth-bar button:hover { background: var(--accent-dark); }
  .sign-out-btn {
    height: 32px;
    border: 1px solid var(--line);
    border-radius: 6px;
    padding: 0 12px;
    background: transparent;
    color: var(--muted);
    font-size: 13px;
    cursor: pointer;
  }
  .sign-out-btn:hover { background: var(--bg); color: var(--text); }
  nav {
    grid-area: sidebar;
    border-right: 1px solid var(--line);
    background: var(--panel);
    padding: 16px 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .nav-link {
    display: block;
    padding: 8px 20px;
    color: var(--muted);
    text-decoration: none;
    font-size: 14px;
    font-weight: 500;
    border-left: 3px solid transparent;
  }
  .nav-link:hover { color: var(--text); background: var(--bg); border-left-color: var(--line); }
  .nav-link.active { color: var(--accent); border-left-color: var(--accent); font-weight: 700; }
  main {
    grid-area: main;
    padding: 24px;
    min-height: 0;
    overflow: auto;
  }
  .page-title { margin: 0 0 20px; font-size: 22px; font-weight: 700; }
  .placeholder { color: var(--muted); font-size: 14px; margin-top: 8px; }
  @media (max-width: 768px) {
    body { grid-template-columns: 1fr; grid-template-areas: "header" "main"; }
    nav { display: none; }
  }
`;

const BEARER_RELAY_SCRIPT = `
  (function () {
    var tokenInput = document.getElementById('admin-token');
    var authForm = document.getElementById('admin-auth-form');
    if (tokenInput) {
      tokenInput.value = localStorage.getItem('fenix.admin.bearerToken') || '';
    }
    if (authForm) {
      authForm.addEventListener('submit', function (event) {
        event.preventDefault();
        var val = tokenInput ? tokenInput.value.trim() : '';
        localStorage.setItem('fenix.admin.bearerToken', val);
      });
    }
    document.body.addEventListener('htmx:configRequest', function (event) {
      var token = localStorage.getItem('fenix.admin.bearerToken');
      if (token) {
        event.detail.headers.Authorization = 'Bearer ' + token;
      }
    });
    var signOut = document.getElementById('admin-sign-out');
    if (signOut) {
      signOut.addEventListener('click', function () {
        localStorage.removeItem('fenix.admin.bearerToken');
        if (tokenInput) tokenInput.value = '';
      });
    }
  }());
`;

export function adminLayout(title: string, bodyContent: string): string {
  return `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>${title} — FenixCRM Admin</title>
    <script src="${HTMX_CDN}"></script>
    <style>${STYLES}</style>
  </head>
  <body>
    <header>
      <div class="header-left">
        <h1>FenixCRM Admin</h1>
        <span class="workspace-badge" id="admin-workspace-badge">workspace</span>
      </div>
      <div class="header-right">
        <form class="auth-bar" id="admin-auth-form">
          <input id="admin-token" type="password" autocomplete="off" placeholder="Bearer token" aria-label="Bearer token">
          <button type="submit">Use Token</button>
        </form>
        <button class="sign-out-btn" id="admin-sign-out" type="button">sign-out</button>
      </div>
    </header>
    <nav aria-label="Admin navigation">
      ${navLinksHtml}
    </nav>
    <main>
      ${bodyContent}
    </main>
    <script>${BEARER_RELAY_SCRIPT}</script>
  </body>
</html>`;
}
