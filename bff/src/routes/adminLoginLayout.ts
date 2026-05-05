// BAL-01: standalone login page HTML renderer — no session, no bearer token, no nav chrome
const LOGIN_STYLES = `
  :root {
    color-scheme: light;
    --bg: #f6f7f9;
    --panel: #ffffff;
    --text: #172033;
    --muted: #5c667a;
    --line: #d9dee8;
    --accent: #1868db;
    --accent-dark: #0f4fa8;
    --error-bg: #fef2f2;
    --error-border: #fca5a5;
    --error-text: #991b1b;
  }
  * { box-sizing: border-box; }
  body {
    margin: 0;
    min-height: 100vh;
    background: var(--bg);
    color: var(--text);
    font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .login-card {
    width: 100%;
    max-width: 380px;
    background: var(--panel);
    border: 1px solid var(--line);
    border-radius: 10px;
    padding: 32px 28px;
  }
  .login-title { margin: 0 0 4px; font-size: 18px; font-weight: 700; }
  .login-subtitle { margin: 0 0 24px; font-size: 13px; color: var(--muted); }
  .field { display: flex; flex-direction: column; gap: 4px; margin-bottom: 16px; }
  .field label { font-size: 13px; font-weight: 600; }
  .field input {
    height: 36px;
    border: 1px solid var(--line);
    border-radius: 6px;
    padding: 0 10px;
    font-size: 14px;
    color: var(--text);
    background: #fff;
    width: 100%;
  }
  .field input:focus { outline: 2px solid var(--accent); outline-offset: 1px; }
  .submit-btn {
    width: 100%;
    height: 38px;
    background: var(--accent);
    color: #fff;
    border: 0;
    border-radius: 6px;
    font-size: 14px;
    font-weight: 700;
    cursor: pointer;
    margin-top: 4px;
  }
  .submit-btn:hover { background: var(--accent-dark); }
  .error-banner {
    background: var(--error-bg);
    border: 1px solid var(--error-border);
    border-radius: 6px;
    color: var(--error-text);
    font-size: 13px;
    padding: 8px 12px;
    margin-bottom: 16px;
  }
`;

export function adminLoginPage(errorMsg?: string): string {
  const errorHtml = errorMsg
    ? `<div class="error-banner" role="alert">${errorMsg}</div>`
    : '';
  return `<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Sign in — FenixCRM Admin</title>
    <style>${LOGIN_STYLES}</style>
  </head>
  <body>
    <div class="login-card">
      <h1 class="login-title">FenixCRM Admin</h1>
      <p class="login-subtitle">Sign in to continue</p>
      ${errorHtml}
      <form method="POST" action="/bff/admin/login" autocomplete="on">
        <div class="field">
          <label for="email">Email</label>
          <input id="email" name="email" type="email" autocomplete="email" required placeholder="operator@company.com">
        </div>
        <div class="field">
          <label for="password">Password</label>
          <input id="password" name="password" type="password" autocomplete="current-password" required>
        </div>
        <button class="submit-btn" type="submit">Sign in</button>
      </form>
    </div>
  </body>
</html>`;
}
