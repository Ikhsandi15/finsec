// FINSEC-TERM Web Client 2.0 (Bloomberg / Koyfin Style)
let state = {
  pane1: [],
  pane2: [],
  pane3: [],
  filtered1: [],
  filtered2: [],
  filtered3: [],
  stats: {
    total_items: 0,
    unread_count: 0,
    crit_count: 0,
    last_sync: null,
    is_syncing: false,
  },
  searchQuery: '',
  selectedItem: null,
  activePane: 1,
  currentView: 'grid', // 'grid', 'pane1', 'pane2', 'pane3'
  topicFilter: 'all',  // sub-topic filter in focus mode
  sortBy: 'latest',    // 'latest' or 'priority'
};

// DOM Elements
const elUtc = document.getElementById('clock-utc');
const elWib = document.getElementById('clock-wib');
const elTotal = document.getElementById('stat-total');
const elUnread = document.getElementById('stat-unread');
const elCrit = document.getElementById('stat-crit');
const elSyncStatus = document.getElementById('sync-status');
const elSearchInput = document.getElementById('search-input');
const elBtnSync = document.getElementById('btn-sync');
const elBtnMarkAll = document.getElementById('btn-mark-all');
const elBtnSort = document.getElementById('btn-sort');
const elSortIcon = document.getElementById('sort-icon');
const elSortText = document.getElementById('sort-text');
const elBtnTheme = document.getElementById('btn-theme');
const elThemeText = document.getElementById('theme-text');

// Live Ticker Strip Elements (100% Real Data)
const elRateUsdIdr = document.getElementById('rate-usdidr');
const elRateBtcUsd = document.getElementById('rate-btcusd');
const elWireTickerContent = document.getElementById('wire-ticker-content');

// Views
const elViewGrid = document.getElementById('view-grid');
const elViewFocus = document.getElementById('view-focus');
const elViewTabs = document.querySelectorAll('.view-tab');

// Pane List Elements (Grid View)
const elList1 = document.getElementById('list-pane1');
const elList2 = document.getElementById('list-pane2');
const elList3 = document.getElementById('list-pane3');

const elCount1 = document.getElementById('count-pane1');
const elCount2 = document.getElementById('count-pane2');
const elCount3 = document.getElementById('count-pane3');

const elTabCount1 = document.getElementById('tab-count-1');
const elTabCount2 = document.getElementById('tab-count-2');
const elTabCount3 = document.getElementById('tab-count-3');

// Focus Mode Elements
const elFocusBadge = document.getElementById('focus-pane-badge');
const elFocusDesc = document.getElementById('focus-pane-desc');
const elFocusTopicPills = document.getElementById('focus-topic-pills');
const elFocusFeedList = document.getElementById('focus-feed-list');
const elReaderBody = document.getElementById('reader-body');
const elReaderOpenLink = document.getElementById('reader-open-link');
const elReaderMarkRead = document.getElementById('reader-mark-read');
const elReaderCopyLink = document.getElementById('reader-copy-link');
const elToastBox = document.getElementById('toast-box');

// Drawer Elements (Grid View modal)
const elDrawer = document.getElementById('inspector-drawer');
const elDrawerBackdrop = document.getElementById('drawer-backdrop');
const elDrawerClose = document.getElementById('drawer-close');
const elDrawerContent = document.getElementById('drawer-content');
const elDrawerOpenLink = document.getElementById('drawer-open-link');

// Formatters
function padZero(n) { return n < 10 ? '0' + n : n; }

function updateClocks() {
  const now = new Date();
  
  // UTC
  const utcHours = padZero(now.getUTCHours());
  const utcMins = padZero(now.getUTCMinutes());
  const utcSecs = padZero(now.getUTCSeconds());
  elUtc.textContent = `UTC ${utcHours}:${utcMins}:${utcSecs}`;

  // WIB (UTC+7)
  const wibDate = new Date(now.getTime() + (7 * 3600 * 1000) + (now.getTimezoneOffset() * 60 * 1000));
  const wibHours = padZero(wibDate.getHours());
  const wibMins = padZero(wibDate.getMinutes());
  const wibSecs = padZero(wibDate.getSeconds());
  elWib.textContent = `WIB ${wibHours}:${wibMins}:${wibSecs}`;
}
setInterval(updateClocks, 1000);
updateClocks();

// 1. REAL LIVE FOREX & CRYPTO QUOTES (FROM GO BACKEND / YAHOO FINANCE & BINANCE)
async function fetchLiveRates() {
  try {
    const res = await fetch('/api/rates');
    if (res.ok) {
      const data = await res.json();
      if (data && data.usdidr > 0) {
        elRateUsdIdr.textContent = `Rp ${Math.round(data.usdidr).toLocaleString('id-ID')}`;
        elRateUsdIdr.title = `Sumber: ${data.source}`;
        const tagEl = document.getElementById('rate-usdidr-tag');
        if (tagEl) {
          if (data.source && data.source.toLowerCase().includes('fixer')) {
            tagEl.textContent = 'Fixer.io';
            tagEl.className = 'ticker-source-tag tag-fixer';
            tagEl.title = data.source;
          } else {
            tagEl.textContent = 'Spot FX';
            tagEl.className = 'ticker-source-tag tag-fallback';
            tagEl.title = data.source;
          }
        }
      }
      if (data && data.btcusd > 0) {
        elRateBtcUsd.textContent = `$${Math.round(data.btcusd).toLocaleString('en-US')}`;
        elRateBtcUsd.title = 'Sumber: Binance Spot';
      }
    }
  } catch (e) {
    if (elRateUsdIdr) elRateUsdIdr.textContent = '--';
  }
}
setInterval(fetchLiveRates, 30000);
fetchLiveRates();

// 2. REAL-TIME FLASH WIRE (100% REAL FROM USER FEEDS - ZERO HOAX)
function updateBreakingWire() {
  if (!elWireTickerContent) return;
  const allItems = [...state.pane1, ...state.pane2, ...state.pane3];
  if (!allItems.length) return;

  // Sort by published_at descending (latest first)
  const latestItems = allItems.sort((a, b) => parseTime(b.published_at) - parseTime(a.published_at)).slice(0, 15);

  const html = latestItems.map(it => {
    return `
      <span class="wire-item" data-id="${it.id}" data-pane="${it.pane}">
        <span class="wire-src">[${escapeHtml(it.source)}]</span>
        <span>${escapeHtml(it.title)}</span>
        <span class="wire-time">(${relativeTime(it.published_at)})</span>
        <span class="wire-sep">◆</span>
      </span>
    `;
  }).join('');

  // Duplicate for seamless marquee loop
  elWireTickerContent.innerHTML = html + html;

  elWireTickerContent.querySelectorAll('.wire-item').forEach(item => {
    item.addEventListener('click', e => {
      e.preventDefault();
      const id = item.getAttribute('data-id');
      const pane = parseInt(item.getAttribute('data-pane'), 10) || 1;
      selectItem(id, pane, true);
    });
  });
}

function parseTime(val) {
  if (!val) return 0;
  const d = new Date(val);
  const t = d.getTime();
  return isNaN(t) ? 0 : t;
}

function relativeTime(dateStr) {
  if (!dateStr) return 'recent';
  const d = new Date(dateStr);
  const diffSec = Math.floor((new Date() - d) / 1000);
  if (diffSec < 60) return 'just now';
  if (diffSec < 3600) return `${Math.floor(diffSec / 60)}m ago`;
  if (diffSec < 86400) return `${Math.floor(diffSec / 3600)}h ago`;
  return `${Math.floor(diffSec / 86400)}d ago`;
}

function formatWIB(dateStr) {
  if (!dateStr) return 'N/A';
  const d = new Date(dateStr);
  return d.toLocaleString('id-ID', {
    timeZone: 'Asia/Jakarta',
    year: 'numeric',
    month: 'short',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  }) + ' WIB';
}

function escapeHtml(text) {
  if (!text) return '';
  const map = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#039;' };
  return text.replace(/[&<>"']/g, m => map[m]);
}

function renderBadges(badges, cves) {
  let html = '';
  if (badges && badges.length) {
    badges.forEach(b => {
      let cls = 'tag-badge ';
      if (b === 'STACK-CRIT') cls += 'tag-stack-crit';
      else if (b === 'TECH-AI') cls += 'tag-tech-ai';
      else if (b === 'ID-MACRO') cls += 'tag-id-macro';
      else if (b === 'ID-POL') cls += 'tag-id-pol';
      else if (b === 'GLOBAL-MKT') cls += 'tag-global-mkt';
      else if (b === 'CVE') cls += 'tag-cve';
      else cls += 'tag-tech-ai';
      html += `<span class="${cls}">${escapeHtml(b)}</span>`;
    });
  }
  if (cves && cves.length) {
    cves.forEach(c => {
      html += `<span class="tag-badge tag-cve">${escapeHtml(c)}</span>`;
    });
  }
  return html;
}

// TOPIC FILTERS CONFIGURATION
const topicPresets = {
  pane1: [
    { id: 'all', label: 'ALL FEEDS' },
    { id: 'ai', label: 'AI & FRONTIER TECH' },
    { id: 'dev', label: 'DEVELOPER & OPEN SOURCE' },
    { id: 'cisa', label: 'CISA KEV' },
    { id: 'stack-crit', label: 'STACK CRITICAL' },
    { id: 'rce', label: 'RCE / ZERO-DAY' },
    { id: 'golang', label: 'GOLANG' },
    { id: 'infra', label: 'CLOUD & DATABASE' },
  ],
  pane2: [
    { id: 'all', label: 'ALL MACRO' },
    { id: 'fed', label: 'THE FED / FOMC' },
    { id: 'cpi', label: 'INFLATION / CPI' },
    { id: 'treasury', label: 'BONDS & YIELDS' },
    { id: 'oil', label: 'CRUDE OIL' },
    { id: 'crypto', label: 'CRYPTO / BTC' },
  ],
  pane3: [
    { id: 'all', label: 'ALL INDONESIA' },
    { id: 'bi', label: 'BANK INDONESIA / BI RATE' },
    { id: 'rupiah', label: 'RUPIAH (USD/IDR)' },
    { id: 'ihsg', label: 'IHSG & EQUITY' },
    { id: 'sbn', label: 'SBN & BONDS' },
    { id: 'ojk', label: 'OJK & BANKING' },
    { id: 'pol', label: 'CABINET & FISCAL' },
    { id: 'cyber', label: 'BSSN & CSIRT SIBER' },
  ],
};

function matchesTopic(item, topic) {
  if (topic === 'all') return true;
  const text = (item.title + ' ' + (item.description || '') + ' ' + (item.source || '')).toLowerCase();
  switch (topic) {
    // Pane 1
    case 'ai': return (item.badges && item.badges.includes('TECH-AI')) || text.includes('ai') || text.includes('artificial intelligence') || text.includes('llm') || text.includes('openai') || text.includes('anthropic') || text.includes('nvidia') || text.includes('gpu') || text.includes('claude') || text.includes('gemini') || text.includes('deepseek') || text.includes('chip') || text.includes('semiconductor');
    case 'dev': return text.includes('github') || text.includes('developer') || text.includes('framework') || text.includes('rust') || text.includes('python') || text.includes('docker') || text.includes('kubernetes') || text.includes('open source') || text.includes('release') || text.includes('vulnerability');
    case 'cisa': return item.is_cisa || item.source.toLowerCase().includes('cisa');
    case 'stack-crit': return item.badges && item.badges.includes('STACK-CRIT');
    case 'rce': return text.includes('rce') || text.includes('remote code execution') || text.includes('zero-day');
    case 'golang': return text.includes('golang') || text.includes('go ');
    case 'infra': return text.includes('nginx') || text.includes('redis') || text.includes('postgres') || text.includes('cloud') || text.includes('database') || text.includes('server');

    // Pane 2
    case 'fed': return text.includes('fed') || text.includes('fomc') || text.includes('powell') || text.includes('interest rate');
    case 'cpi': return text.includes('cpi') || text.includes('pce') || text.includes('inflation') || text.includes('inflasi');
    case 'treasury': return text.includes('treasury') || text.includes('yield') || text.includes('bond');
    case 'oil': return text.includes('oil') || text.includes('crude') || text.includes('energy');
    case 'crypto': return text.includes('bitcoin') || text.includes('crypto') || text.includes('coindesk');

    // Pane 3
    case 'bi': return text.includes('bank indonesia') || text.includes('bi rate') || text.includes('gubernur bi');
    case 'rupiah': return text.includes('rupiah') || text.includes('idr') || text.includes('dolar');
    case 'ihsg': return text.includes('ihsg') || text.includes('saham') || text.includes('bursa');
    case 'sbn': return text.includes('sbn') || text.includes('sun') || text.includes('obligasi');
    case 'ojk': return text.includes('ojk') || text.includes('bank') || text.includes('perbankan');
    case 'pol': return text.includes('prabowo') || text.includes('kemenkeu') || text.includes('apbn') || text.includes('danantara') || text.includes('purbaya');
    case 'cyber': return text.includes('bssn') || text.includes('csirt') || text.includes('siber') || text.includes('kebocoran') || text.includes('hacker');
  }
  return true;
}

// Apply Filtering & Sorting
function applyFilter() {
  const q = state.searchQuery.trim().toLowerCase();
  const filterFn = item => {
    if (!q) return true;
    if (item.title && item.title.toLowerCase().includes(q)) return true;
    if (item.description && item.description.toLowerCase().includes(q)) return true;
    if (item.source && item.source.toLowerCase().includes(q)) return true;
    if (item.cves && item.cves.some(c => c.toLowerCase().includes(q))) return true;
    if (item.cwes && item.cwes.some(c => c.toLowerCase().includes(q))) return true;
    if (item.badges && item.badges.some(b => b.toLowerCase().includes(q))) return true;
    return false;
  };

  const sortFn = (a, b) => {
    const timeA = parseTime(a.published_at);
    const timeB = parseTime(b.published_at);
    if (state.sortBy === 'latest') {
      return timeB - timeA;
    }
    if (b.priority_score !== a.priority_score) {
      return b.priority_score - a.priority_score;
    }
    return timeB - timeA;
  };

  state.filtered1 = state.pane1.filter(filterFn).sort(sortFn);
  state.filtered2 = state.pane2.filter(filterFn).sort(sortFn);
  state.filtered3 = state.pane3.filter(filterFn).sort(sortFn);
}

// Render Card List
function renderList(container, items, paneNum, isFocusMode = false) {
  if (!items || items.length === 0) {
    container.innerHTML = `
      <div class="empty-state">
        <span>Tidak ada berita yang cocok dengan kriteria filter</span>
      </div>
    `;
    return;
  }

  container.innerHTML = items.map((item, idx) => {
    const isUnread = !item.is_read;
    const isCrit = item.severity === 'CRIT' || (item.badges && item.badges.includes('STACK-CRIT'));
    const isSelected = state.selectedItem && state.selectedItem.id === item.id;
    
    let cardClass = 'feed-card';
    if (isUnread) cardClass += ' unread';
    if (isCrit) cardClass += ' crit';
    if (!isUnread) cardClass += ' read';
    if (isSelected) cardClass += ' active';

    const badgesHtml = renderBadges(item.badges, item.cves);

    return `
      <div class="${cardClass}" data-id="${item.id}" data-pane="${paneNum}">
        <div class="feed-card-header">
          <span class="source-badge" title="${escapeHtml(item.source)}">${escapeHtml(item.source)}</span>
          <span class="time-stamp">${relativeTime(item.published_at)}</span>
        </div>
        <div class="feed-card-title">${escapeHtml(item.title)}</div>
        ${badgesHtml ? `<div class="badges-row">${badgesHtml}</div>` : ''}
        ${item.description ? `<div class="feed-card-snippet">${escapeHtml(item.description)}</div>` : ''}
      </div>
    `;
  }).join('');

  // Attach click listener
  container.querySelectorAll('.feed-card').forEach(card => {
    card.addEventListener('click', () => {
      const id = card.getAttribute('data-id');
      const pane = parseInt(card.getAttribute('data-pane'), 10);
      selectItem(id, pane);
    });
  });
}

function updateUI() {
  applyFilter();

  // Grid view counts
  elCount1.textContent = state.filtered1.length;
  elCount2.textContent = state.filtered2.length;
  elCount3.textContent = state.filtered3.length;

  // Tab counts
  elTabCount1.textContent = state.filtered1.length;
  elTabCount2.textContent = state.filtered2.length;
  elTabCount3.textContent = state.filtered3.length;

  elTotal.textContent = `${state.stats.total_items} ITEMS`;
  elUnread.textContent = `${state.stats.unread_count} UNREAD`;
  elCrit.textContent = `${state.stats.crit_count} CRITICAL`;

  if (state.stats.is_syncing) {
    elSyncStatus.innerHTML = `<span class="spinner-icon">↻</span> SYNCING...`;
    elBtnSync.disabled = true;
  } else {
    elSyncStatus.innerHTML = `<span style="color:var(--green)">●</span> LIVE`;
    elBtnSync.disabled = false;
  }

  if (state.currentView === 'grid') {
    elViewGrid.style.display = 'grid';
    elViewFocus.style.display = 'none';

    renderList(elList1, state.filtered1, 1);
    renderList(elList2, state.filtered2, 2);
    renderList(elList3, state.filtered3, 3);
  } else {
    elViewGrid.style.display = 'none';
    elViewFocus.style.display = 'grid';
    renderFocusMode();
  }
}

// RENDER FOCUS MASTER-DETAIL MODE
function renderFocusMode() {
  let targetItems = [];
  let paneName = '';
  let paneDesc = '';
  let paneNum = 1;

  if (state.currentView === 'pane1') {
    targetItems = state.filtered1;
    paneName = '01 // TECH & THREAT RADAR';
    paneDesc = 'AI Engineering, Server Infrastructure & Exploit Tracking';
    paneNum = 1;
  } else if (state.currentView === 'pane2') {
    targetItems = state.filtered2;
    paneName = '02 // GLOBAL MACRO & FOREX';
    paneDesc = 'Central Bank Policy, Inflation, Sovereign Debt & Energy';
    paneNum = 2;
  } else if (state.currentView === 'pane3') {
    targetItems = state.filtered3;
    paneName = '03 // INDONESIA MACRO & POL';
    paneDesc = 'Bank Indonesia, Rupiah Spot, Fiscal Budget & Cyber Incidents';
    paneNum = 3;
  }

  elFocusBadge.textContent = paneName;
  elFocusDesc.textContent = paneDesc;

  // Render Topic Pills
  const presets = topicPresets[state.currentView] || [];
  elFocusTopicPills.innerHTML = presets.map(p => {
    const isActive = state.topicFilter === p.id;
    return `<button class="topic-pill ${isActive ? 'active' : ''}" data-topic="${p.id}">${escapeHtml(p.label)}</button>`;
  }).join('');

  elFocusTopicPills.querySelectorAll('.topic-pill').forEach(btn => {
    btn.addEventListener('click', () => {
      state.topicFilter = btn.getAttribute('data-topic');
      renderFocusMode();
    });
  });

  // Filter items by topic
  const topicFiltered = targetItems.filter(item => matchesTopic(item, state.topicFilter));

  // Render Master List
  renderList(elFocusFeedList, topicFiltered, paneNum, true);

  // If no item is selected or current selected item not in this list, select first item automatically
  if (topicFiltered.length > 0) {
    if (!state.selectedItem || !topicFiltered.some(it => it.id === state.selectedItem.id)) {
      selectItem(topicFiltered[0].id, paneNum, false);
    } else {
      renderReaderPane(state.selectedItem);
    }
  } else {
    elReaderBody.innerHTML = `
      <div class="reader-empty-state">
        <div class="empty-icon-box">EMPTY</div>
        <h3>NO MATCHING INTELLIGENCE</h3>
        <p>No feeds match the filter criteria: <strong>${escapeHtml(state.topicFilter.toUpperCase())}</strong>.</p>
      </div>
    `;
  }
}

// Select Item
function selectItem(id, paneNum, openDrawerInGrid = true) {
  let item = null;
  const allItems = [...state.pane1, ...state.pane2, ...state.pane3];
  item = allItems.find(it => it.id === id);
  if (!item) return;

  state.selectedItem = item;
  state.activePane = paneNum;

  // Mark as read in state & backend
  if (!item.is_read) {
    item.is_read = true;
    fetch('/api/read', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id: item.id }),
    }).catch(console.error);
  }

  if (state.currentView === 'grid') {
    if (openDrawerInGrid) {
      openDrawer(item);
    }
    updateUI();
  } else {
    // In focus mode, update the reader pane right away
    renderReaderPane(item);
    // highlight selected card in list
    document.querySelectorAll('#focus-feed-list .feed-card').forEach(c => {
      if (c.getAttribute('data-id') === item.id) {
        c.classList.add('active');
        c.classList.remove('unread');
        c.classList.add('read');
      } else {
        c.classList.remove('active');
      }
    });
  }
}

// TOAST NOTIFICATIONS
let toastTimer = null;
function showToast(msg) {
  if (!elToastBox) return;
  elToastBox.textContent = msg;
  elToastBox.classList.add('show');
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => {
    elToastBox.classList.remove('show');
  }, 2200);
}

// RENDER DEDICATED READER PANE IN FOCUS MODE
function renderReaderPane(item) {
  if (!item) return;
  const badgesHtml = renderBadges(item.badges, item.cves);
  const paneTitle = state.currentView === 'pane1' ? '01 // TECH & THREAT' : (state.currentView === 'pane2' ? '02 // GLOBAL MACRO' : '03 // INDONESIA MACRO');

  elReaderBody.innerHTML = `
    <div class="reader-breadcrumb">
      <span class="bc-radar">RADAR</span>
      <span class="bc-sep">/</span>
      <span class="bc-pane">${escapeHtml(paneTitle)}</span>
      <span class="bc-sep">/</span>
      <span class="bc-src">${escapeHtml(item.source)}</span>
    </div>

    <div class="reader-article-header">
      <div class="badges-row" style="margin-bottom: 6px;">${badgesHtml}</div>
      <h1 class="reader-main-title">${escapeHtml(item.title)}</h1>

      <div class="reader-meta-cards">
        <div class="reader-meta-card">
          <span class="meta-k">SOURCE</span>
          <span class="meta-v" style="color:var(--col-info);">${escapeHtml(item.source)}</span>
        </div>
        <div class="reader-meta-card">
          <span class="meta-k">PUBLISHED</span>
          <span class="meta-v">${formatWIB(item.published_at)}</span>
        </div>
        <div class="reader-meta-card">
          <span class="meta-k">SEVERITY</span>
          <span class="meta-v" style="color:${item.severity === 'CRIT' ? 'var(--col-crit)' : 'var(--col-warn)'}">
            ${escapeHtml(item.severity || 'INFO')} (SCORE: ${item.priority_score})
          </span>
        </div>
        ${item.cves && item.cves.length ? `
          <div class="reader-meta-card">
            <span class="meta-k">CVE IDENTIFIERS</span>
            <span class="meta-v">
              ${item.cves.map(c => `<a href="https://nvd.nist.gov/vuln/detail/${encodeURIComponent(c)}" target="_blank" style="color:var(--col-crit); font-weight:700; text-decoration:none; margin-right:6px;">${escapeHtml(c)} ↗</a>`).join('')}
            </span>
          </div>
        ` : ''}
        ${item.vendor_project || item.product ? `
          <div class="reader-meta-card">
            <span class="meta-k">AFFECTS / VENDOR</span>
            <span class="meta-v">${escapeHtml(item.vendor_project || '')} / ${escapeHtml(item.product || '')}</span>
          </div>
        ` : ''}
      </div>
    </div>

    <div class="reader-content-block">
      ${escapeHtml(item.description || 'No extended synopsis provided in this syndicated feed. Click "ORIGINAL SOURCE" above to inspect the complete article.')}
    </div>
  `;

  elReaderOpenLink.href = item.url || '#';
}

// Drawer for Grid Mode
function openDrawer(item) {
  const badgesHtml = renderBadges(item.badges, item.cves);

  elDrawerContent.innerHTML = `
    <div class="inspector-main-title">${escapeHtml(item.title)}</div>

    <div class="meta-grid">
      <div class="meta-label">Sumber:</div>
      <div class="meta-val"><strong style="color:var(--cyan)">${escapeHtml(item.source)}</strong></div>

      <div class="meta-label">Waktu:</div>
      <div class="meta-val">${formatWIB(item.published_at)} (${relativeTime(item.published_at)})</div>

      <div class="meta-label">Severity:</div>
      <div class="meta-val"><strong style="color:${item.severity === 'CRIT' ? 'var(--red-crit)' : 'var(--amber)'}">${escapeHtml(item.severity || 'INFO')}</strong> (Skor: ${item.priority_score})</div>

      ${item.cves && item.cves.length ? `
        <div class="meta-label">CVE IDs:</div>
        <div class="meta-val">${item.cves.map(c => `<a href="https://nvd.nist.gov/vuln/detail/${encodeURIComponent(c)}" target="_blank" style="color:var(--red-crit); font-weight:700; margin-right:6px;">${escapeHtml(c)} ↗</a>`).join('')}</div>
      ` : ''}

      ${badgesHtml ? `
        <div class="meta-label">Tags:</div>
        <div class="meta-val" style="display:flex; flex-wrap:wrap; gap:4px;">${badgesHtml}</div>
      ` : ''}

      <div class="meta-label">Target URL:</div>
      <div class="meta-val"><a href="${escapeHtml(item.url)}" target="_blank" style="color:var(--cyan); word-break:break-all;">${escapeHtml(item.url)}</a></div>
    </div>

    <div class="inspector-description">${escapeHtml(item.description || 'Tidak ada ringkasan teks panjang.')}</div>
  `;

  elDrawerOpenLink.href = item.url || '#';
  elDrawer.classList.add('open');
  elDrawerBackdrop.classList.add('open');
}

function closeDrawer() {
  elDrawer.classList.remove('open');
  elDrawerBackdrop.classList.remove('open');
}

elDrawerClose.addEventListener('click', closeDrawer);
elDrawerBackdrop.addEventListener('click', closeDrawer);

// VIEW SWITCHING (Tabs)
function switchView(targetView) {
  state.currentView = targetView;
  state.topicFilter = 'all';

  elViewTabs.forEach(t => {
    if (t.getAttribute('data-view') === targetView) {
      t.classList.add('active');
    } else {
      t.classList.remove('active');
    }
  });

  updateUI();
}

elViewTabs.forEach(tab => {
  tab.addEventListener('click', () => {
    const view = tab.getAttribute('data-view');
    switchView(view);
  });
});

// "Fokus" buttons on pane headers in grid mode
document.querySelectorAll('.btn-focus-pane').forEach(btn => {
  btn.addEventListener('click', e => {
    e.stopPropagation();
    const target = btn.getAttribute('data-target');
    switchView(target);
  });
});

// API Calls
async function fetchFeeds() {
  try {
    const res = await fetch('/api/feeds');
    const data = await res.json();
    handleIncomingData(data);
  } catch (err) {
    console.error('Failed to fetch feeds:', err);
  }
}

function handleIncomingData(data) {
  if (!data) return;
  state.pane1 = data.pane1_cve || [];
  state.pane2 = data.pane2_global_macro || [];
  state.pane3 = data.pane3_id_macro || [];
  if (data.stats) {
    state.stats = data.stats;
  }
  updateUI();
  updateBreakingWire();
}

function triggerSync() {
  state.stats.is_syncing = true;
  updateUI();
  fetch('/api/refresh', { method: 'POST' }).catch(console.error);
}

function markAllRead() {
  fetch('/api/read', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ all: true, pane: 0 }),
  }).then(fetchFeeds).catch(console.error);
}

elBtnSync.addEventListener('click', triggerSync);
elBtnMarkAll.addEventListener('click', markAllRead);

// Sort Toggle
elBtnSort.addEventListener('click', () => {
  if (state.sortBy === 'latest') {
    state.sortBy = 'priority';
    if (elSortIcon) elSortIcon.textContent = '';
    elSortText.textContent = 'PRIORITY FIRST';
  } else {
    state.sortBy = 'latest';
    if (elSortIcon) elSortIcon.textContent = '';
    elSortText.textContent = 'LATEST FIRST';
  }
  updateUI();
});

if (elReaderCopyLink) {
  elReaderCopyLink.addEventListener('click', () => {
    if (state.selectedItem && state.selectedItem.url) {
      navigator.clipboard.writeText(state.selectedItem.url).then(() => {
        showToast('COPIED LINK TO CLIPBOARD');
      }).catch(() => {
        showToast('COULD NOT COPY LINK');
      });
    }
  });
}

// Live Search Input
elSearchInput.addEventListener('input', e => {
  state.searchQuery = e.target.value;
  updateUI();
});

// SSE Setup
function setupSSE() {
  const evtSource = new EventSource('/api/events');

  evtSource.addEventListener('initial', e => {
    try {
      const data = JSON.parse(e.data);
      handleIncomingData(data);
    } catch (err) { console.error(err); }
  });

  evtSource.addEventListener('sync_started', () => {
    state.stats.is_syncing = true;
    updateUI();
  });

  evtSource.addEventListener('sync_finished', e => {
    try {
      const data = JSON.parse(e.data);
      handleIncomingData(data);
    } catch (err) { console.error(err); }
  });

  evtSource.addEventListener('feeds_updated', e => {
    try {
      const data = JSON.parse(e.data);
      handleIncomingData(data);
    } catch (err) { console.error(err); }
  });
}

// Global Keyboard Shortcuts
window.addEventListener('keydown', e => {
  if (document.activeElement === elSearchInput) {
    if (e.key === 'Escape') {
      elSearchInput.blur();
      elSearchInput.value = '';
      state.searchQuery = '';
      updateUI();
    }
    return;
  }

  if (e.key === '/') {
    e.preventDefault();
    elSearchInput.focus();
    elSearchInput.select();
  } else if (e.key === 'Escape') {
    closeDrawer();
  } else if (e.key === 'r' || e.key === 'R') {
    triggerSync();
  } else if (e.key === 'm' || e.key === 'M') {
    markAllRead();
  } else if (e.key === '0' || e.key === '`') {
    switchView('grid');
  } else if (e.key === '1') {
    switchView('pane1');
  } else if (e.key === '2') {
    switchView('pane2');
  } else if (e.key === '3') {
    switchView('pane3');
  } else if (e.key === 'o' || e.key === 'O') {
    if (state.selectedItem && state.selectedItem.url) {
      window.open(state.selectedItem.url, '_blank');
    }
  } else if (e.key === 'c' || e.key === 'C') {
    if (state.selectedItem && state.selectedItem.url) {
      navigator.clipboard.writeText(state.selectedItem.url).then(() => {
        showToast('COPIED LINK TO CLIPBOARD');
      });
    }
  } else if (e.key === 'j' || e.key === 'ArrowDown') {
    navigateItem(1);
  } else if (e.key === 'k' || e.key === 'ArrowUp') {
    navigateItem(-1);
  } else if (e.key === 't' || e.key === 'T') {
    toggleTheme();
  }
});

function navigateItem(direction) {
  if (state.currentView === 'grid') return;
  const list = state.currentView === 'pane1' ? state.filtered1 : (state.currentView === 'pane2' ? state.filtered2 : state.filtered3);
  const topicFiltered = list.filter(item => matchesTopic(item, state.topicFilter));
  if (!topicFiltered.length) return;

  let currentIndex = topicFiltered.findIndex(it => it.id === (state.selectedItem && state.selectedItem.id));
  if (currentIndex === -1) currentIndex = 0;
  else currentIndex += direction;

  if (currentIndex < 0) currentIndex = 0;
  if (currentIndex >= topicFiltered.length) currentIndex = topicFiltered.length - 1;

  const target = topicFiltered[currentIndex];
  selectItem(target.id, state.activePane, false);

  // Scroll active card into view
  const activeCard = document.querySelector(`#focus-feed-list .feed-card[data-id="${target.id}"]`);
  if (activeCard) {
    activeCard.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
  }
}

// THEME CONTROLLER (INSTITUTIONAL DARK & LIGHT)
function initTheme() {
  const saved = localStorage.getItem('finsec-theme') || (window.matchMedia && window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark');
  setTheme(saved);
}

function setTheme(theme) {
  document.documentElement.setAttribute('data-theme', theme);
  localStorage.setItem('finsec-theme', theme);
  if (elThemeText) {
    elThemeText.textContent = theme.toUpperCase();
  }
}

function toggleTheme() {
  const current = document.documentElement.getAttribute('data-theme') || 'dark';
  const next = current === 'dark' ? 'light' : 'dark';
  setTheme(next);
}

if (elBtnTheme) {
  elBtnTheme.addEventListener('click', toggleTheme);
}

// Initial boot
initTheme();
fetchFeeds();
setupSSE();
