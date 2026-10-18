/* WebUI của module: nói chuyện với ctl.sh qua cầu nối exec của KernelSU / APatch. */
(() => {
  'use strict';

  const MODDIR = '/data/adb/modules/adguardhome_superuser';
  const CTL = `sh ${MODDIR}/scripts/ctl.sh`;

  const PRESETS = {
    cloudflare: ['https://1.1.1.1/dns-query', 'https://1.0.0.1/dns-query'],
    google: ['https://8.8.8.8/dns-query', 'https://8.8.4.4/dns-query'],
    quad9: ['https://9.9.9.9/dns-query', 'https://149.112.112.112/dns-query'],
    adguard: ['https://dns.adguard-dns.com/dns-query'],
  };
  // Phải khớp với is_upstream / cmd_set_setting trong ctl.sh (script kiểm tra lại lần nữa).
  const UPSTREAM_RE = /^[\]A-Za-z0-9:/._@?=&%#~+[-]+$/;
  const ADDR_LIST_RE = /^[0-9A-Fa-f:./ ]*$/;

  const $ = (sel, root = document) => root.querySelector(sel);
  const $$ = (sel, root = document) => Array.from(root.querySelectorAll(sel));

  const state = {
    lang: 'vi',
    status: null,
    version: '',
    busy: false,
    upstream: [],
    upstreamLoaded: false,
    filters: [],
    filtersLoaded: false,
    formLoaded: false,
    logKind: 'core',
    debugText: '',
    tab: 'overview',
  };

  // ------------------------------------------------------------ lưu trữ cục bộ
  const store = {
    get(k) { try { return localStorage.getItem(k); } catch (e) { return null; } },
    set(k, v) { try { localStorage.setItem(k, v); } catch (e) { /* bỏ qua */ } },
  };

  // ------------------------------------------------------------ i18n
  function t(key, vars) {
    const dict = window.I18N || {};
    let s = (dict[state.lang] && dict[state.lang][key]) || (dict.vi && dict.vi[key]) || key;
    if (vars) for (const k of Object.keys(vars)) s = s.split(`{${k}}`).join(vars[k]);
    return s;
  }

  function applyI18n() {
    document.documentElement.lang = state.lang;
    $$('[data-i18n]').forEach((el) => { el.textContent = t(el.dataset.i18n); });
    $$('[data-i18n-placeholder]').forEach((el) => { el.placeholder = t(el.dataset.i18nPlaceholder); });
    $$('[data-lang]').forEach((el) => el.classList.toggle('active', el.dataset.lang === state.lang));
  }

  function setLang(lang, persist) {
    if (!window.I18N || !window.I18N[lang]) return;
    state.lang = lang;
    store.set('lang', lang);
    applyI18n();
    render();
    if (persist && bridge()) ctl(['set-setting', 'language', lang]);
  }

  // ------------------------------------------------------------ cầu nối root
  function bridge() {
    if (typeof ksu !== 'undefined' && ksu && typeof ksu.exec === 'function') return ksu;
    if (typeof ap !== 'undefined' && ap && typeof ap.exec === 'function') return ap;
    return null;
  }

  let execSeq = 0;
  function exec(cmd) {
    return new Promise((resolve) => {
      const b = bridge();
      if (!b) { resolve({ errno: -1, stdout: '', stderr: 'no bridge' }); return; }
      const cb = `agh_exec_${Date.now()}_${execSeq++}`;
      window[cb] = (errno, stdout, stderr) => {
        delete window[cb];
        resolve({ errno: Number(errno), stdout: String(stdout || ''), stderr: String(stderr || '') });
      };
      try {
        b.exec(cmd, JSON.stringify({}), cb);
      } catch (e) {
        delete window[cb];
        resolve({ errno: -1, stdout: '', stderr: String(e) });
      }
    });
  }

  const q = (s) => `'${String(s).replace(/'/g, `'\\''`)}'`;
  const ctl = (args) => exec(`${CTL} ${args.map(q).join(' ')}`);

  function parseKV(text) {
    const out = {};
    text.split('\n').forEach((line) => {
      const m = /^([a-z_0-9]+)=(.*)$/.exec(line.replace(/\r$/, ''));
      if (m) out[m[1]] = m[2];
    });
    return out;
  }

  // ------------------------------------------------------------ giao diện chung
  let toastTimer = 0;
  function toast(msg, isErr) {
    const el = $('#toast');
    el.textContent = msg;
    el.classList.toggle('err', !!isErr);
    el.classList.add('show');
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => el.classList.remove('show'), 3200);
  }

  function confirmDialog(text) {
    return new Promise((resolve) => {
      const modal = $('#modal');
      $('#modal-text').textContent = text;
      modal.hidden = false;
      const done = (v) => {
        modal.hidden = true;
        $('#modal-ok').onclick = null;
        $('#modal-cancel').onclick = null;
        resolve(v);
      };
      $('#modal-ok').onclick = () => done(true);
      $('#modal-cancel').onclick = () => done(false);
      $('#modal-cancel').focus();
    });
  }

  async function run(fn) {
    if (state.busy) return undefined;
    state.busy = true;
    render();
    try {
      return await fn();
    } finally {
      state.busy = false;
      await refresh();
    }
  }

  // rc 75 = ctl.sh đang bận (khóa); các mã khác khác 0 = lỗi, kèm dòng lỗi đầu tiên để dễ chẩn đoán
  function reportResult(r, okMsg) {
    if (r.errno === 0) { toast(okMsg); return true; }
    if (r.errno === 75) toast(t('toast.busy'), true);
    else if (r.errno === -1) toast(t('toast.noBridge'), true);
    else {
      const detail = (r.stderr || r.stdout || '').trim().split('\n')[0].slice(0, 120);
      toast(detail ? t('toast.failedDetail', { e: `${detail} (${r.errno})` }) : t('toast.failed'), true);
    }
    return false;
  }

  // ------------------------------------------------------------ hiển thị
  function banner(kind, text) {
    const d = document.createElement('div');
    d.className = `banner ${kind}`;
    d.textContent = text;
    return d;
  }

  function render() {
    const st = state.status;
    const has = !!bridge();
    const running = !!st && st.running === '1';

    const dot = $('#status-dot');
    dot.className = 'dot';
    let text = t('status.unknown');
    if (state.busy) { dot.classList.add('busy'); text = t('status.working'); }
    else if (st) { dot.classList.add(running ? 'on' : 'off'); text = t(running ? 'status.running' : 'status.stopped'); }
    $('#status-text').textContent = text;

    $('#fact-pid').textContent = running && st.pid ? st.pid : '–';
    $('#fact-version').textContent = state.version || '–';
    $('#fact-port').textContent = st ? st.cfg_redir_port || '–' : '–';
    $('#fact-panel').textContent = st ? st.panel || '–' : '–';
    let redirect = '–';
    if (st) {
      if (st.cfg_enable_iptables !== 'true') redirect = t('fact.off');
      else redirect = t(st.iptables_active === '1' ? 'fact.active' : 'fact.inactive');
    }
    $('#fact-redirect').textContent = redirect;

    $('#btn-start').disabled = !has || state.busy || running;
    $('#btn-stop').disabled = !has || state.busy || !running;
    $('#btn-restart').disabled = !has || state.busy || !running;
    $('#btn-panel').disabled = !has || state.busy || !running;
    ['#btn-dns-save', '#btn-filters-save', '#btn-set-save', '#btn-acc-set', '#btn-debug', '#btn-reset', '#btn-log-refresh']
      .forEach((id) => { $(id).disabled = !has || state.busy; });

    const banners = $('#banners');
    banners.textContent = '';
    if (!has) banners.append(banner('err', t('banner.noBridge')));
    if (st && st.binary === '0') banners.append(banner('err', t('banner.noBinary')));
    if (st && st.password_locked === '1') banners.append(banner('warn', t('banner.locked')));
    if (st && running && st.cfg_enable_iptables === 'true' && st.iptables_active !== '1') {
      banners.append(banner('warn', t('banner.noRedirect')));
    }

    const acc = $('#acc-state');
    if (st) {
      const locked = st.password_locked === '1';
      acc.textContent = t(locked ? 'acc.locked' : 'acc.ready');
      acc.className = `pill ${locked ? 'bad' : 'ok'}`;
    }

    renderPresets();
    updateFilterCount();
  }

  function sameList(a, b) {
    if (a.length !== b.length) return false;
    const sb = new Set(b);
    return a.every((x) => sb.has(x));
  }

  function currentLines() {
    return $('#upstream-list').value.split('\n').map((s) => s.trim()).filter(Boolean);
  }

  function renderPresets() {
    const box = $('#presets');
    const lines = currentLines();
    box.textContent = '';
    let matched = false;
    Object.keys(PRESETS).forEach((key) => {
      const b = document.createElement('button');
      b.className = 'chip';
      b.textContent = t(`dns.preset.${key}`);
      if (sameList(lines, PRESETS[key])) { b.classList.add('active'); matched = true; }
      b.addEventListener('click', () => {
        $('#upstream-list').value = PRESETS[key].join('\n');
        renderPresets();
      });
      box.append(b);
    });
    const c = document.createElement('button');
    c.className = 'chip';
    c.textContent = t('dns.custom');
    if (!matched) c.classList.add('active');
    c.addEventListener('click', () => $('#upstream-list').focus());
    box.append(c);
  }

  function fillSettingsForm() {
    const st = state.status;
    if (!st) return;
    $('#set-iptables').checked = st.cfg_enable_iptables === 'true';
    $('#set-blockv6').checked = st.cfg_block_ipv6_dns === 'true';
    $('#set-watchdog').checked = st.cfg_watchdog === 'true';
    $('#set-port').value = st.cfg_redir_port || '';
    $('#set-ignore-dest').value = st.cfg_ignore_dest_list || '';
    $('#set-ignore-src').value = st.cfg_ignore_src_list || '';
    state.formLoaded = true;
  }

  // ------------------------------------------------------------ dữ liệu
  async function refresh() {
    if (!bridge()) { render(); return; }
    const r = await ctl(['status']);
    if (r.errno === 0) {
      state.status = parseKV(r.stdout);
      if (!state.formLoaded) {
        const l = state.status.cfg_language;
        if ((l === 'vi' || l === 'en') && l !== state.lang) setLang(l, false);
        fillSettingsForm();
      }
      if (!state.version) {
        const v = await ctl(['version']);
        if (v.errno === 0) state.version = v.stdout.trim();
      }
    }
    render();
  }

  async function loadUpstream() {
    const r = await ctl(['get-upstream']);
    if (r.errno !== 0) return;
    state.upstream = r.stdout.split('\n').map((s) => s.trim()).filter(Boolean);
    state.upstreamLoaded = true;
    $('#upstream-list').value = state.upstream.join('\n');
    renderPresets();
  }

  async function loadLog() {
    const view = $('#log-view');
    let text = '';
    if (state.logKind === 'debug') {
      text = state.debugText;
    } else {
      const r = await ctl(['logs', state.logKind, '200']);
      text = r.stdout;
    }
    view.textContent = text.trim() ? text : t('log.empty');
    view.scrollTop = view.scrollHeight;
  }

  function hostOf(url) {
    const m = /^[a-z]+:\/\/([^/]+)/i.exec(url);
    return m ? m[1] : url;
  }

  function renderFilters() {
    const box = $('#filters-list');
    box.textContent = '';
    if (!state.filters.length) {
      const p = document.createElement('p');
      p.className = 'muted';
      p.textContent = t('filters.none');
      box.append(p);
    }
    state.filters.forEach((f) => {
      const row = document.createElement('label');
      row.className = 'switch-row';
      const span = document.createElement('span');
      const strong = document.createElement('strong');
      strong.textContent = f.name;
      const small = document.createElement('small');
      small.textContent = hostOf(f.url);
      span.append(strong, small);
      const input = document.createElement('input');
      input.type = 'checkbox';
      input.className = 'switch';
      input.checked = f.enabled;
      input.dataset.id = f.id;
      input.addEventListener('change', () => { f.enabled = input.checked; updateFilterCount(); });
      row.append(span, input);
      box.append(row);
    });
    updateFilterCount();
  }

  function updateFilterCount() {
    const n = state.filters.filter((f) => f.enabled).length;
    $('#filters-count').textContent = t('filters.count', { n });
  }

  async function loadFilters() {
    const r = await ctl(['get-filters']);
    if (r.errno !== 0) return;
    state.filters = r.stdout.split('\n').map((l) => l.trim()).filter(Boolean).map((l) => {
      const p = l.split('|');
      return { id: p[0], enabled: p[1] === 'true', name: p.slice(2, -1).join('|'), url: p[p.length - 1] };
    });
    state.filtersOrig = state.filters.map((f) => f.enabled);
    state.filtersLoaded = true;
    renderFilters();
  }

  async function saveFilters() {
    const specs = [];
    state.filters.forEach((f, i) => {
      if (f.enabled !== state.filtersOrig[i]) specs.push(`${f.id}:${f.enabled}`);
    });
    if (!specs.length) { toast(t('filters.saved')); return; }
    await run(async () => {
      const r = await ctl(['set-filters', ...specs]);
      if (reportResult(r, t('filters.saved'))) await loadFilters();
    });
  }

  // ------------------------------------------------------------ hành động
  const actions = {
    start: () => run(async () => reportResult(await ctl(['start']), t('toast.started'))),
    stop: () => run(async () => reportResult(await ctl(['stop']), t('toast.stopped'))),
    restart: () => run(async () => reportResult(await ctl(['restart']), t('toast.restarted'))),
  };

  async function openPanel() {
    let addr = (state.status && state.status.panel) || '127.0.0.1:3000';
    addr = addr.replace(/^0\.0\.0\.0/, '127.0.0.1');
    if (!/^[0-9A-Za-z.:[\]-]+$/.test(addr)) addr = '127.0.0.1:3000';
    const url = `http://${addr}`;
    const r = await exec(`am start -a android.intent.action.VIEW -d ${q(url)} --user 0`);
    if (r.errno !== 0) {
      try { window.open(url, '_blank'); } catch (e) { /* bỏ qua */ }
      toast(t('toast.panelFailed', { url }), true);
    }
  }

  async function saveUpstream() {
    const lines = currentLines();
    if (!lines.length) { toast(t('dns.empty'), true); return; }
    const bad = lines.find((l) => !UPSTREAM_RE.test(l));
    if (bad) { toast(t('dns.invalid', { v: bad }), true); return; }
    await run(async () => {
      const r = await ctl(['set-upstream', ...lines]);
      if (reportResult(r, t('dns.saved'))) await loadUpstream();
    });
  }

  async function saveSettings() {
    const st = state.status || {};
    const want = {
      enable_iptables: $('#set-iptables').checked ? 'true' : 'false',
      block_ipv6_dns: $('#set-blockv6').checked ? 'true' : 'false',
      watchdog: $('#set-watchdog').checked ? 'true' : 'false',
      ignore_dest_list: $('#set-ignore-dest').value.trim().replace(/\s+/g, ' '),
      ignore_src_list: $('#set-ignore-src').value.trim().replace(/\s+/g, ' '),
    };
    const port = $('#set-port').value.trim();
    if (!/^\d+$/.test(port) || +port < 1 || +port > 65535) { toast(t('set.invalidPort'), true); return; }
    if (!ADDR_LIST_RE.test(want.ignore_dest_list) || !ADDR_LIST_RE.test(want.ignore_src_list)) {
      toast(t('set.invalidList'), true);
      return;
    }
    await run(async () => {
      let ok = true;
      for (const key of Object.keys(want)) {
        if ((st[`cfg_${key}`] || '') === want[key]) continue;
        const r = await ctl(['set-setting', key, want[key]]);
        if (r.errno !== 0) { ok = false; reportResult(r, ''); }
      }
      const running = st.running === '1';
      if (ok && port !== st.cfg_redir_port) {
        // set-port tự khởi động lại AdGuardHome (nếu đang chạy) sau khi cài đặt khác đã được ghi
        const r = await ctl(['set-port', port]);
        ok = reportResult(r, t('set.saved'));
      } else if (ok && running) {
        const r = await ctl(['restart']);
        ok = reportResult(r, t('set.saved'));
      } else if (ok) {
        toast(t('set.saved'));
      }
      state.formLoaded = false;
    });
  }

  async function setPassword() {
    const p1 = $('#acc-pass').value;
    const p2 = $('#acc-pass2').value;
    if (p1.length < 8) { toast(t('acc.short'), true); return; }
    if (new TextEncoder().encode(p1).length > 72) { toast(t('acc.long'), true); return; }
    if (p1 !== p2) { toast(t('acc.mismatch'), true); return; }
    const lib = (window.dcodeIO && window.dcodeIO.bcrypt) || window.bcrypt;
    if (!lib) { toast(t('toast.failed'), true); return; }
    await run(async () => {
      const hash = await new Promise((resolve, reject) => {
        lib.hash(p1, 10, (err, h) => (err ? reject(err) : resolve(h)));
      }).catch(() => '');
      if (!hash) { toast(t('toast.failed'), true); return; }
      const r = await ctl(['set-password', hash]);
      if (reportResult(r, t('acc.saved'))) {
        $('#acc-pass').value = '';
        $('#acc-pass2').value = '';
      }
    });
  }

  async function collectDebug() {
    await run(async () => {
      const r = await ctl(['debug']);
      state.debugText = r.stdout;
      state.logKind = 'debug';
      toast(t('maint.debugDone'));
    });
    showTab('logs');
  }

  async function resetConfig() {
    if (!(await confirmDialog(t('maint.resetConfirm')))) return;
    await run(async () => {
      const r = await ctl(['reset-config']);
      if (reportResult(r, t('maint.resetDone'))) {
        state.upstreamLoaded = false;
        state.formLoaded = false;
        await loadUpstream();
        await loadFilters();
      }
    });
  }

  // ------------------------------------------------------------ tab
  function showTab(name) {
    state.tab = name;
    $$('.tabs button').forEach((b) => b.classList.toggle('active', b.dataset.tab === name));
    $$('.tab').forEach((s) => s.classList.toggle('active', s.id === `tab-${name}`));
    if (name === 'dns' && !state.upstreamLoaded) loadUpstream();
    if (name === 'dns' && !state.filtersLoaded) loadFilters();
    if (name === 'logs') {
      $$('#log-chips .chip').forEach((c) => c.classList.toggle('active', c.dataset.log === state.logKind));
      loadLog();
    }
  }

  // ------------------------------------------------------------ khởi tạo
  function init() {
    const saved = store.get('lang');
    state.lang = saved === 'en' ? 'en' : 'vi';
    applyI18n();

    $$('.tabs button').forEach((b) => b.addEventListener('click', () => showTab(b.dataset.tab)));
    $$('#lang-chips .chip').forEach((b) => b.addEventListener('click', () => setLang(b.dataset.lang, true)));
    $$('#log-chips .chip').forEach((b) => b.addEventListener('click', () => {
      state.logKind = b.dataset.log;
      showTab('logs');
    }));
    $('#btn-start').addEventListener('click', actions.start);
    $('#btn-stop').addEventListener('click', actions.stop);
    $('#btn-restart').addEventListener('click', actions.restart);
    $('#btn-panel').addEventListener('click', openPanel);
    $('#btn-dns-save').addEventListener('click', saveUpstream);
    $('#btn-filters-save').addEventListener('click', saveFilters);
    $('#btn-set-save').addEventListener('click', saveSettings);
    $('#btn-acc-set').addEventListener('click', setPassword);
    $('#btn-debug').addEventListener('click', collectDebug);
    $('#btn-reset').addEventListener('click', resetConfig);
    $('#btn-log-refresh').addEventListener('click', loadLog);
    $('#upstream-list').addEventListener('input', renderPresets);

    render();
    refresh();
    setInterval(() => {
      if (document.visibilityState === 'visible' && !state.busy) refresh();
    }, 5000);
  }

  init();
})();
