<script>
  import { onMount } from 'svelte';
  import { schedules as schedulesApi } from '../lib/api.js';

  let { projectId } = $props();

  const QUICK_SCHEDULES = [
    { id: 'every-30m', label: 'Every 30 minutes', cronExpr: '*/30 * * * *' },
    { id: 'hourly', label: 'Every hour', cronExpr: '0 * * * *' },
    { id: 'every-3h', label: 'Every 3 hours', cronExpr: '0 */3 * * *' },
    { id: 'daily', label: 'Every day', type: 'daily', needsTime: true },
    { id: 'weekdays', label: 'Weekdays', type: 'weekdays', needsTime: true },
  ];

  let scheduleList = $state([]);
  let loading = $state(false);
  let error = $state('');

  let newPlatform = $state('reddit');
  let quickScheduleId = $state(QUICK_SCHEDULES[0].id);
  let scheduleTime = $state('09:00');
  let useCustomCron = $state(false);
  let customCron = $state('');
  let adding = $state(false);

  let activeCount = $derived(scheduleList.filter(s => s.enabled).length);
  let selectedQuickSchedule = $derived(QUICK_SCHEDULES.find(item => item.id === quickScheduleId) || QUICK_SCHEDULES[0]);
  let cronExpr = $derived(useCustomCron ? customCron.trim() : buildCronExpr(selectedQuickSchedule, scheduleTime));

  async function loadSchedules() {
    loading = true;
    error = '';
    try {
      scheduleList = await schedulesApi.list(projectId);
    } catch (e) {
      error = e.message;
    } finally {
      loading = false;
    }
  }

  async function toggleEnabled(s) {
    try {
      const updated = await schedulesApi.update(projectId, s.id, { enabled: !s.enabled });
      scheduleList = scheduleList.map(item => item.id === s.id ? { ...item, ...updated } : item);
    } catch (e) {
      error = e.message;
    }
  }

  async function deleteSchedule(s) {
    if (!confirm('Delete this schedule?')) return;
    try {
      await schedulesApi.delete(projectId, s.id);
      scheduleList = scheduleList.filter(item => item.id !== s.id);
    } catch (e) {
      error = e.message;
    }
  }

  async function addSchedule() {
    if (!cronExpr) return;
    adding = true;
    error = '';
    try {
      const created = await schedulesApi.create(projectId, {
        platform: newPlatform,
        cron_expr: cronExpr,
      });
      scheduleList = [...scheduleList, created];
      customCron = '';
      useCustomCron = false;
    } catch (e) {
      error = e.message;
    } finally {
      adding = false;
    }
  }

  function getScheduleLabel(cronValue) {
    const quickSchedule = QUICK_SCHEDULES.find(item => item.cronExpr === cronValue);
    if (quickSchedule) return quickSchedule.label;

    const parts = String(cronValue || '').trim().split(/\s+/);
    if (parts.length !== 5) return 'Custom schedule';

    const [minutePart, hourPart, dayOfMonth, month, dayOfWeek] = parts;
    if (!/^\d+$/.test(minutePart) || !/^\d+$/.test(hourPart)) return 'Custom schedule';

    const minute = Number.parseInt(minutePart, 10);
    const hour = Number.parseInt(hourPart, 10);
    if (minute < 0 || minute > 59 || hour < 0 || hour > 23) return 'Custom schedule';
    if (dayOfMonth !== '*' || month !== '*') return 'Custom schedule';

    const timeLabel = formatTime(hour, minute);
    if (dayOfWeek === '*') return `Every day at ${timeLabel}`;
    if (dayOfWeek === '1-5') return `Weekdays at ${timeLabel}`;
    return 'Custom schedule';
  }

  function parseTime(value) {
    const [hourText = '9', minuteText = '0'] = String(value || '').split(':');
    const hour = Math.min(23, Math.max(0, Number.parseInt(hourText, 10) || 0));
    const minute = Math.min(59, Math.max(0, Number.parseInt(minuteText, 10) || 0));
    return { hour, minute };
  }

  function buildCronExpr(schedule, timeValue) {
    if (!schedule) return '';
    if (schedule.cronExpr) return schedule.cronExpr;

    const { hour, minute } = parseTime(timeValue);
    if (schedule.type === 'daily') return `${minute} ${hour} * * *`;
    if (schedule.type === 'weekdays') return `${minute} ${hour} * * 1-5`;
    return '';
  }

  function formatTime(hour, minute) {
    const date = new Date();
    date.setHours(hour, minute, 0, 0);
    return date.toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' });
  }

  function relativeTime(dateStr) {
    if (!dateStr) return 'Never';
    const diff = Date.now() - new Date(dateStr).getTime();
    const mins = Math.floor(diff / 60000);
    if (mins < 1) return 'Just now';
    if (mins < 60) return `${mins}m ago`;
    const hours = Math.floor(mins / 60);
    if (hours < 24) return `${hours}h ago`;
    const days = Math.floor(hours / 24);
    return `${days}d ago`;
  }

  onMount(loadSchedules);
</script>

<div class="schedule-manager">
  <div class="section-header">
    <h3 class="section-title">Schedules</h3>
    <span class="count">{activeCount} active</span>
  </div>

  {#if error}
    <div class="error-msg">{error}</div>
  {/if}

  {#if loading}
    <div class="loading">Loading schedules...</div>
  {:else if scheduleList.length === 0}
    <div class="empty-msg">No schedules configured. Add one below.</div>
  {:else}
    <div class="schedule-list">
      {#each scheduleList as s (s.id)}
        <div class="schedule-row">
          <span
            class="platform-badge"
            class:reddit={s.platform === 'reddit'}
            class:bluesky={s.platform === 'bluesky'}
            class:google={s.platform === 'google'}
          >
            {s.platform === 'reddit' ? 'Reddit' : s.platform === 'bluesky' ? 'Bluesky' : 'Google'}
          </span>
          <div class="schedule-details">
            <span class="schedule-label">{getScheduleLabel(s.cron_expr)}</span>
            <code class="cron-expr">{s.cron_expr}</code>
          </div>
          <span class="last-run">Last: {relativeTime(s.last_run_at)}</span>
          <label class="toggle" title={s.enabled ? 'Disable' : 'Enable'}>
            <input type="checkbox" checked={s.enabled} onchange={() => toggleEnabled(s)} />
            <span class="toggle-slider"></span>
          </label>
          <button class="delete-btn" onclick={() => deleteSchedule(s)} title="Delete schedule">&#x2715;</button>
        </div>
      {/each}
    </div>
  {/if}

  <form class="add-form" onsubmit={(e) => { e.preventDefault(); addSchedule(); }}>
    <div class="add-form-title">Add Schedule</div>
    <div class="form-row">
      <select bind:value={newPlatform} class="platform-select">
        <option value="reddit">Reddit</option>
        <option value="bluesky">Bluesky</option>
        <option value="google">Google</option>
      </select>
      {#if useCustomCron}
        <input
          class="cron-input"
          type="text"
          placeholder="e.g., */30 * * * *"
          bind:value={customCron}
        />
      {:else}
        <select bind:value={quickScheduleId} class="quick-schedule-select">
          {#each QUICK_SCHEDULES as option}
            <option value={option.id}>{option.label}</option>
          {/each}
        </select>
      {/if}
    </div>

    {#if !useCustomCron && selectedQuickSchedule.needsTime}
      <div class="time-row">
        <label class="time-label" for="schedule-time">Time</label>
        <input id="schedule-time" class="time-input" type="time" step="60" bind:value={scheduleTime} />
        <span class="time-note">server local time</span>
      </div>
    {/if}

    <label class="custom-mode-toggle">
      <input type="checkbox" bind:checked={useCustomCron} />
      Use custom cron expression
    </label>

    <div class="cron-preview">
      <span class="cron-preview-label">Will run as:</span>
      <span class="cron-preview-human">{getScheduleLabel(cronExpr)}</span>
      <code>{cronExpr || '—'}</code>
    </div>
    <div class="cron-hint">
      {#if useCustomCron}
        Cron format: minute hour day-of-month month day-of-week (server local time)
      {:else if selectedQuickSchedule.needsTime}
        Pick the exact time for this schedule, then click Add Schedule.
      {:else}
        Pick a ready-made schedule to avoid cron syntax mistakes.
      {/if}
    </div>
    <button class="add-btn" type="submit" disabled={adding || !cronExpr}>
      {adding ? 'Adding...' : 'Add Schedule'}
    </button>
  </form>
</div>

<style>
  .schedule-manager {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .section-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .section-title {
    font-size: 15px;
    font-weight: 600;
    color: #e2e2e8;
  }

  .count {
    font-size: 12px;
    color: #666;
  }

  .error-msg {
    padding: 10px 14px;
    background: #3a1a1a;
    border: 1px solid #6a2a2a;
    border-radius: 6px;
    color: #f87171;
    font-size: 13px;
  }

  .loading,
  .empty-msg {
    color: #666;
    font-size: 14px;
    text-align: center;
    padding: 20px 0;
  }

  .schedule-list {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .schedule-row {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 14px;
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 6px;
  }

  .platform-badge {
    flex-shrink: 0;
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .platform-badge.reddit {
    background: #ff4500;
    color: #fff;
  }

  .platform-badge.bluesky {
    background: #0085ff;
    color: #fff;
  }

  .platform-badge.google {
    background: #34a853;
    color: #fff;
  }

  .cron-expr {
    font-family: monospace;
    font-size: 12px;
    color: #c0c0d0;
  }

  .schedule-details {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .schedule-label {
    font-size: 13px;
    color: #e2e2e8;
  }

  .last-run {
    font-size: 12px;
    color: #666;
    white-space: nowrap;
  }

  .toggle {
    flex-shrink: 0;
    position: relative;
    width: 36px;
    height: 20px;
    cursor: pointer;
  }

  .toggle input {
    opacity: 0;
    width: 0;
    height: 0;
    position: absolute;
  }

  .toggle-slider {
    position: absolute;
    inset: 0;
    background: #333;
    border-radius: 20px;
    transition: background 0.2s;
  }

  .toggle-slider::after {
    content: '';
    position: absolute;
    width: 14px;
    height: 14px;
    left: 3px;
    top: 3px;
    background: #fff;
    border-radius: 50%;
    transition: transform 0.2s;
  }

  .toggle input:checked + .toggle-slider {
    background: #7c6af5;
  }

  .toggle input:checked + .toggle-slider::after {
    transform: translateX(16px);
  }

  .delete-btn {
    flex-shrink: 0;
    width: 26px;
    height: 26px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: none;
    border: 1px solid #3a2a2a;
    border-radius: 4px;
    color: #888;
    font-size: 12px;
    cursor: pointer;
    transition: all 0.15s;
  }

  .delete-btn:hover {
    background: #3a1a1a;
    border-color: #f87171;
    color: #f87171;
  }

  .add-form {
    display: flex;
    flex-direction: column;
    gap: 10px;
    padding: 16px;
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 8px;
  }

  .add-form-title {
    font-size: 13px;
    font-weight: 600;
    color: #888;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .form-row {
    display: flex;
    gap: 10px;
  }

  .platform-select {
    padding: 7px 10px;
    background: #0f0f13;
    border: 1px solid #2a2a3a;
    border-radius: 6px;
    color: #e2e2e8;
    font-size: 13px;
    cursor: pointer;
  }

  .cron-input {
    flex: 1;
    padding: 7px 10px;
    background: #0f0f13;
    border: 1px solid #2a2a3a;
    border-radius: 6px;
    color: #e2e2e8;
    font-size: 13px;
    font-family: monospace;
  }

  .quick-schedule-select {
    flex: 1;
    padding: 7px 10px;
    background: #0f0f13;
    border: 1px solid #2a2a3a;
    border-radius: 6px;
    color: #e2e2e8;
    font-size: 13px;
    cursor: pointer;
  }

  .quick-schedule-select:focus {
    outline: none;
    border-color: #7c6af5;
  }

  .time-row {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .time-label {
    font-size: 12px;
    color: #9a9ab0;
  }

  .time-input {
    padding: 7px 10px;
    background: #0f0f13;
    border: 1px solid #2a2a3a;
    border-radius: 6px;
    color: #e2e2e8;
    font-size: 13px;
  }

  .time-input:focus {
    outline: none;
    border-color: #7c6af5;
  }

  .time-note {
    font-size: 11px;
    color: #666;
  }

  .cron-input::placeholder {
    color: #555;
    font-family: monospace;
  }

  .cron-input:focus,
  .platform-select:focus {
    outline: none;
    border-color: #7c6af5;
  }

  .custom-mode-toggle {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 12px;
    color: #9a9ab0;
  }

  .custom-mode-toggle input {
    accent-color: #7c6af5;
  }

  .cron-preview {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 12px;
    color: #8d8da3;
  }

  .cron-preview code {
    font-family: monospace;
    color: #c0c0d0;
    background: #0f0f13;
    border: 1px solid #2a2a3a;
    border-radius: 4px;
    padding: 3px 6px;
  }

  .cron-preview-label {
    color: #7b7b92;
  }

  .cron-preview-human {
    color: #cfd0de;
    font-size: 12px;
  }

  .cron-hint {
    font-size: 11px;
    color: #555;
  }

  .add-btn {
    align-self: flex-start;
    padding: 7px 16px;
    background: #7c6af5;
    border: none;
    border-radius: 6px;
    color: #fff;
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition: background 0.15s;
  }

  .add-btn:hover:not(:disabled) {
    background: #6a58e3;
  }

  .add-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
