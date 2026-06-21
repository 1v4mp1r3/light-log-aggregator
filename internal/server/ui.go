package server

const uiHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Loglite</title>
  <style>
    :root {
      color-scheme: dark;
      --bg: #101114;
      --panel: #171a1f;
      --panel-2: #1d2229;
      --text: #e6edf3;
      --muted: #8d99a8;
      --line: #2c333d;
      --accent: #4dd5a1;
      --warn: #f5c542;
      --bad: #ff6b7a;
      --radius: 8px;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      min-height: 100vh;
      background:
        linear-gradient(180deg, rgba(77,213,161,.08), transparent 280px),
        var(--bg);
      color: var(--text);
      font: 14px/1.5 ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    }
    main { max-width: 1240px; margin: 0 auto; padding: 28px 18px 44px; }
    header { display: flex; align-items: end; justify-content: space-between; gap: 20px; margin-bottom: 22px; }
    h1 { margin: 0; font-size: 30px; letter-spacing: 0; }
    .sub { color: var(--muted); margin-top: 4px; }
    .metrics { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 10px; min-width: 360px; }
    .metric, .query, .table-wrap { background: rgba(23,26,31,.92); border: 1px solid var(--line); border-radius: var(--radius); }
    .metric { padding: 12px 14px; }
    .metric span { color: var(--muted); display: block; font-size: 12px; }
    .metric b { font-size: 22px; }
    .query { padding: 14px; margin-bottom: 14px; }
    form { display: grid; grid-template-columns: 1.8fr .9fr .9fr .5fr auto; gap: 10px; align-items: end; }
    label { display: grid; gap: 5px; color: var(--muted); font-size: 12px; }
    input, select, button {
      height: 38px;
      border: 1px solid var(--line);
      border-radius: 6px;
      background: var(--panel-2);
      color: var(--text);
      padding: 0 10px;
      font: inherit;
    }
    input:focus, select:focus, button:focus { outline: 2px solid rgba(77,213,161,.45); outline-offset: 1px; }
    button { background: var(--accent); border-color: var(--accent); color: #05120d; font-weight: 700; cursor: pointer; }
    .table-wrap { overflow: hidden; }
    table { width: 100%; border-collapse: collapse; }
    th, td { padding: 10px 12px; border-bottom: 1px solid var(--line); text-align: left; vertical-align: top; }
    th { color: var(--muted); font-size: 12px; text-transform: uppercase; letter-spacing: .06em; background: #15181d; }
    tr:last-child td { border-bottom: 0; }
    code { color: #b7c7d8; overflow-wrap: anywhere; }
    .level-error { color: var(--bad); font-weight: 700; }
    .level-warn { color: var(--warn); font-weight: 700; }
    .level-info { color: var(--accent); font-weight: 700; }
    .empty { color: var(--muted); text-align: center; padding: 26px; }
    @media (max-width: 900px) {
      header { display: block; }
      .metrics { grid-template-columns: repeat(3, 1fr); min-width: 0; margin-top: 16px; }
      form { grid-template-columns: 1fr 1fr; }
      button { grid-column: span 2; }
      .table-wrap { overflow-x: auto; }
    }
  </style>
</head>
<body>
  <main>
    <header>
      <div>
        <h1>Loglite</h1>
        <div class="sub">Compact log ingestion, redaction, search, and retention for owned services.</div>
      </div>
      <section class="metrics" aria-label="metrics">
        <div class="metric"><span>Entries</span><b id="entries">0</b></div>
        <div class="metric"><span>Ingested</span><b id="ingested">0</b></div>
        <div class="metric"><span>Redactions</span><b id="redactions">0</b></div>
      </section>
    </header>

    <section class="query">
      <form id="search">
        <label>Text
          <input name="q" placeholder="failed password, timeout, service=api">
        </label>
        <label>Label
          <input name="label" placeholder="service=api">
        </label>
        <label>Level
          <select name="level">
            <option value="">any</option>
            <option>info</option>
            <option>warn</option>
            <option>error</option>
            <option>debug</option>
          </select>
        </label>
        <label>Since
          <input name="since" placeholder="1h">
        </label>
        <button type="submit">Search</button>
      </form>
    </section>

    <section class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Time</th>
            <th>Level</th>
            <th>Message</th>
            <th>Labels</th>
            <th>Fields</th>
          </tr>
        </thead>
        <tbody id="rows"><tr><td class="empty" colspan="5">No results yet.</td></tr></tbody>
      </table>
    </section>
  </main>
  <script>
    const rows = document.querySelector('#rows');
    const form = document.querySelector('#search');
    const esc = value => String(value ?? '').replace(/[&<>"']/g, ch => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[ch]));
    const pairs = value => Object.entries(value || {}).map(([k,v]) => '<code>' + esc(k) + '=' + esc(v) + '</code>').join('<br>');

    async function metrics() {
      const text = await fetch('/api/metrics').then(r => r.text());
      const get = name => (text.match(new RegExp(name + ' ([0-9]+)')) || [0,0])[1];
      document.querySelector('#entries').textContent = get('loglite_entries');
      document.querySelector('#ingested').textContent = get('loglite_ingested_total');
      document.querySelector('#redactions').textContent = get('loglite_redactions_total');
    }

    async function search(event) {
      if (event) event.preventDefault();
      const data = new FormData(form);
      const params = new URLSearchParams();
      for (const [key, value] of data.entries()) if (value) params.append(key, value);
      params.set('limit', '200');
      const payload = await fetch('/api/search?' + params.toString()).then(r => r.json());
      if (!payload.results || payload.results.length === 0) {
        rows.innerHTML = '<tr><td class="empty" colspan="5">No matching events.</td></tr>';
        return;
      }
      rows.innerHTML = payload.results.map(entry =>
        '<tr>' +
          '<td><code>' + esc(new Date(entry.timestamp).toLocaleString()) + '</code></td>' +
          '<td class="level-' + esc(entry.level) + '">' + esc(entry.level) + '</td>' +
          '<td>' + esc(entry.message) + '</td>' +
          '<td>' + pairs(entry.labels) + '</td>' +
          '<td>' + pairs(entry.fields) + '</td>' +
        '</tr>'
      ).join('');
    }

    form.addEventListener('submit', search);
    metrics();
    search();
    setInterval(metrics, 5000);
  </script>
</body>
</html>`
