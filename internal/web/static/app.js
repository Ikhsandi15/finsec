// FINSEC-TERM Web Client
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
  sortBy: 'latest', // 'latest' (newest first) or 'priority'
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

const elList1 = document.getElementById('list-pane1');
const elList2 = document.getElementById('list-pane2');
const elList3 = document.getElementById('list-pane3');

const elCount1 = document.getElementById('count-pane1');
const elCount2 = document.getElementById('count-pane2');
const elCount3 = document.getElementById('count-pane3');

const elDrawer = document.getElementById('inspector-drawer');
const elDrawerBackdrop = document.getElementById('drawer-backdrop');
const elDrawerClose = document.getElementById('drawer-close');
const elDrawerContent = document.getElementById('drawer-content');
const elDrawerOpenLink = document.getElementById('drawer-open-link');

// Formatters
function padZero(n) {
  return n < 10 ? '0' + n : n;
}

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

// Render Badge helper
function renderBadges(badges, cves) {
  let html = '';
  if (badges && badges.length) {
    badges.forEach(b => {
      let cls = 'tag-badge ';
      if (b === 'STACK-CRIT') cls += 'tag-stack-crit';
      else if (b === 'ID-MACRO') cls += 'tag-id-macro';
      else if (b === 'ID-POL') cls += 'tag-id-pol';
      else if (b === 'GLOBAL-MKT') cls += 'tag-global-mkt';
      else if (b === 'CVE') cls += 'tag-cve';
      else cls += 'tag-cve';
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

// Render Feeds List
function renderList(container, items, paneNum) {
  if (!items || items.length === 0) {
    container.innerHTML = `
      <div class="empty-state">
        <span>No feed items available</span>
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

  renderList(elList1, state.filtered1, 1);
  renderList(elList2, state.filtered2, 2);
  renderList(elList3, state.filtered3, 3);

  elCount1.textContent = state.filtered1.length;
  elCount2.textContent = state.filtered2.length;
  elCount3.textContent = state.filtered3.length;

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
}

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
    const timeA = new Date(a.published_at || 0).getTime();
    const timeB = new Date(b.published_at || 0).getTime();
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

// Select Item & Open Inspector
function selectItem(id, paneNum) {
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

  openInspector(item);
  updateUI();
}

function openInspector(item) {
  const badgesHtml = renderBadges(item.badges, item.cves);

  elDrawerContent.innerHTML = `
    <div class="inspector-main-title">${escapeHtml(item.title)}</div>

    <div class="meta-grid">
      <div class="meta-label">Source:</div>
      <div class="meta-val"><strong style="color:var(--cyan)">${escapeHtml(item.source)}</strong></div>

      <div class="meta-label">Published:</div>
      <div class="meta-val">${formatWIB(item.published_at)} (${relativeTime(item.published_at)})</div>

      <div class="meta-label">Severity:</div>
      <div class="meta-val"><strong style="color:${item.severity === 'CRIT' ? 'var(--red-crit)' : 'var(--amber)'}">${escapeHtml(item.severity || 'INFO')}</strong> (Priority Score: ${item.priority_score})</div>

      ${item.cves && item.cves.length ? `
        <div class="meta-label">CVE IDs:</div>
        <div class="meta-val">${item.cves.map(c => `<a href="https://nvd.nist.gov/vuln/detail/${encodeURIComponent(c)}" target="_blank" style="color:var(--red-crit); font-weight:700; margin-right:6px;">${escapeHtml(c)} ↗</a>`).join('')}</div>
      ` : ''}

      ${item.cwes && item.cwes.length ? `
        <div class="meta-label">CWE IDs:</div>
        <div class="meta-val">${escapeHtml(item.cwes.join(', '))}</div>
      ` : ''}

      ${badgesHtml ? `
        <div class="meta-label">Tags:</div>
        <div class="meta-val" style="display:flex; flex-wrap:wrap; gap:4px;">${badgesHtml}</div>
      ` : ''}

      ${item.vendor_project || item.product ? `
        <div class="meta-label">Affects:</div>
        <div class="meta-val"><strong>${escapeHtml(item.vendor_project || '')}</strong> / ${escapeHtml(item.product || '')}</div>
      ` : ''}

      <div class="meta-label">Target URL:</div>
      <div class="meta-val"><a href="${escapeHtml(item.url)}" target="_blank" style="color:var(--cyan); word-break:break-all;">${escapeHtml(item.url)}</a></div>
    </div>

    <div class="inspector-description">${escapeHtml(item.description || 'No extended summary provided in this feed entry.')}</div>
  `;

  elDrawerOpenLink.href = item.url || '#';
  elDrawer.classList.add('open');
  elDrawerBackdrop.classList.add('open');
}

function closeInspector() {
  elDrawer.classList.remove('open');
  elDrawerBackdrop.classList.remove('open');
}

elDrawerClose.addEventListener('click', closeInspector);
elDrawerBackdrop.addEventListener('click', closeInspector);

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
elBtnSort.addEventListener('click', () => {
  if (state.sortBy === 'latest') {
    state.sortBy = 'priority';
    elSortIcon.textContent = '⚡';
    elSortText.textContent = 'PRIORITY FIRST';
  } else {
    state.sortBy = 'latest';
    elSortIcon.textContent = '⏱️';
    elSortText.textContent = 'LATEST FIRST';
  }
  updateUI();
});

// Live Search Input
elSearchInput.addEventListener('input', e => {
  state.searchQuery = e.target.value;
  updateUI();
});

// SSE Setup for Real-time push updates
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

  evtSource.onerror = () => {
    console.warn('SSE connection interrupted, falling back to polling.');
  };
}

// Global Keyboard Shortcuts
window.addEventListener('keydown', e => {
  // Ignore shortcuts when user is typing in search
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
    closeInspector();
  } else if (e.key === 'r' || e.key === 'R') {
    triggerSync();
  } else if (e.key === 'm' || e.key === 'M') {
    markAllRead();
  } else if (e.key === 'o' || e.key === 'O') {
    if (state.selectedItem && state.selectedItem.url) {
      window.open(state.selectedItem.url, '_blank');
    }
  }
});

// Initial boot
fetchFeeds();
setupSSE();
