/* Nhận diện quốc gia cho danh sách chặn theo quốc gia.
 *
 * Ưu tiên tín hiệu từ chính chiếc máy: mạng di động đang dùng, SIM, vùng hệ thống, múi giờ.  Chúng chính
 * xác hơn IP (IP di động hay bị định vị sai, hoặc là IP của VPN) và không cần gọi dịch vụ nào.  Chỉ khi không
 * có tín hiệu nào, hoặc các tín hiệu mâu thuẫn nhau (ví dụ đang đi nước ngoài), mới tra IP để phân xử.
 * Ngôn ngữ của máy KHÔNG được dùng: người ở Việt Nam hoàn toàn có thể để máy tiếng Anh (en-US).
 */
(function (root, factory) {
  const api = factory();
  if (typeof module === 'object' && module.exports) module.exports = api;
  else root.AGH_GEO = api;
}(typeof self !== 'undefined' ? self : this, () => {
  'use strict';

  // Các nhóm danh sách theo quốc gia mà module có (khớp với config/filters.groups).
  // "ARAB" và "CZ" là nhóm theo ngôn ngữ/khu vực, không phải mã ISO (AR là Argentina).
  const GROUPS = ['VN', 'KR', 'CN', 'ID', 'PL', 'TR', 'HU', 'LT', 'IL', 'SE', 'NO', 'MK', 'IR', 'UA',
    'DE', 'FR', 'ES', 'PT', 'IT', 'NL', 'RU', 'ARAB', 'CZ'];

  // Mã quốc gia ISO -> nhóm danh sách dùng chung ngôn ngữ/khu vực
  const ISO_TO_GROUP = {};
  const alias = (group, codes) => codes.split(' ').forEach((c) => { ISO_TO_GROUP[c] = group; });
  alias('DE', 'AT CH LI');
  alias('ES', 'MX AR CL CO PE VE EC UY PY BO CR PA DO GT HN NI SV CU PR');
  alias('PT', 'BR AO MZ');
  alias('CZ', 'SK');
  alias('RU', 'BY KZ');
  alias('ARAB', 'SA AE EG IQ JO KW LB LY MA DZ TN OM QA BH YE SD SY PS');

  function groupOf(cc) {
    if (!cc) return '';
    const c = String(cc).toUpperCase();
    return ISO_TO_GROUP[c] || (GROUPS.includes(c) ? c : '');
  }

  // Múi giờ -> mã quốc gia ISO.  Chỉ cần các nước có danh sách riêng (kể cả nước chung nhóm) và vài nước
  // phổ biến khác để phát hiện mâu thuẫn; múi giờ lạ (UTC, GMT...) cho kết quả rỗng.
  const TZ_BY_COUNTRY = {
    VN: 'Asia/Ho_Chi_Minh Asia/Saigon',
    KR: 'Asia/Seoul',
    CN: 'Asia/Shanghai Asia/Chongqing Asia/Harbin Asia/Urumqi Asia/Kashgar',
    ID: 'Asia/Jakarta Asia/Pontianak Asia/Makassar Asia/Jayapura',
    PL: 'Europe/Warsaw',
    TR: 'Europe/Istanbul Asia/Istanbul',
    HU: 'Europe/Budapest',
    LT: 'Europe/Vilnius',
    IL: 'Asia/Jerusalem Asia/Tel_Aviv',
    SE: 'Europe/Stockholm',
    NO: 'Europe/Oslo',
    MK: 'Europe/Skopje',
    IR: 'Asia/Tehran',
    UA: 'Europe/Kyiv Europe/Kiev Europe/Uzhgorod Europe/Zaporozhye',
    DE: 'Europe/Berlin Europe/Busingen',
    AT: 'Europe/Vienna',
    CH: 'Europe/Zurich',
    LI: 'Europe/Vaduz',
    FR: 'Europe/Paris',
    ES: 'Europe/Madrid Africa/Ceuta Atlantic/Canary',
    PT: 'Europe/Lisbon Atlantic/Madeira Atlantic/Azores',
    IT: 'Europe/Rome',
    NL: 'Europe/Amsterdam',
    RU: 'Europe/Moscow Europe/Kaliningrad Europe/Samara Europe/Volgograd Europe/Saratov Europe/Ulyanovsk Europe/Kirov '
      + 'Europe/Astrakhan Asia/Yekaterinburg Asia/Omsk Asia/Novosibirsk Asia/Barnaul Asia/Tomsk Asia/Novokuznetsk '
      + 'Asia/Krasnoyarsk Asia/Irkutsk Asia/Chita Asia/Yakutsk Asia/Vladivostok Asia/Magadan Asia/Sakhalin '
      + 'Asia/Kamchatka Asia/Anadyr',
    BY: 'Europe/Minsk',
    KZ: 'Asia/Almaty Asia/Aqtobe Asia/Aqtau Asia/Oral Asia/Qyzylorda Asia/Qostanay',
    SA: 'Asia/Riyadh',
    AE: 'Asia/Dubai',
    EG: 'Africa/Cairo',
    IQ: 'Asia/Baghdad',
    JO: 'Asia/Amman',
    KW: 'Asia/Kuwait',
    LB: 'Asia/Beirut',
    MA: 'Africa/Casablanca',
    DZ: 'Africa/Algiers',
    TN: 'Africa/Tunis',
    QA: 'Asia/Qatar',
    BH: 'Asia/Bahrain',
    OM: 'Asia/Muscat',
    YE: 'Asia/Aden',
    SD: 'Africa/Khartoum',
    SY: 'Asia/Damascus',
    LY: 'Africa/Tripoli',
    PS: 'Asia/Gaza Asia/Hebron',
    CZ: 'Europe/Prague',
    SK: 'Europe/Bratislava',
    MX: 'America/Mexico_City America/Cancun America/Merida America/Monterrey America/Tijuana America/Chihuahua America/Hermosillo America/Mazatlan',
    AR: 'America/Argentina/Buenos_Aires America/Buenos_Aires America/Argentina/Cordoba America/Cordoba',
    CL: 'America/Santiago',
    CO: 'America/Bogota',
    PE: 'America/Lima',
    VE: 'America/Caracas',
    EC: 'America/Guayaquil',
    UY: 'America/Montevideo',
    PY: 'America/Asuncion',
    BO: 'America/La_Paz',
    CR: 'America/Costa_Rica',
    PA: 'America/Panama',
    DO: 'America/Santo_Domingo',
    GT: 'America/Guatemala',
    HN: 'America/Tegucigalpa',
    NI: 'America/Managua',
    SV: 'America/El_Salvador',
    CU: 'America/Havana',
    PR: 'America/Puerto_Rico',
    BR: 'America/Sao_Paulo America/Manaus America/Bahia America/Fortaleza America/Recife America/Belem America/Cuiaba '
      + 'America/Campo_Grande America/Porto_Velho America/Rio_Branco America/Noronha',
    AO: 'Africa/Luanda',
    MZ: 'Africa/Maputo',
    // Không có danh sách riêng, nhưng giúp phát hiện mâu thuẫn (ví dụ SIM Việt Nam đang ở Thái Lan)
    TH: 'Asia/Bangkok',
    SG: 'Asia/Singapore',
    MY: 'Asia/Kuala_Lumpur Asia/Kuching',
    PH: 'Asia/Manila',
    JP: 'Asia/Tokyo',
    IN: 'Asia/Kolkata Asia/Calcutta',
    AU: 'Australia/Sydney Australia/Melbourne Australia/Brisbane Australia/Perth Australia/Adelaide',
    GB: 'Europe/London',
    US: 'America/New_York America/Chicago America/Denver America/Los_Angeles America/Phoenix America/Anchorage Pacific/Honolulu America/Detroit',
    CA: 'America/Toronto America/Vancouver America/Edmonton America/Winnipeg America/Halifax',
    KH: 'Asia/Phnom_Penh',
    LA: 'Asia/Vientiane',
    TW: 'Asia/Taipei',
    HK: 'Asia/Hong_Kong',
  };
  const TZ_TO_ISO = {};
  Object.keys(TZ_BY_COUNTRY).forEach((cc) => TZ_BY_COUNTRY[cc].split(' ').forEach((tz) => { TZ_TO_ISO[tz] = cc; }));

  const tzCountry = (tz) => TZ_TO_ISO[String(tz || '')] || '';

  // "vn,vn" / "VN," -> ['VN'] (SIM kép trả về nhiều mã, cách nhau bằng dấu phẩy)
  function parseCodes(s) {
    const out = [];
    String(s || '').toUpperCase().split(',').map((x) => x.trim()).forEach((c) => {
      if (/^[A-Z]{2}$/.test(c) && !out.includes(c)) out.push(c);
    });
    return out;
  }

  const SOURCES = ['net', 'sim', 'region', 'tz'];

  /**
   * decide(signals, lookupIp) -> { cc, source }
   *   signals  = { net, sim, region, tz }   (net/sim/region: chuỗi "vn,vn"; tz: tên múi giờ IANA)
   *   lookupIp = async () => ({ cc }) | null  (chỉ gọi khi cần)
   * source: net | sim | region | tz | ip | '' (không nhận diện được)
   */
  async function decide(signals, lookupIp) {
    const sig = signals || {};
    const dev = [];
    SOURCES.forEach((k) => {
      const codes = k === 'tz' ? [tzCountry(sig.tz)].filter(Boolean) : parseCodes(sig[k]);
      codes.forEach((cc) => dev.push({ cc, source: k }));
    });
    // Mọi tín hiệu của máy đồng ý với nhau: dùng luôn, không cần gọi dịch vụ ngoài
    if (dev.length && dev.every((d) => d.cc === dev[0].cc)) return { cc: dev[0].cc, source: dev[0].source };

    let ip = null;
    if (typeof lookupIp === 'function') {
      try { ip = await lookupIp(); } catch (e) { ip = null; }
    }
    const ipcc = ip && /^[A-Z]{2}$/.test(String(ip.cc || '').toUpperCase()) ? String(ip.cc).toUpperCase() : '';
    if (!dev.length) return ipcc ? { cc: ipcc, source: 'ip' } : { cc: '', source: '' };
    // Tín hiệu mâu thuẫn: IP phân xử nếu trùng với một tín hiệu; không thì tin tín hiệu ưu tiên cao nhất
    const hit = ipcc && dev.find((d) => d.cc === ipcc);
    return hit ? { cc: hit.cc, source: hit.source } : { cc: dev[0].cc, source: dev[0].source };
  }

  // ---- Tra IP (chỉ trong trình duyệt).  Cloudflare và db-ip không bị danh sách chặn nào chặn; mục 1.1.1.1
  // dùng địa chỉ IP nên không đi qua DNS, tức là không thể bị chặn bằng DNS.
  const parseTrace = (text) => {
    const m = /^loc=([A-Za-z]{2})\s*$/m.exec(String(text));
    return m ? m[1].toUpperCase() : '';
  };
  const parseDbip = (text) => {
    try { return String(JSON.parse(text).countryCode || '').toUpperCase(); } catch (e) { return ''; }
  };
  const IP_SOURCES = [
    ['https://www.cloudflare.com/cdn-cgi/trace', parseTrace],
    ['https://api.db-ip.com/v2/free/self', parseDbip],
  ];
  const IP_FALLBACK = ['https://1.1.1.1/cdn-cgi/trace', parseTrace];

  async function fetchCode(fetchFn, url, parse, timeoutMs) {
    const ctrl = typeof AbortController === 'function' ? new AbortController() : null;
    const timer = setTimeout(() => { if (ctrl) ctrl.abort(); }, timeoutMs);
    try {
      const res = await Promise.race([
        fetchFn(url, { signal: ctrl ? ctrl.signal : undefined, cache: 'no-store' }),
        new Promise((_, rej) => { setTimeout(() => rej(new Error('timeout')), timeoutMs + 200); }),
      ]);
      if (!res || !res.ok) return '';
      const cc = parse(await res.text());
      return /^[A-Z]{2}$/.test(cc) ? cc : '';
    } catch (e) {
      return '';
    } finally {
      clearTimeout(timer);
    }
  }

  // -> { cc, agree } | null.  Hai nguồn độc lập chạy song song; trùng nhau thì chắc chắn hơn.
  async function lookupIp(fetchFn, timeoutMs) {
    const ms = timeoutMs || 5000;
    const f = fetchFn || (typeof fetch === 'function' ? fetch.bind(self) : null);
    if (!f) return null;
    const res = await Promise.all(IP_SOURCES.map(([u, p]) => fetchCode(f, u, p, ms)));
    let cc = res.find(Boolean) || '';
    if (!cc) cc = await fetchCode(f, IP_FALLBACK[0], IP_FALLBACK[1], ms);
    if (!cc) return null;
    return { cc, agree: res[0] !== '' && res[0] === res[1] };
  }

  return { GROUPS, ISO_TO_GROUP, groupOf, tzCountry, parseCodes, decide, lookupIp, parseTrace, parseDbip };
}));
