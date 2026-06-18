"use strict";

const state = {
    apis: [],
    logs: [],
    selectedLogId: null,
    paused: false,
    stream: null,
    detailWindows: {},
};

const DETAIL_STORAGE_KEY = "phantom-detail-entry";

async function api(method, path) {
    const res = await fetch(path, { method });
    if (!res.ok) {
        let msg = res.statusText;
        try { msg = (await res.json()).error || msg; } catch (_) {}
        throw new Error(msg);
    }
    if (res.status === 204) return null;
    return res.json();
}

function toast(message, isError) {
    const el = document.getElementById("toast");
    el.textContent = message;
    el.classList.toggle("error", !!isError);
    el.classList.remove("hidden");
    clearTimeout(toast._t);
    toast._t = setTimeout(() => el.classList.add("hidden"), 2600);
}

function escapeHtml(s) {
    return String(s ?? "")
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;");
}

function formatTime(ts) {
    return new Date(ts).toLocaleString("ko-KR", { hour12: false });
}

function formatUptime(sec) {
    sec = Math.floor(sec || 0);
    const h = Math.floor(sec / 3600);
    const m = Math.floor((sec % 3600) / 60);
    const s = sec % 60;
    if (h) return `${h}h ${m}m`;
    if (m) return `${m}m ${s}s`;
    return `${s}s`;
}

function setPill(id, text, ok) {
    const el = document.getElementById(id);
    el.textContent = "";
    const dot = document.createElement("span");
    dot.className = "badge-dot";
    el.appendChild(dot);
    el.appendChild(document.createTextNode(text));
    el.classList.remove("ok", "bad");
    if (ok === true) el.classList.add("ok");
    if (ok === false) el.classList.add("bad");
}

async function refreshStatus() {
    try {
        const s = await api("GET", "/api/status");
        setPill("pill-config", "Config OK", true);
        setPill("pill-scheme", `${s.scheme}:${s.port}`, null);
        setPill("pill-apis", `APIs ${s.apiCount}`, null);
        setPill("pill-uptime", "Uptime " + formatUptime(s.uptimeSeconds), null);
    } catch (_) {
        setPill("pill-config", "Config ERR", false);
    }
}

async function loadAPIs() {
    state.apis = await api("GET", "/api/apis");
    renderAPIs();
    populatePathFilter();
}

function renderAPIs() {
    const list = document.getElementById("api-list");
    const selectedPath = document.getElementById("filter-path").value;
    list.innerHTML = "";
    for (const a of state.apis) {
        const li = document.createElement("li");
        li.className = "api-card" + (selectedPath === a.path ? " active" : "");
        const methods = (a.methods || []).map(m =>
            `<span class="chip chip-${escapeHtml(m)}">${escapeHtml(m)}</span>`).join("");
        li.innerHTML = `
            <div class="api-path">${escapeHtml(a.path)}</div>
            <div class="api-desc">${escapeHtml(a.description || "")}</div>
            <div class="api-methods">${methods}</div>`;
        li.addEventListener("click", () => {
            document.getElementById("filter-path").value = a.path;
            reloadLogs();
            renderAPIs();
        });
        list.appendChild(li);
    }
}

function populatePathFilter() {
    const sel = document.getElementById("filter-path");
    const current = sel.value;
    sel.innerHTML = '<option value="">All</option>';
    for (const a of state.apis) {
        const opt = document.createElement("option");
        opt.value = a.path;
        opt.textContent = a.path;
        sel.appendChild(opt);
    }
    sel.value = current;
}

function filterParams() {
    const params = new URLSearchParams();
    const path = document.getElementById("filter-path").value;
    const method = document.getElementById("filter-method").value;
    const q = document.getElementById("filter-text").value.trim();
    if (path) params.set("path", path);
    if (method) params.set("method", method);
    if (q) params.set("q", q);
    params.set("limit", "200");
    return params;
}

function matchesCurrentFilter(entry) {
    const path = document.getElementById("filter-path").value;
    const method = document.getElementById("filter-method").value;
    const q = document.getElementById("filter-text").value.trim().toLowerCase();
    if (path && entry.path !== path) return false;
    if (method && entry.method !== method) return false;
    if (q) {
        const blob = [
            entry.body, entry.query, entry.clientIp,
            JSON.stringify(entry.headers || {}),
        ].join(" ").toLowerCase();
        if (!blob.includes(q)) return false;
    }
    return true;
}

async function reloadLogs() {
    try {
        const params = filterParams();
        state.logs = await api("GET", "/api/logs?" + params.toString());
        renderLogs();
    } catch (e) {
        toast(e.message, true);
    }
}

function renderLogs() {
    const box = document.getElementById("logs");
    const empty = document.getElementById("logs-empty");
    if (!state.logs.length) {
        box.innerHTML = "";
        box.classList.add("hidden");
        empty.classList.remove("hidden");
        return;
    }
    empty.classList.add("hidden");
    box.classList.remove("hidden");
    box.innerHTML = state.logs.map(entry => logRowHtml(entry)).join("");
    box.querySelectorAll(".log-item").forEach(row => {
        row.addEventListener("click", () => selectLog(Number(row.dataset.id)));
    });
}

function logRowHtml(entry) {
    const preview = (entry.body || entry.query || "(empty)").slice(0, 120);
    const selected = entry.id === state.selectedLogId ? " selected" : "";
    const methodClass = "method-" + escapeHtml(entry.method);
    return `<article class="log-item ${methodClass}${selected}" data-id="${entry.id}">
        <div class="log-item-head">
            <span class="log-time">${escapeHtml(formatTime(entry.time))}</span>
            <span class="chip chip-${escapeHtml(entry.method)}">${escapeHtml(entry.method)}</span>
            <span class="log-path">${escapeHtml(entry.path)}</span>
        </div>
        <div class="log-meta">
            <span>${escapeHtml(entry.clientIp)}</span>
            <span>·</span>
            <span>${entry.bodySize} bytes</span>
        </div>
        <div class="log-preview">${escapeHtml(preview)}</div>
    </article>`;
}

function selectLog(id) {
    state.selectedLogId = id;
    const entry = state.logs.find(e => e.id === id);
    renderLogs();
    openDetailPopup(entry);
}

function openDetailPopup(entry) {
    if (!entry) return;

    sessionStorage.setItem(DETAIL_STORAGE_KEY, JSON.stringify(entry));

    const winName = "phantom-detail-" + entry.id;
    let win = state.detailWindows[entry.id];

    if (win && !win.closed) {
        win.focus();
        win.postMessage({ type: "phantom-detail-update", entry }, window.location.origin);
        return;
    }

    const features = [
        "popup=yes",
        "width=620",
        "height=760",
        "resizable=yes",
        "scrollbars=yes",
        "menubar=no",
        "toolbar=no",
        "location=no",
        "status=no",
    ].join(",");

    win = window.open("/static/detail.html?id=" + entry.id, winName, features);
    if (!win) {
        toast("팝업이 차단되었습니다. 브라우저에서 팝업을 허용해 주세요.", true);
        return;
    }
    state.detailWindows[entry.id] = win;
}

function prependLog(entry) {
    if (!matchesCurrentFilter(entry)) return;
    state.logs.unshift(entry);
    if (state.logs.length > 200) state.logs.pop();
    const box = document.getElementById("logs");
    document.getElementById("logs-empty").classList.add("hidden");
    box.classList.remove("hidden");
    const div = document.createElement("div");
    div.innerHTML = logRowHtml(entry);
    const row = div.firstElementChild;
    row.addEventListener("click", () => selectLog(entry.id));
    box.insertBefore(row, box.firstChild);
}

function connectStream() {
    if (state.stream) {
        state.stream.close();
        state.stream = null;
    }
    if (state.paused) return;

    const params = filterParams();
    params.delete("limit");
    const url = "/api/logs/stream?" + params.toString();
    const es = new EventSource(url);
    state.stream = es;

    es.addEventListener("log", (ev) => {
        try {
            const entry = JSON.parse(ev.data);
            prependLog(entry);
        } catch (_) {}
    });

    es.onerror = () => {
        es.close();
        state.stream = null;
        if (!state.paused) {
            setTimeout(connectStream, 2000);
        }
    };
}

function clearFilter() {
    document.getElementById("filter-path").value = "";
    document.getElementById("filter-method").value = "";
    document.getElementById("filter-text").value = "";
    reloadLogs();
    renderAPIs();
    connectStream();
}

function togglePause() {
    state.paused = !state.paused;
    const btn = document.getElementById("btn-pause");
    btn.textContent = state.paused ? "Resume" : "Pause";
    btn.classList.toggle("active", state.paused);
    if (state.paused && state.stream) {
        state.stream.close();
        state.stream = null;
    } else {
        connectStream();
    }
}

let filterDebounce;
function onFilterChange() {
    clearTimeout(filterDebounce);
    filterDebounce = setTimeout(() => {
        reloadLogs();
        renderAPIs();
        connectStream();
    }, 300);
}

async function init() {
    document.getElementById("btn-clear-filter").addEventListener("click", clearFilter);
    document.getElementById("btn-pause").addEventListener("click", togglePause);
    document.getElementById("filter-path").addEventListener("change", onFilterChange);
    document.getElementById("filter-method").addEventListener("change", onFilterChange);
    document.getElementById("filter-text").addEventListener("input", onFilterChange);

    try {
        await loadAPIs();
        await reloadLogs();
        connectStream();
    } catch (e) {
        toast(e.message, true);
    }

    refreshStatus();
    setInterval(refreshStatus, 5000);
}

init();
