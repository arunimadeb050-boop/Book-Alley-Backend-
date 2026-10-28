// NEW FILE: shared helpers used by every page (load it BEFORE the page's own script).

// Talks to the Go backend. Same server serves the pages, so plain paths like "/login" work.
// Automatically adds the login token when there is one.
async function api(path, method = 'GET', body) {
  const headers = { 'Content-Type': 'application/json' };
  const token = localStorage.getItem('token');
  if (token) headers['Authorization'] = 'Bearer ' + token;
  const res = await fetch(path, { method, headers, body: body ? JSON.stringify(body) : undefined });
  let data = null;
  try { data = await res.json(); } catch (e) {}
  return { ok: res.ok, status: res.status, data };
}

// Stops HTML injection when showing text from the database.
const esc = (s) => String(s ?? '').replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));

// Pages that need a login call this. No token -> go to the login page.
function requireLogin() {
  if (!localStorage.getItem('token')) { location.href = 'login.html'; return false; }
  return true;
}

// Theme toggle (moved here from each page, same behaviour).
(function () {
  const root = document.documentElement;
  const btn = document.getElementById('themeToggle');
  const saved = localStorage.getItem('theme');
  if (saved) root.setAttribute('data-theme', saved);
  const label = () => { if (btn) btn.textContent = root.getAttribute('data-theme') === 'dark' ? 'Light mode' : 'Dark mode'; };
  label();
  if (btn) btn.addEventListener('click', () => {
    const next = root.getAttribute('data-theme') === 'dark' ? 'light' : 'dark';
    root.setAttribute('data-theme', next);
    localStorage.setItem('theme', next);
    label();
  });

  // Nav: show "Logout" instead of "Login" when logged in.
  const loginLink = document.querySelector('nav a[href="login.html"]');
  // NEW: when logged in, add links to the new pages (cart, orders, requests, profile)
  if (loginLink && localStorage.getItem('token')) {
    [['profile.html', 'Profile'], ['requests.html', 'Requests'], ['orders.html', 'Orders'], ['cart.html', 'Cart']].forEach(([href, text]) => {
      const a = document.createElement('a');
      a.href = href; a.textContent = text;
      loginLink.before(a);
    });
  }
  if (loginLink && localStorage.getItem('token')) {
    loginLink.textContent = 'Logout';
    loginLink.addEventListener('click', (e) => {
      e.preventDefault();
      localStorage.removeItem('token');
      location.href = 'index.html';
    });
  }
})();
