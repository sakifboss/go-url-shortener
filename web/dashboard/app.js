const state = { token: sessionStorage.getItem('goshort_access_token') || '' };

const $ = (selector) => document.querySelector(selector);
const message = $('#message');

function setMessage(text, isError = true) {
    message.textContent = text;
    message.style.color = isError ? 'var(--coral)' : 'var(--teal)';
}

function setSignedIn(signedIn) {
    $('#login-form').classList.toggle('is-hidden', signedIn);
    $('#analytics-controls').classList.toggle('is-hidden', !signedIn);
    $('#create-url-form').classList.toggle('is-hidden', !signedIn);
    $('#dashboard').classList.toggle('is-hidden', !signedIn);
}

function renderRankList(target, items, emptyLabel) {
    const element = $(target);
    element.textContent = '';
    if (!items.length) {
        element.innerHTML = `<p class="empty-state">${emptyLabel}</p>`;
        return;
    }

    const maximum = Math.max(...items.map((item) => item.count), 1);
    items.forEach((item) => {
        const row = document.createElement('div');
        row.className = 'rank-row';
        row.innerHTML = `<span>${escapeHTML(item.name)}</span><strong>${item.count}</strong><div class="rank-track"><span style="width:${(item.count / maximum) * 100}%"></span></div>`;
        element.appendChild(row);
    });
}

function renderDailyChart(items) {
    const chart = $('#daily-chart');
    chart.textContent = '';
    if (!items.length) {
        chart.innerHTML = '<p class="empty-state">No visits recorded yet.</p>';
        $('#date-range').textContent = '';
        return;
    }

    const maximum = Math.max(...items.map((item) => item.count), 1);
    $('#date-range').textContent = `${items[0].date} - ${items[items.length - 1].date}`;
    items.slice(-14).forEach((item) => {
        const column = document.createElement('div');
        column.className = 'bar-item';
        column.innerHTML = `<span class="bar-value">${item.count}</span><div class="bar" style="height:${Math.max((item.count / maximum) * 82, 5)}%" title="${item.count} clicks"></div><span class="bar-label">${item.date.slice(5)}</span>`;
        chart.appendChild(column);
    });
}

function renderAnalytics(data) {
    $('#total-clicks').textContent = data.total_clicks;
    const topDevice = data.by_device[0];
    const topReferrer = data.by_referrer[0];
    $('#top-device').textContent = topDevice?.name || '-';
    $('#top-device-count').textContent = topDevice ? `${topDevice.count} clicks` : 'No visits yet';
    $('#top-referrer').textContent = topReferrer?.name || '-';
    $('#top-referrer-count').textContent = topReferrer ? `${topReferrer.count} clicks` : 'No visits yet';
    renderRankList('#device-list', data.by_device, 'No device data yet.');
    renderRankList('#referrer-list', data.by_referrer, 'No referrer data yet.');
    renderDailyChart(data.daily_clicks);
}

async function requestJSON(url, options = {}) {
    const response = await fetch(url, {
        ...options,
        headers: { 'Content-Type': 'application/json', ...(options.headers || {}) },
    });
    if (!response.ok) {
        const text = await response.text();
        throw new Error(text || `Request failed (${response.status})`);
    }
    return response.json();
}

async function signIn(event) {
    event.preventDefault();
    setMessage('Signing in...', false);
    try {
        const data = await requestJSON('/api/v1/auth/login', {
            method: 'POST',
            body: JSON.stringify({ email: $('#email').value, password: $('#password').value }),
        });
        state.token = data.access_token;
        sessionStorage.setItem('goshort_access_token', state.token);
        setSignedIn(true);
        setMessage('Signed in. Enter a URL ID to load analytics.', false);
    } catch (error) {
        setMessage(error.message);
    }
}

async function loadAnalytics() {
    const urlID = Number($('#url-id').value);
    if (!Number.isInteger(urlID) || urlID < 1) {
        setMessage('Enter a valid URL ID.');
        return;
    }
    setMessage('Loading analytics...', false);
    try {
        const data = await requestJSON(`/api/v1/urls/${urlID}/analytics`, {
            headers: { Authorization: `Bearer ${state.token}` },
        });
        renderAnalytics(data);
        setMessage(`Analytics loaded for URL #${urlID}.`, false);
    } catch (error) {
        setMessage(error.message);
    }
}

async function createShortURL(event) {
    event.preventDefault();
    const url = $('#long-url').value.trim();
    const alias = $('#custom-alias').value.trim();
    const requestBody = { url };
    if (alias) {
        requestBody.alias = alias;
    }

    setMessage('Creating short link...', false);
    $('#created-link').classList.add('is-hidden');
    try {
        const data = await requestJSON('/api/v1/urls', {
            method: 'POST',
            headers: {
                Authorization: `Bearer ${state.token}`,
                'Idempotency-Key': `dashboard-${Date.now()}-${Math.random().toString(36).slice(2)}`,
            },
            body: JSON.stringify(requestBody),
        });
        $('#created-link-url').href = data.short_url;
        $('#created-link-url').textContent = data.short_url;
        $('#created-link').classList.remove('is-hidden');
        $('#url-id').value = data.id;
        setMessage('Short link created. You can load its analytics below.', false);
    } catch (error) {
        setMessage(error.message);
    }
}

async function copyLink() {
    const link = $('#created-link-url').href;
    try {
        await navigator.clipboard.writeText(link);
        setMessage('Short link copied.', false);
    } catch {
        setMessage('Copy failed. Select the link and copy it manually.');
    }
}

function signOut() {
    state.token = '';
    sessionStorage.removeItem('goshort_access_token');
    setSignedIn(false);
    setMessage('Signed out.', false);
}

function escapeHTML(value) {
    return String(value).replace(/[&<>'"]/g, (character) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', "'": '&#39;', '"': '&quot;' }[character]));
}

$('#login-form').addEventListener('submit', signIn);
$('#create-url-form').addEventListener('submit', createShortURL);
$('#load-analytics').addEventListener('click', loadAnalytics);
$('#sign-out').addEventListener('click', signOut);
$('#copy-link').addEventListener('click', copyLink);
setSignedIn(Boolean(state.token));
