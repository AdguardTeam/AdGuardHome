/* WebUI của module: nói chuyện với ctl.sh qua cầu nối exec của KernelSU / APatch. */
(() => {
  'use strict';

  const GEO = window.AGH_GEO;
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

  // Tên hiển thị gọn của các mức chặn chính (tên trong cấu hình của bản cũ có thể còn hậu tố).
  const CANON = { 1: 'AdGuard DNS filter', 101: 'uBlock filters', 48: 'HaGeZi Pro', 51: 'HaGeZi Pro++', 49: 'HaGeZi Ultimate' };
  // Khi cấu hình cũ có nhiều mức chặn cùng bật, hiển thị mức mạnh nhất (khớp normalize_selection_inner trong ctl.sh).
  const MAIN_PRIORITY = ['49', '51', '48', '1', '101'];
  const CACHE_KEY = 'cache2';
  const OPT_PHASE = { start: 'starting_op', stop: 'stopping_op', restart: 'restarting_op', apply: 'working' };

  const $ = (sel, root = document) => root.querySelector(sel);
  const $$ = (sel, root = document) => Array.from(root.querySelectorAll(sel));
  const sleep = (ms) => new Promise((r) => { setTimeout(r, ms); });
  const el = (tag, cls, text) => {
    const e = document.createElement(tag);
    if (cls) e.className = cls;
    if (text !== undefined) e.textContent = text;
    return e;
  };
  const setText = (node, v) => { if (node.textContent !== v) node.textContent = v; };

  const state = {
    lang: 'vi',
    status: null,
    fresh: false, // false = đang hiển thị dữ liệu từ bộ nhớ đệm
    busy: null, // null | start | stop | restart | apply
    busyBtn: '',
    phase: '',
    phaseSince: 0,
    upstream: [],
    upstreamLoaded: false,
    upstreamDirty: false,
    filters: [],
    filtersLoaded: false,
    sel: { main: '', country: '0' }, // lựa chọn đang hiển thị
    selOrig: { main: '', country: '0' }, // lựa chọn đã áp dụng
    open: {}, // nhóm bộ lọc nào đang mở
    device: null,
    country: null, // null = đang nhận diện | { cc, group, source }
    showAllCountries: false,
    formLoaded: false,
    logKind: 'core',
    debugText: '',
    tab: 'overview',
  };

  // ------------------------------------------------------------ lưu trữ cục bộ
  const store = {
    get(k) { try { return localStorage.getItem(k); } catch (e) { return null; } },
    set(k, v) { try { localStorage.setItem(k, v); } catch (e) { /* bỏ qua */ } },
    del(k) { try { localStorage.removeItem(k); } catch (e) { /* bỏ qua */ } },
  };

  // ------------------------------------------------------------ i18n
  const hasKey = (key) => !!(window.I18N && window.I18N[state.lang] && window.I18N[state.lang][key]);
  function t(key, vars) {
    const dict = window.I18N || {};
    let s = (dict[state.lang] && dict[state.lang][key]) || (dict.vi && dict.vi[key]) || key;
    if (vars) for (const k of Object.keys(vars)) s = s.split(`{${k}}`).join(vars[k]);
    return s;
  }

  function applyI18n() {
    document.documentElement.lang = state.lang;
    $$('[data-i18n]').forEach((n) => setText(n, t(n.dataset.i18n)));
    $$('[data-i18n-placeholder]').forEach((n) => { n.placeholder = t(n.dataset.i18nPlaceholder); });
    $$('[data-lang]').forEach((n) => n.classList.toggle('active', n.dataset.lang === state.lang));
  }

  function setLang(lang, persist) {
    if (!window.I18N || !window.I18N[lang]) return;
    state.lang = lang;
    store.set('lang', lang);
    applyI18n();
    lastBannerKey = '';
    renderAll();
    renderFilters();
    if (persist && bridge()) ctl(['set-setting', 'language', lang]);
  }

  // ------------------------------------------------------------ cầu nối root
  function bridge() {
    if (typeof ksu !== 'undefined' && ksu && typeof ksu.exec === 'function') return ksu;
    if (typeof ap !== 'undefined' && ap && typeof ap.exec === 'function') return ap;
    return null;
  }

  let execSeq = 0;
  // exec(lệnh, thời gian chờ tối đa ms): lệnh đọc dữ liệu có giới hạn thời gian để giao diện không bao giờ bị treo.
  function exec(cmd, timeoutMs) {
    return new Promise((resolve) => {
      const b = bridge();
      if (!b) { resolve({ errno: -1, stdout: '', stderr: 'no bridge' }); return; }
      const cb = `agh_exec_${Date.now()}_${execSeq++}`;
      let done = false;
      let timer = 0;
      const finish = (r) => { if (done) return; done = true; clearTimeout(timer); resolve(r); };
      window[cb] = (errno, stdout, stderr) => {
        delete window[cb];
        finish({ errno: Number(errno), stdout: String(stdout || ''), stderr: String(stderr || '') });
      };
      if (timeoutMs) timer = setTimeout(() => finish({ errno: -2, stdout: '', stderr: 'timeout' }), timeoutMs);
      try {
        b.exec(cmd, JSON.stringify({}), cb);
      } catch (e) {
        delete window[cb];
        finish({ errno: -1, stdout: '', stderr: String(e) });
      }
    });
  }

  const q = (s) => `'${String(s).replace(/'/g, `'\\''`)}'`;
  // Thao tác ghi có thể chờ khóa (45 s) rồi khởi động lại AdGuardHome (30 s); quá 150 s coi như cầu nối root bị treo.
  const LONG = 150000;
  const ctl = (args, timeoutMs = LONG) => exec(`${CTL} ${args.map(q).join(' ')}`, timeoutMs);

  function parseKV(text) {
    const out = {};
    String(text).split('\n').forEach((line) => {
      const m = /^([a-z_0-9]+)=(.*)$/.exec(line.replace(/\r$/, ''));
      if (m) out[m[1]] = m[2];
    });
    return out;
  }

  function parseFilterLine(line) {
    const p = line.split('|');
    return { id: p[0], enabled: p[1] === 'true', name: p.slice(2, -2).join('|'), url: p[p.length - 2], group: p[p.length - 1] };
  }

  // ctl.sh snapshot: các phần cách nhau bằng dòng "##tên"
  function parseSnapshot(text) {
    const sec = {};
    let cur = null;
    String(text).split('\n').forEach((raw) => {
      const l = raw.replace(/\r$/, '');
      if (l.startsWith('##')) { cur = l.slice(2); sec[cur] = []; } else if (cur) sec[cur].push(l);
    });
    const lines = (k) => (sec[k] || []).map((s) => s.trim()).filter(Boolean);
    return {
      status: parseKV((sec.status || []).join('\n')),
      upstream: lines('upstream'),
      filters: lines('filters').filter((l) => l.split('|').length >= 5).map(parseFilterLine),
      device: parseKV((sec.device || []).join('\n')),
    };
  }

  // ------------------------------------------------------------ bộ nhớ đệm: mở là hiện ngay, cập nhật ngầm sau
  function saveCache() {
    store.set(CACHE_KEY, JSON.stringify({
      ts: Date.now(),
      status: state.status,
      upstream: state.upstream,
      filters: state.filters.map((f) => ({ id: f.id, enabled: f.enabled, name: f.name, url: f.url, group: f.group })),
      device: state.device,
    }));
  }

  function hydrateFromCache() {
    try {
      const c = JSON.parse(store.get(CACHE_KEY) || 'null');
      if (!c || !c.status) return;
      state.status = c.status;
      if (Array.isArray(c.upstream) && c.upstream.length) { state.upstream = c.upstream; state.upstreamLoaded = true; $('#upstream-list').value = c.upstream.join('\n'); }
      if (Array.isArray(c.filters) && c.filters.length) { applyFilters(c.filters, true); }
      state.device = c.device || null;
    } catch (e) { /* cache hỏng: bỏ qua */ }
  }

  // ------------------------------------------------------------ giao diện chung
  let toastTimer = 0;
  function hideToast() { $('#toast').classList.remove('show'); }
  // kind: ok | warn | err.  opts.log: thêm nút "Xem nhật ký"
  function toast(msg, kind, opts) {
    const k = kind || 'ok';
    const node = $('#toast');
    setText($('#toast-text'), msg);
    node.classList.remove('ok', 'warn', 'err');
    node.classList.add(k);
    const act = $('#toast-action');
    if (opts && opts.log) {
      act.hidden = false;
      act.textContent = t('toast.viewLog');
      act.onclick = () => { hideToast(); state.logKind = 'history'; showTab('logs'); };
    } else {
      act.hidden = true;
      act.onclick = null;
    }
    node.classList.add('show');
    clearTimeout(toastTimer);
    toastTimer = setTimeout(hideToast, k === 'ok' ? 3000 : 6500);
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

  // Báo kết quả của một lệnh ctl.sh bằng thông báo cụ thể theo mã "result=" (xem ctl.sh).
  // Mã thoát: 0 ok, 3 = đã áp dụng nhưng AdGuardHome chưa bật chuyển hướng DNS, 75 = bận, khác = lỗi.
  // Trả về true nếu thay đổi đã có hiệu lực (kể cả khi kèm cảnh báo).
  function report(r, okMsg) {
    const kv = parseKV(r.stdout);
    const code = kv.result || '';
    const key = `result.${code}`;
    const known = code && hasKey(key);
    const msg = () => t(key, { detail: kv.detail || '' });
    if (r.errno === 0) { toast(okMsg || (known ? msg() : t('result.applied')), 'ok'); return true; }
    if (r.errno === 75 || code === 'busy') { toast(t('result.busy'), 'warn'); return false; }
    if (r.errno === -1) { toast(t('toast.noBridge'), 'err'); return false; }
    if (r.errno === 3 && known) { toast(msg(), 'warn', { log: true }); return true; } // đã áp dụng nhưng kèm cảnh báo
    if (known) { toast(msg(), 'err', { log: code !== 'invalid' }); return false; }
    const detail = (r.stderr || r.stdout || '').trim().split('\n')[0].slice(0, 120);
    toast(detail ? t('toast.failedDetail', { e: `${detail} (${r.errno})` }) : t('toast.failedDetail', { e: String(r.errno) }), 'err', { log: true });
    return false;
  }

  function setProgress(on) { $('#progress').classList.toggle('on', !!on); }

  // ------------------------------------------------------------ hiển thị
  const fmtDuration = (s) => {
    if (s < 60) return t('time.s', { n: s });
    if (s < 3600) return t('time.m', { n: Math.floor(s / 60) });
    return t('time.h', { h: Math.floor(s / 3600), m: Math.floor((s % 3600) / 60) });
  };

  function phaseLabel() {
    const key = `phase.${state.phase}`;
    return state.phase && hasKey(key) ? t(key) : t('status.working');
  }

  function phaseSub() {
    if (state.phase !== 'waiting_dns' || !state.phaseSince) return '';
    return t('status.waited', { d: fmtDuration(Math.max(0, Math.floor(Date.now() / 1000 - state.phaseSince))) });
  }

  function filtersFact(st, running) {
    if (!running) return { text: '–', warn: false };
    const n = Number(st.filters_enabled || 0);
    const p = Number(st.filters_pending || 0);
    const up = Number(st.uptime || 0);
    if (!n) return { text: t('fact.filters.none'), warn: false };
    if (p > 0 && up < 180) return { text: t('fact.filters.loading', { d: n - p, n }), warn: false };
    if (p > 0) return { text: t('fact.filters.stuck', { p, n }), warn: true };
    return { text: `${t('fact.filters.ready')} (${n})`, warn: false };
  }

  function renderStatus() {
    const st = state.status;
    const running = !!st && st.running === '1';
    let cls = 'dot';
    let text;
    let sub = '';
    if (state.busy) { cls += ' busy'; text = phaseLabel(); sub = phaseSub(); } else if (!st) { text = t('status.unknown'); } else {
      cls += running ? ' on' : ' off';
      text = t(running ? 'status.running' : 'status.stopped');
      if (running && Number(st.uptime) > 0) sub = t('status.uptime', { d: fmtDuration(Number(st.uptime)) });
    }
    const dot = $('#status-dot');
    if (dot.className !== cls) dot.className = cls;
    setText($('#status-text'), text);
    setText($('#status-sub'), sub);
    $('#status-card').setAttribute('aria-busy', state.busy ? 'true' : 'false');
    document.body.classList.toggle('stale', !state.fresh);

    const fact = (id, value, warn) => {
      const n = $(id);
      setText(n, value);
      n.classList.remove('skel');
      n.classList.toggle('warn', !!warn);
    };
    if (st) {
      fact('#fact-pid', running && st.pid ? st.pid : '–');
      fact('#fact-version', st.version || '–');
      fact('#fact-port', st.cfg_redir_port || '–');
      fact('#fact-panel', st.panel || '–');
      let redirect;
      if (st.cfg_enable_iptables !== 'true') redirect = t('fact.off');
      else redirect = t(st.iptables_active === '1' ? 'fact.active' : 'fact.inactive');
      fact('#fact-redirect', redirect);
      const ff = filtersFact(st, running);
      fact('#fact-filters', ff.text, ff.warn);
    }
  }

  let lastBannerKey = '';
  function renderBanners() {
    const st = state.status;
    const running = !!st && st.running === '1';
    const items = [];
    if (!bridge()) items.push(['err', 'banner.noBridge']);
    if (st && st.binary === '0') items.push(['err', 'banner.noBinary']);
    if (st && st.password_locked === '1') items.push(['warn', 'banner.locked']);
    if (st && running && !state.busy && st.cfg_enable_iptables === 'true' && st.iptables_active !== '1') items.push(['warn', 'banner.noRedirect']);
    const key = `${state.lang}|${items.map((i) => i[1]).join(',')}`;
    if (key === lastBannerKey) return;
    lastBannerKey = key;
    const box = $('#banners');
    box.textContent = '';
    items.forEach(([kind, k]) => box.append(el('div', `banner ${kind}`, t(k))));
  }

  function renderButtons() {
    const has = !!bridge();
    const busy = !!state.busy;
    const running = !!state.status && state.status.running === '1';
    const dis = (id, v) => { const b = $(id); if (b.disabled !== v) b.disabled = v; };
    dis('#btn-start', !has || busy || running);
    dis('#btn-stop', !has || busy || !running);
    dis('#btn-restart', !has || busy || !running);
    dis('#btn-panel', !has || busy || !running);
    ['#btn-dns-save', '#btn-set-save', '#btn-acc-set', '#btn-debug', '#btn-reset', '#btn-log-refresh']
      .forEach((id) => dis(id, !has || busy));
    $$('.btn').forEach((b) => b.classList.toggle('loading', busy && !!state.busyBtn && b.id === state.busyBtn));
    updateFilterMeta();
  }

  function renderAccount() {
    const st = state.status;
    const acc = $('#acc-state');
    if (!st) return;
    const locked = st.password_locked === '1';
    setText(acc, t(locked ? 'acc.locked' : 'acc.ready'));
    const cls = `pill ${locked ? 'bad' : 'ok'}`;
    if (acc.className !== cls) acc.className = cls;
  }

  function renderAll() {
    renderStatus();
    renderBanners();
    renderButtons();
    renderAccount();
    renderPresets();
  }

  const sameList = (a, b) => a.length === b.length && a.every((x) => b.includes(x));
  const currentLines = () => $('#upstream-list').value.split('\n').map((s) => s.trim()).filter(Boolean);

  function renderPresets() {
    const box = $('#presets');
    const lines = currentLines();
    const key = `${state.lang}|${lines.join(',')}`;
    if (box.dataset.key === key) return;
    box.dataset.key = key;
    box.textContent = '';
    let matched = false;
    Object.keys(PRESETS).forEach((k) => {
      const b = el('button', 'chip', t(`dns.preset.${k}`));
      if (sameList(lines, PRESETS[k])) { b.classList.add('active'); matched = true; }
      b.addEventListener('click', () => { $('#upstream-list').value = PRESETS[k].join('\n'); renderPresets(); });
      box.append(b);
    });
    const c = el('button', 'chip', t('dns.custom'));
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
  function applyStatus(st) {
    state.status = st;
    state.fresh = true;
    if (!state.formLoaded) {
      const l = st.cfg_language;
      if ((l === 'vi' || l === 'en') && l !== state.lang) setLang(l, false);
      fillSettingsForm();
    }
  }

  // Một lần gọi lấy mọi thứ cần hiển thị (mỗi lần exec qua trình quản lý root đều tốn thời gian).
  // force: bỏ lựa chọn bộ lọc chưa lưu; forceUpstream: bỏ cả danh sách DNS đang gõ dở.
  async function loadSnapshot(force, forceUpstream) {
    const r = await ctl(['snapshot'], 25000);
    if (r.errno !== 0) return false;
    const snap = parseSnapshot(r.stdout);
    applyStatus(snap.status);
    state.upstream = snap.upstream;
    state.upstreamLoaded = true;
    if (forceUpstream || !state.upstreamDirty) {
      $('#upstream-list').value = snap.upstream.join('\n');
      state.upstreamDirty = false;
    }
    applyFilters(snap.filters, !!force);
    state.device = snap.device;
    saveCache();
    renderAll();
    resolveCountry().then(renderFilters);
    return true;
  }

  // Chỉ lấy trạng thái: nhẹ, dùng cho vòng cập nhật định kỳ
  async function refresh() {
    if (!bridge()) { renderAll(); return false; }
    const r = await ctl(['status'], 15000);
    if (r.errno !== 0) return false;
    applyStatus(parseKV(r.stdout));
    saveCache();
    renderAll();
    return true;
  }

  async function loadLog(keepScroll) {
    const view = $('#log-view');
    let text = '';
    if (state.logKind === 'debug') {
      text = state.debugText;
    } else {
      const r = await ctl(['logs', state.logKind, '200'], 8000);
      text = r.stdout;
    }
    const atBottom = view.scrollHeight - view.scrollTop - view.clientHeight < 40;
    const next = text.trim() ? text : t('log.empty');
    if (view.textContent !== next) {
      view.textContent = next;
      if (!keepScroll || atBottom) view.scrollTop = view.scrollHeight;
    }
  }

  // ------------------------------------------------------------ nhận diện quốc gia
  const tzName = () => { try { return Intl.DateTimeFormat().resolvedOptions().timeZone || ''; } catch (e) { return ''; } };
  const regionName = (cc) => {
    try { return new Intl.DisplayNames([state.lang], { type: 'region' }).of(cc) || cc; } catch (e) { return cc; }
  };

  // Tra IP chỉ khi cần; kết quả nhớ 6 giờ để không gọi dịch vụ lặp lại.
  async function ipLookupCached() {
    try {
      const c = JSON.parse(store.get('ip_country') || 'null');
      if (c && c.cc && Date.now() - c.ts < 6 * 3600 * 1000) return { cc: c.cc };
    } catch (e) { /* bỏ qua */ }
    const r = await GEO.lookupIp();
    if (r && r.cc) store.set('ip_country', JSON.stringify({ cc: r.cc, ts: Date.now() }));
    return r;
  }

  let resolving = null;
  function resolveCountry() {
    if (resolving) return resolving;
    const run = (async () => {
      try {
        const manual = store.get('country_manual');
        if (manual && GEO.GROUPS.includes(manual)) {
          state.country = { cc: manual, group: manual, source: 'manual' };
          return;
        }
        const dev = state.device || {};
        const res = await GEO.decide({ net: dev.net, sim: dev.sim, region: dev.region, tz: tzName() }, ipLookupCached);
        // Người dùng vừa chọn thủ công trong lúc đang tra: giữ lựa chọn của họ
        if (!store.get('country_manual')) state.country = { cc: res.cc, group: GEO.groupOf(res.cc), source: res.source };
      } catch (e) {
        state.country = { cc: '', group: '', source: '' };
      }
    })();
    // Đặt lại sau khi xong (không đặt trong finally của chính hàm async: nhánh chạy đồng bộ sẽ xóa trước khi gán)
    resolving = run;
    run.then(() => { if (resolving === run) resolving = null; });
    return run;
  }

  // ------------------------------------------------------------ bộ lọc: mỗi nhóm chỉ chọn một
  const byId = (id) => state.filters.find((f) => f.id === id);
  const hostOf = (url) => { const m = /^[a-z]+:\/\/([^/]+)/i.exec(url); return m ? m[1] : url; };
  const isCountryList = (f) => f.group !== 'main' && f.group !== 'other';

  function displayName(f) {
    if (CANON[f.id]) return CANON[f.id];
    return f.name.replace(/^[A-Z]{3} - /, '').replace(/\s*\(.*\)\s*$/, '');
  }

  const noteOf = (f) => { const k = `filters.note.${f.id}`; return hasKey(k) ? t(k) : ''; };

  function deriveSelection() {
    const on = (f) => f.enabled;
    let main = '';
    for (const id of MAIN_PRIORITY) { const f = byId(id); if (f && f.group === 'main' && on(f)) { main = id; break; } }
    if (!main) { const f = state.filters.find((x) => x.group === 'main' && on(x)); if (f) main = f.id; }
    const cf = state.filters.find((f) => isCountryList(f) && on(f));
    state.selOrig = { main, country: cf ? cf.id : '0' };
    state.sel = { ...state.selOrig };
    state.filters.forEach((f) => { f.orig = f.enabled; });
  }

  function filterDirty() {
    return state.sel.main !== state.selOrig.main || state.sel.country !== state.selOrig.country
      || state.filters.some((f) => f.group === 'other' && f.enabled !== f.orig);
  }

  // Không ghi đè lựa chọn chưa lưu của người dùng, trừ khi force
  function applyFilters(list, force) {
    if (state.filtersLoaded && filterDirty() && !force) return;
    state.filters = list.map((f) => ({ ...f }));
    state.filtersLoaded = state.filters.length > 0;
    deriveSelection();
    renderFilters();
  }

  function updateFilterMeta() {
    const names = [];
    const m = byId(state.selOrig.main);
    if (m) names.push(displayName(m));
    const c = byId(state.selOrig.country);
    if (c) names.push(displayName(c));
    state.filters.filter((f) => f.group === 'other' && f.orig).forEach((f) => names.push(displayName(f)));
    const line = state.filtersLoaded
      ? (names.length ? t('filters.current', { names: names.join(' + ') }) : t('filters.current.none')) : '';
    setText($('#filters-current'), line);
    const dirty = filterDirty();
    $('#filters-unsaved').hidden = !dirty;
    const btn = $('#btn-filters-save');
    const off = !bridge() || !!state.busy || !dirty;
    if (btn.disabled !== off) btn.disabled = off;
  }

  function refreshSelectedRows() {
    $$('#filters-list .switch-row[data-kind]').forEach((row) => {
      const kind = row.dataset.kind;
      row.classList.toggle('selected', kind !== 'other' && state.sel[kind] === row.dataset.id);
    });
  }

  function filterRow(f, kind) {
    const row = el('label', 'switch-row');
    row.dataset.kind = kind;
    row.dataset.id = f.id;
    const span = el('span');
    span.append(el('strong', '', displayName(f)));
    const sub = noteOf(f) || (kind === 'other' ? hostOf(f.url) : '');
    if (sub) span.append(el('small', '', sub));
    const input = el('input');
    if (kind === 'other') {
      input.type = 'checkbox';
      input.className = 'switch';
      input.checked = f.enabled;
      input.addEventListener('change', () => { f.enabled = input.checked; updateFilterMeta(); });
    } else {
      input.type = 'radio';
      input.className = 'radio';
      input.name = kind === 'main' ? 'fmain' : 'fcountry';
      input.value = f.id;
      input.checked = state.sel[kind] === f.id;
      input.addEventListener('change', () => { state.sel[kind] = f.id; refreshSelectedRows(); updateFilterMeta(); });
    }
    row.append(span, input);
    return row;
  }

  function noneRow() {
    const row = el('label', 'switch-row none-row');
    row.dataset.kind = 'country';
    row.dataset.id = '0';
    const span = el('span');
    span.append(el('strong', '', t('filters.pick.none')));
    const input = el('input');
    input.type = 'radio';
    input.className = 'radio';
    input.name = 'fcountry';
    input.value = '0';
    input.checked = state.sel.country === '0';
    input.addEventListener('change', () => { state.sel.country = '0'; refreshSelectedRows(); updateFilterMeta(); });
    row.append(span, input);
    return row;
  }

  function filterGroup(key, title, hint, rows, openDefault) {
    const d = el('details', 'fgroup');
    d.dataset.group = key;
    d.open = key in state.open ? state.open[key] : openDefault;
    const sum = el('summary');
    sum.append(el('span', '', title));
    if (hint) sum.append(el('span', 'ghint', `(${hint})`));
    d.append(sum, ...rows);
    sum.addEventListener('click', () => { state.open[key] = !d.open; });
    return d;
  }

  // Tên không kèm cờ, dùng để sắp xếp theo thứ tự chữ cái; Việt Nam luôn đứng đầu.
  const plainCountry = (g) => t(`country.${g}`).replace(/^\S+\s+/, '');
  const byCountryName = (x, y) => (x === 'VN' ? -1 : y === 'VN' ? 1 : plainCountry(x).localeCompare(plainCountry(y), state.lang));

  const countryLabel = (cc, group) => {
    if (group && cc === group && hasKey(`country.${group}`)) return t(`country.${group}`);
    const name = regionName(cc);
    return group && hasKey(`country.${group}`) ? `${name} · ${t(`country.${group}`)}` : name;
  };

  function countryBar() {
    const bar = el('div', 'cbar');
    const c = state.country;
    const info = el('p', 'muted small');
    if (c === null) {
      info.textContent = t('filters.country.detecting');
    } else if (!c.cc) {
      info.textContent = t('filters.country.unknown');
    } else {
      const via = t('filters.country.via', { s: t(`geo.source.${c.source}`) });
      const label = countryLabel(c.cc, c.group);
      info.textContent = `${t('filters.country.detected', { c: label })} (${c.source === 'manual' ? t('geo.source.manual') : via})`;
      if (!c.group) info.textContent += ` ${t('filters.country.none', { c: regionName(c.cc) })}`;
    }
    const row = el('div', 'cbar-row');
    const select = el('select', 'chip');
    select.setAttribute('aria-label', t('filters.country.label'));
    select.id = 'country-select';
    const auto = el('option', '', t('filters.country.auto'));
    auto.value = 'auto';
    select.append(auto);
    GEO.GROUPS.slice().sort(byCountryName).forEach((g) => { const o = el('option', '', t(`country.${g}`)); o.value = g; select.append(o); });
    select.value = store.get('country_manual') && GEO.GROUPS.includes(store.get('country_manual')) ? store.get('country_manual') : 'auto';
    select.addEventListener('change', () => {
      if (select.value === 'auto') { store.del('country_manual'); state.country = null; renderFilters(); resolveCountry().then(renderFilters); } else {
        store.set('country_manual', select.value);
        state.country = { cc: select.value, group: select.value, source: 'manual' };
        renderFilters();
      }
    });
    const all = el('button', 'chip', t(state.showAllCountries ? 'filters.country.mine' : 'filters.country.all'));
    all.id = 'btn-all-countries';
    all.addEventListener('click', () => { state.showAllCountries = !state.showAllCountries; renderFilters(); });
    row.append(select, all);
    bar.append(info, row);
    return bar;
  }

  function renderFilters() {
    const box = $('#filters-list');
    const frag = document.createDocumentFragment();
    if (!state.filtersLoaded) {
      for (let i = 0; i < 3; i += 1) frag.append(el('span', 'skel skel-row'));
      box.textContent = '';
      box.append(frag);
      updateFilterMeta();
      return;
    }
    const query = ($('#filters-search').value || '').trim().toLowerCase();
    const match = (f) => {
      if (!query) return true;
      const g = isCountryList(f) ? `${t(`country.${f.group}`)} ${f.group}` : '';
      return `${displayName(f)} ${f.name} ${f.url} ${noteOf(f)} ${g}`.toLowerCase().includes(query);
    };
    const shown = state.filters.filter(match);
    if (!shown.length) frag.append(el('p', 'muted', t(state.filters.length ? 'filters.noMatch' : 'filters.none')));

    const mains = shown.filter((f) => f.group === 'main');
    if (mains.length) frag.append(filterGroup('main', t('filters.main.title'), t('filters.main.hint'), mains.map((f) => filterRow(f, 'main')), true));

    const byCountry = {};
    shown.filter(isCountryList).forEach((f) => { (byCountry[f.group] = byCountry[f.group] || []).push(f); });
    const countryKeys = Object.keys(byCountry);
    if (countryKeys.length || !query) {
      const h = el('p', 'fsection');
      h.append(el('span', '', t('filters.country.title')), el('span', 'ghint', `(${t('filters.country.hint')})`));
      frag.append(h);
    }
    if (!query) frag.append(countryBar());
    if (!query) frag.append(noneRow());
    const mine = state.country && state.country.group;
    const selected = byId(state.sel.country);
    const selectedGroup = selected && isCountryList(selected) ? selected.group : '';
    let visible;
    if (query || state.showAllCountries) visible = countryKeys;
    else visible = countryKeys.filter((g) => g === mine || g === selectedGroup);
    visible.sort(byCountryName);
    visible.forEach((g) => frag.append(filterGroup(g, t(`country.${g}`), '', byCountry[g].map((f) => filterRow(f, 'country')),
      !!query || g === mine || g === selectedGroup)));

    const others = shown.filter((f) => f.group === 'other');
    if (others.length) frag.append(filterGroup('other', t('filters.other.title'), '', others.map((f) => filterRow(f, 'other')), true));

    box.textContent = '';
    box.append(frag);
    refreshSelectedRows();
    updateFilterMeta();
  }

  async function saveFilters() {
    if (!filterDirty()) { toast(t('filters.unchanged'), 'ok'); return; }
    await operate('apply', 'btn-filters-save', async () => {
      let ok = true;
      if (state.sel.main !== state.selOrig.main || state.sel.country !== state.selOrig.country) {
        const r = await ctl(['set-filter-selection', state.sel.main || '0', state.sel.country || '0']);
        ok = report(r, t('filters.saved'));
      }
      const specs = state.filters.filter((f) => f.group === 'other' && f.enabled !== f.orig).map((f) => `${f.id}:${f.enabled}`);
      if (ok && specs.length) {
        const r = await ctl(['set-filters', ...specs]);
        ok = report(r, t('filters.saved'));
      }
      await loadSnapshot(true);
    });
  }

  // ------------------------------------------------------------ thao tác
  // Chạy một thao tác dài: khóa nút, hiện thanh tiến trình và giai đoạn thật mà module đang làm.
  function startPhasePoll() {
    let stop = false;
    (async () => {
      await sleep(250);
      while (!stop) {
        const r = await ctl(['phase'], 4000);
        if (stop) break;
        if (r.errno === 0) {
          const kv = parseKV(r.stdout);
          if (kv.phase && (kv.phase !== state.phase || kv.phase === 'waiting_dns')) {
            state.phase = kv.phase;
            state.phaseSince = Number(kv.phase_since) || state.phaseSince;
            renderStatus();
          }
        }
        await sleep(350);
      }
    })();
    return () => { stop = true; };
  }

  async function operate(kind, btnId, fn) {
    if (state.busy) return undefined;
    state.busy = kind;
    state.busyBtn = btnId || '';
    state.phase = OPT_PHASE[kind] || 'working';
    state.phaseSince = 0;
    renderAll();
    setProgress(true);
    const stopPoll = startPhasePoll();
    try {
      return await fn();
    } finally {
      stopPoll();
      state.busy = null;
      state.busyBtn = '';
      state.phase = '';
      setProgress(false);
      await refresh();
      renderAll();
      schedulePoll();
    }
  }

  const actions = {
    start: () => operate('start', 'btn-start', async () => report(await ctl(['start']))),
    stop: () => operate('stop', 'btn-stop', async () => report(await ctl(['stop']))),
    restart: () => operate('restart', 'btn-restart', async () => report(await ctl(['restart']))),
  };

  async function openPanel() {
    let addr = (state.status && state.status.panel) || '127.0.0.1:3000';
    addr = addr.replace(/^0\.0\.0\.0/, '127.0.0.1');
    if (!/^[0-9A-Za-z.:[\]-]+$/.test(addr)) addr = '127.0.0.1:3000';
    const url = `http://${addr}`;
    const r = await exec(`am start -a android.intent.action.VIEW -d ${q(url)} --user 0`, 8000);
    if (r.errno !== 0) {
      try { window.open(url, '_blank'); } catch (e) { /* bỏ qua */ }
      toast(t('toast.panelFailed', { url }), 'err');
    }
  }

  async function saveUpstream() {
    const lines = currentLines();
    if (!lines.length) { toast(t('dns.empty'), 'err'); return; }
    const bad = lines.find((l) => !UPSTREAM_RE.test(l));
    if (bad) { toast(t('dns.invalid', { v: bad }), 'err'); return; }
    if (state.upstream.length === lines.length && state.upstream.every((u, i) => u === lines[i])) { toast(t('dns.unchanged'), 'ok'); return; }
    await operate('apply', 'btn-dns-save', async () => {
      const r = await ctl(['set-upstream', ...lines]);
      if (report(r, t('dns.saved'))) { state.upstream = lines; state.upstreamDirty = false; }
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
    if (!/^\d+$/.test(port) || +port < 1 || +port > 65535) { toast(t('set.invalidPort'), 'err'); return; }
    if (!ADDR_LIST_RE.test(want.ignore_dest_list) || !ADDR_LIST_RE.test(want.ignore_src_list)) {
      toast(t('set.invalidList'), 'err');
      return;
    }
    const changed = Object.keys(want).filter((k) => (st[`cfg_${k}`] || '') !== want[k]);
    if (!changed.length && port === st.cfg_redir_port) { toast(t('set.unchanged'), 'ok'); return; }
    await operate('apply', 'btn-set-save', async () => {
      let ok = true;
      for (const key of changed) {
        const r = await ctl(['set-setting', key, want[key]]);
        if (r.errno !== 0) { ok = false; report(r, ''); break; }
      }
      const running = st.running === '1';
      if (ok && port !== st.cfg_redir_port) {
        // set-port tự khởi động lại AdGuardHome (nếu đang chạy) sau khi cài đặt khác đã được ghi
        ok = report(await ctl(['set-port', port]), t('set.saved'));
      } else if (ok && running) {
        ok = report(await ctl(['restart']), t('set.saved'));
      } else if (ok) {
        toast(t('set.saved'), 'ok');
      }
      state.formLoaded = false;
    });
  }

  // Thư viện bcrypt chỉ nạp khi cần (nặng ~20 KB, chỉ dùng lúc đặt mật khẩu); nạp sẵn khi người dùng chạm vào ô mật khẩu.
  let bcryptPromise = null;
  function getBcrypt() {
    if (!bcryptPromise) {
      bcryptPromise = new Promise((resolve, reject) => {
        const s = document.createElement('script');
        s.src = 'assets/bcrypt.min.js';
        s.onload = () => resolve((window.dcodeIO && window.dcodeIO.bcrypt) || window.bcrypt);
        s.onerror = () => { bcryptPromise = null; reject(new Error('load')); };
        document.head.append(s);
      });
    }
    return bcryptPromise;
  }

  async function setPassword() {
    const p1 = $('#acc-pass').value;
    const p2 = $('#acc-pass2').value;
    if (p1.length < 8) { toast(t('acc.short'), 'err'); return; }
    if (new TextEncoder().encode(p1).length > 72) { toast(t('acc.long'), 'err'); return; }
    if (p1 !== p2) { toast(t('acc.mismatch'), 'err'); return; }
    await operate('apply', 'btn-acc-set', async () => {
      let hash = '';
      try {
        const lib = await getBcrypt();
        hash = await new Promise((resolve, reject) => { lib.hash(p1, 10, (err, h) => (err ? reject(err) : resolve(h))); });
      } catch (e) { hash = ''; }
      if (!hash) { toast(t('acc.libFailed'), 'err'); return; }
      const r = await ctl(['set-password', hash]);
      if (report(r, t('acc.saved'))) {
        $('#acc-pass').value = '';
        $('#acc-pass2').value = '';
      }
    });
  }

  async function collectDebug() {
    await operate('apply', 'btn-debug', async () => {
      const r = await ctl(['debug']);
      state.debugText = r.stdout;
      state.logKind = 'debug';
      toast(t('maint.debugDone'), 'ok');
    });
    showTab('logs');
  }

  async function resetConfig() {
    if (!(await confirmDialog(t('maint.resetConfirm')))) return;
    await operate('apply', 'btn-reset', async () => {
      const r = await ctl(['reset-config']);
      if (report(r, t('maint.resetDone'))) {
        state.formLoaded = false;
        await loadSnapshot(true, true);
      }
    });
  }

  // ------------------------------------------------------------ tab và vòng cập nhật
  let logTimer = 0;
  function showTab(name, silent) {
    state.tab = name;
    store.set('tab', name);
    $$('.tabs button').forEach((b) => b.classList.toggle('active', b.dataset.tab === name));
    $$('.tab').forEach((s) => s.classList.toggle('active', s.id === `tab-${name}`));
    clearInterval(logTimer);
    if (name === 'logs') {
      $$('#log-chips .chip').forEach((c) => c.classList.toggle('active', c.dataset.log === state.logKind));
      loadLog(false);
      // Nhật ký tự cập nhật khi đang xem (giữ nguyên vị trí cuộn nếu bạn đang đọc phía trên)
      logTimer = setInterval(() => { if (document.visibilityState === 'visible' && state.logKind !== 'debug') loadLog(true); }, 4000);
    }
    if (!silent) schedulePoll();
  }

  let pollTimer = 0;
  function schedulePoll() {
    clearTimeout(pollTimer);
    const st = state.status || {};
    const loadingFilters = st.running === '1' && Number(st.filters_pending) > 0;
    const delay = loadingFilters ? 2000 : (state.tab === 'overview' ? 5000 : 9000);
    pollTimer = setTimeout(async () => {
      if (document.visibilityState === 'visible' && !state.busy) await refresh();
      schedulePoll();
    }, delay);
  }

  // ------------------------------------------------------------ khởi tạo
  function init() {
    state.lang = store.get('lang') === 'en' ? 'en' : 'vi';
    applyI18n();

    $$('.tabs button').forEach((b) => b.addEventListener('click', () => showTab(b.dataset.tab)));
    $$('#lang-chips .chip').forEach((b) => b.addEventListener('click', () => setLang(b.dataset.lang, true)));
    $$('#log-chips .chip').forEach((b) => b.addEventListener('click', () => { state.logKind = b.dataset.log; showTab('logs'); }));
    $('#btn-start').addEventListener('click', actions.start);
    $('#btn-stop').addEventListener('click', actions.stop);
    $('#btn-restart').addEventListener('click', actions.restart);
    $('#btn-panel').addEventListener('click', openPanel);
    $('#btn-dns-save').addEventListener('click', saveUpstream);
    $('#btn-filters-save').addEventListener('click', saveFilters);
    let searchTimer = 0;
    $('#filters-search').addEventListener('input', () => { clearTimeout(searchTimer); searchTimer = setTimeout(renderFilters, 120); });
    $('#btn-set-save').addEventListener('click', saveSettings);
    $('#btn-acc-set').addEventListener('click', setPassword);
    $('#acc-pass').addEventListener('focus', () => { getBcrypt().catch(() => {}); }, { once: true });
    $('#btn-debug').addEventListener('click', collectDebug);
    $('#btn-reset').addEventListener('click', resetConfig);
    $('#btn-log-refresh').addEventListener('click', () => loadLog(false));
    $('#upstream-list').addEventListener('input', () => { state.upstreamDirty = true; renderPresets(); });
    document.addEventListener('visibilitychange', () => { if (document.visibilityState === 'visible' && !state.busy) refresh(); });

    // Hiện ngay dữ liệu lần trước (nếu có), rồi cập nhật ngầm
    hydrateFromCache();
    if (state.device) resolveCountry().then(renderFilters); // tín hiệu của máy đã có trong bộ nhớ đệm: hiện quốc gia ngay
    const savedTab = store.get('tab');
    if (['overview', 'dns', 'settings', 'logs'].includes(savedTab)) showTab(savedTab, true);
    renderAll();
    renderFilters();
    if (bridge()) {
      loadSnapshot(false).then(() => schedulePoll());
    } else {
      state.fresh = true;
      renderAll();
    }
  }

  init();
})();
