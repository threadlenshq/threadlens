<script>
  import { onMount, onDestroy } from 'svelte';
  import ProjectSettings from './components/ProjectSettings.svelte';
  import QueryEditor from './components/QueryEditor.svelte';
  import ScoutRunButton from './components/ScoutRunButton.svelte';
  import ActiveRunBanner from './components/ActiveRunBanner.svelte';
  import TopbarJobBanner from './components/TopbarJobBanner.svelte';
  import FindingCard from './components/inbox/FindingCard.svelte';
  import DetailPanel from './components/DetailPanel.svelte';
  import { projects as projectsApi, posts as postsApi, scout as scoutApi, queries as queriesApi, queryReviewJobs as queryReviewJobsApi, runtime as runtimeApi, onboarding as onboardingApi, google as googleApi, filters as filtersApi } from './lib/api.js';
  import { FILTERED_FINDINGS_LABEL, REFILTER_SELECTED_LABEL } from './lib/filterLabels.js';
  import FilteredFindingsTab from './components/FilteredFindingsTab.svelte';
  import { readUrlState, writeUrlState, clearUrlState } from './lib/url.js';
  import OnboardingWizard from './components/OnboardingWizard.svelte';
  import ExplorationChecklist from './components/onboarding/ExplorationChecklist.svelte';
  import TourCallout from './components/onboarding/TourCallout.svelte';
  import { normalizeOnboardingStatus, shouldShowRequiredWizard, shouldShowExploration } from './lib/onboardingState.js';
  import { parseApiTimestamp } from './lib/format.js';
  import TelemetryConsentToast from './components/TelemetryConsentToast.svelte';
  import PrivacySettings from './components/PrivacySettings.svelte';
  import { telemetry as telemetryApi } from './lib/api.js';
  import { initTelemetry } from './lib/telemetry.js';
  import AppShell from './components/layout/AppShell.svelte';
  import WorkspaceRail from './components/layout/WorkspaceRail.svelte';
  import TopContextBar from './components/layout/TopContextBar.svelte';
  import ReportsTab from './components/ReportsTab.svelte';
  import ReportView from './components/ReportView.svelte';
  import GoogleReportsTab from './components/GoogleReportsTab.svelte';
  import GoogleReportView from './components/GoogleReportView.svelte';
  import ModelConfig from './components/ModelConfig.svelte';
  import ManualTasker from './components/ManualTasker.svelte';
  import NewProjectModal from './components/NewProjectModal.svelte';
  import GoogleLockedNotice from './components/GoogleLockedNotice.svelte';
  import { isGoogleScoutLocked } from './lib/capabilities.js';
  import { POST_STATUSES } from '@scout/shared';

  // --- Onboarding gate ---
  let onboardingStatus = $state(null);
  let onboardingChecked = $state(false);
  let onboardingError = $state('');
  let showTourCallout = $state(true);
  let checklistOpen = $state(false);
  let onboardingRequiresSetup = $derived(shouldShowRequiredWizard(onboardingStatus));
  let onboardingShowsExploration = $derived(shouldShowExploration(onboardingStatus));

  // --- New project modal ---
  let showNewProjectModal = $state(false);

  async function checkOnboarding() {
    onboardingError = '';
    try {
      const data = await onboardingApi.status();
      onboardingStatus = normalizeOnboardingStatus(data);
    } catch (e) {
      onboardingError = e.message || 'Failed to load onboarding status.';
      onboardingStatus = null;
    }
    onboardingChecked = true;
  }

  // --- State ---
  let projectList = $state([]);
  let selectedProjectId = $state(null);
  let view = $state('posts'); // 'posts' | 'settings' | 'sources' | 'reports' | 'models'
  let settingsTab = $state('general');
  let activeReportId = $state(null);
  let activeGoogleReportId = $state(null);
  let reportSource = $state('social'); // 'social' | 'google'
  let capabilitySnapshot = $state(null);
  let googleLocked = $derived(isGoogleScoutLocked(capabilitySnapshot));

  // Google report presence state
  let showGoogleLockedNotice = $state(false);
  let googleReportCount = $state(0);

  // Posts state
  let postsList = $state([]);
  let selectedPost = $state(null);
  let loadingProjects = $state(false);
  let loadingPosts = $state(false);
  let postsPage = $state(1);
  let postsPageLimit = $state(20);
  let postsPagination = $state({
    page: 1,
    limit: 20,
    total: 0,
    totalPages: 1,
    hasPreviousPage: false,
    hasNextPage: false,
  });

  // DetailPanel state
  let generating = $state(false);
  let generateError = $state(null);
  let postingReply = $state(false);
  let postReplyResult = $state(null);

  // Query readiness state
  let enabledQueryCount = $state(null);

  // Run-awareness state
  let activeRuns = $state([]);
  let activeRunIds = $state(new Set());
  let failedRuns = $state([]);
  let completedRuns = $state([]);
  let lastCompletedRun = $state(null);
  let pollTimer = $state(null);
  let timeTick = $state(0);
  let tickTimer = setInterval(() => { timeTick++; }, 60000);

  // Query-review job state
  let queryReviewJobs = $state([]);
  let queryJobPollTimer = $state(null);
  let activeQueryReviewJobId = $state(null);

  let runningQueryJobs = $derived(queryReviewJobs.filter(j => j.status === 'running' || j.status === 'pending'));
  let completedQueryJobs = $derived(queryReviewJobs.filter(j => j.status === 'completed'));
  let failedQueryJobs = $derived(queryReviewJobs.filter(j => j.status === 'failed'));
  let activeQueryReviewJob = $derived(activeQueryReviewJobId ? queryReviewJobs.find(j => j.id === activeQueryReviewJobId) || null : null);

  // Filter job state
  let filterJobs = $state([]);
  let filterJobPollTimer = $state(null);
  const DISMISSED_FILTER_JOB_IDS_STORAGE_KEY = 'scout:dismissed-filter-job-ids';
  const FILTER_JOB_AUTO_DISMISS_MS = 8000;
  let dismissedFilterJobIds = $state(new Set());
  let filterJobDismissTimers = new Map();

  function filterJobKey(id) {
    if (id == null) return '';
    return String(id);
  }

  let runningFilterJobs = $derived(filterJobs.filter(j => j.status === 'running'));
  let completedFilterJobs = $derived(filterJobs.filter(j => j.status === 'completed' && !dismissedFilterJobIds.has(filterJobKey(j.id))));
  let failedFilterJobs = $derived(filterJobs.filter(j => j.status === 'failed' && !dismissedFilterJobIds.has(filterJobKey(j.id))));

  function formatFilterJobResult(result) {
    if (!result) return 'Complete';
    return `Filtered ${result.filtered ?? 0} · Restored ${result.restored_by_trust ?? 0} · Unchanged ${result.unchanged ?? 0} · Failed ${result.failed ?? 0}`;
  }

  let normalizedRunningFilterJobs = $derived(runningFilterJobs.map(j => ({ ...j, id: `filter-${j.id}`, label: 'Filtering', step: j.step || 'Running...' })));
  let normalizedCompletedFilterJobs = $derived(completedFilterJobs.map(j => ({ ...j, rawId: j.id, id: `filter-${j.id}`, label: 'Filtering', step: formatFilterJobResult(j.result), hideReview: true, dismissible: true })));
  let normalizedFailedFilterJobs = $derived(failedFilterJobs.map(j => ({ ...j, rawId: j.id, id: `filter-${j.id}`, label: 'Filtering', step: j.error || 'Job failed', hideReview: true, dismissible: true })));

  function readDismissedFilterJobIds() {
    if (typeof window === 'undefined') return new Set();
    try {
      const raw = localStorage.getItem(DISMISSED_FILTER_JOB_IDS_STORAGE_KEY);
      if (!raw) return new Set();
      const parsed = JSON.parse(raw);
      if (!Array.isArray(parsed)) return new Set();
      return new Set(
        parsed
          .filter(id => (typeof id === 'string' && id.length > 0) || typeof id === 'number')
          .map(id => String(id))
      );
    } catch {
      return new Set();
    }
  }

  function persistDismissedFilterJobIds() {
    if (typeof window === 'undefined') return;
    try {
      localStorage.setItem(DISMISSED_FILTER_JOB_IDS_STORAGE_KEY, JSON.stringify([...dismissedFilterJobIds]));
    } catch {
      // Ignore storage errors (private mode, quota, etc.)
    }
  }

  function clearFilterJobDismissTimer(id) {
    const key = filterJobKey(id);
    const timer = filterJobDismissTimers.get(key);
    if (timer) {
      clearTimeout(timer);
      filterJobDismissTimers.delete(key);
    }
  }

  function clearAllFilterJobDismissTimers() {
    for (const timer of filterJobDismissTimers.values()) {
      clearTimeout(timer);
    }
    filterJobDismissTimers.clear();
  }

  function dismissFilterJobById(id) {
    const key = filterJobKey(id);
    if (!key) return;
    dismissedFilterJobIds.add(key);
    dismissedFilterJobIds = new Set(dismissedFilterJobIds);
    persistDismissedFilterJobIds();
    clearFilterJobDismissTimer(key);
  }

  function scheduleAutoDismissFilterJob(id) {
    const key = filterJobKey(id);
    if (!key || dismissedFilterJobIds.has(key)) return;
    clearFilterJobDismissTimer(key);
    const timer = setTimeout(() => {
      dismissFilterJobById(key);
      filterJobDismissTimers.delete(key);
    }, FILTER_JOB_AUTO_DISMISS_MS);
    filterJobDismissTimers.set(key, timer);
  }

  function dismissCompletedFilterJob(job) {
    const id = job?.rawId || job?.id;
    dismissFilterJobById(id);
  }

  function dismissFailedFilterJob(job) {
    const id = job?.rawId || job?.id;
    dismissFilterJobById(id);
  }

  async function fetchFilterJobs() {
    if (!selectedProjectId) {
      filterJobs = [];
      return;
    }
    try {
      const jobs = await filtersApi.jobs(selectedProjectId);
      filterJobs = Array.isArray(jobs) ? jobs.filter(j => j.status === 'running') : [];
      // Resume polling if any jobs are still running
      const hasRunning = filterJobs.some(j => j.status === 'running');
      if (hasRunning) {
        scheduleFilterJobPoll();
      }
    } catch (e) {
      console.error('Failed to fetch filter jobs:', e);
    }
  }

  async function pollFilterJobs() {
    if (!selectedProjectId) return;
    try {
      const jobs = await filtersApi.jobs(selectedProjectId);
      const prevRunning = filterJobs.filter(j => j.status === 'running').map(j => j.id);
      const previouslyVisibleTerminalIds = new Set(
        filterJobs
          .filter(j => j.status === 'completed' || j.status === 'failed')
          .map(j => filterJobKey(j.id))
      );

      const allJobs = Array.isArray(jobs) ? jobs : [];
      const newlyFinished = allJobs.filter(
        j => (j.status === 'completed' || j.status === 'failed') && prevRunning.includes(j.id)
      );
      const newlyFinishedIds = new Set(newlyFinished.map(j => filterJobKey(j.id)));

      filterJobs = allJobs.filter(j =>
        j.status === 'running' ||
        ((j.status === 'completed' || j.status === 'failed') &&
          (previouslyVisibleTerminalIds.has(filterJobKey(j.id)) || newlyFinishedIds.has(filterJobKey(j.id))))
      );

      const nowCompleted = newlyFinished.filter(j => j.status === 'completed');
      for (const job of newlyFinished) {
        scheduleAutoDismissFilterJob(job.id);
      }
      if (nowCompleted.length > 0) {
        await loadPosts();
      }
      const stillRunning = filterJobs.some(j => j.status === 'running');
      if (stillRunning) {
        scheduleFilterJobPoll();
      } else {
        filterJobPollTimer = null;
      }
    } catch (e) {
      console.error('Failed to poll filter jobs:', e);
    }
  }

  function scheduleFilterJobPoll() {
    clearTimeout(filterJobPollTimer);
    filterJobPollTimer = setTimeout(pollFilterJobs, 3000);
  }

  function stopFilterJobPoll() {
    clearTimeout(filterJobPollTimer);
    filterJobPollTimer = null;
  }

  function relativeTime(dateStr) {
    if (!dateStr) return '';
    const startMs = parseApiTimestamp(dateStr);
    if (Number.isNaN(startMs)) return '';
    const seconds = Math.floor((Date.now() - startMs) / 1000);
    if (seconds < 60) return 'just now';
    const minutes = Math.floor(seconds / 60);
    if (minutes < 60) return `${minutes}m ago`;
    const hours = Math.floor(minutes / 60);
    if (hours < 24) return `${hours}h ago`;
    return `${Math.floor(hours / 24)}d ago`;
  }

  let lastRunLabel = $derived((timeTick, relativeTime(lastCompletedRun?.completed_at)));

  async function loadCapabilities() {
    try {
      capabilitySnapshot = await runtimeApi.capabilities();
    } catch {
      // advisory only - ScoutRunButton falls back to allow-all when null
    }
  }

  async function pollActiveRuns() {
    if (!selectedProjectId || activeRunIds.size === 0) return;
    try {
      const results = await Promise.all(
        [...activeRunIds].map(id => scoutApi.getRun(selectedProjectId, id).catch(() => null))
      );

      const stillRunning = [];
      let anyCompleted = false;

      for (const run of results) {
        if (!run) continue;
        if (run.status === 'running') {
          stillRunning.push(run);
        } else {
          activeRunIds.delete(run.id);
          if (run.status === 'completed') {
            anyCompleted = true;
            completedRuns = [...completedRuns, run];
            if (!lastCompletedRun || run.completed_at > lastCompletedRun.completed_at) {
              lastCompletedRun = run;
            }
          } else if (run.status === 'failed') {
            addFailedRun(run);
          }
        }
      }

      activeRuns = stillRunning;

      if (anyCompleted) {
        await loadPosts();
      }

      if (activeRunIds.size > 0) {
        schedulePoll();
      }
    } catch (e) {
      console.error('Failed to poll runs:', e);
    }
  }

  function schedulePoll() {
    clearTimeout(pollTimer);
    pollTimer = setTimeout(pollActiveRuns, 5000);
  }

  function stopPoll() {
    clearTimeout(pollTimer);
    pollTimer = null;
    activeRunIds = new Set();
  }

  function addFailedRun(run) {
    failedRuns = [...failedRuns, run];
  }

  function dismissFailedRun(runId) {
    failedRuns = failedRuns.filter(r => r.id !== runId);
  }

  function dismissCompletedRun(runId) {
    completedRuns = completedRuns.filter(r => r.id !== runId);
  }

  async function cancelRun(runId) {
    try {
      await scoutApi.cancelRun(selectedProjectId, runId);
    } catch (err) {
      console.error('Failed to cancel run:', err);
    }
  }

  async function fetchInitialRunState() {
    if (!selectedProjectId) return;
    try {
      const runs = await scoutApi.runs(selectedProjectId);
      const running = runs.filter(r => r.status === 'running');
      activeRuns = running;
      activeRunIds = new Set(running.map(r => r.id));

      const completed = runs.filter(r => r.status === 'completed');
      lastCompletedRun = completed[0] || null;

      if (activeRunIds.size > 0) {
        schedulePoll();
      }
    } catch (e) {
      console.error('Failed to check runs:', e);
    }
  }

  async function handleScoutTriggered(detail = {}) {
    failedRuns = [];
    completedRuns = [];
    const runIds = detail?.runIds;
    if (runIds && runIds.length > 0) {
      for (const id of runIds) activeRunIds.add(id);
      const results = await Promise.all(
        runIds.map(id => scoutApi.getRun(selectedProjectId, id).catch(() => null))
      );
      const running = [];
      for (const run of results) {
        if (!run) continue;
        if (run.status === 'running') {
          running.push(run);
        } else {
          activeRunIds.delete(run.id);
          if (run.status === 'failed') addFailedRun(run);
        }
      }
      activeRuns = [...activeRuns, ...running];
      if (activeRunIds.size > 0) schedulePoll();
    } else {
      await fetchInitialRunState();
    }
  }

  async function fetchQueryReviewJobs() {
    if (!selectedProjectId) {
      queryReviewJobs = [];
      return;
    }
    try {
      const jobs = await queryReviewJobsApi.list(selectedProjectId);
      // Keep only active/recent jobs: running, pending, completed, failed (not reviewed/archived)
      queryReviewJobs = Array.isArray(jobs) ? jobs.filter(j =>
        j.status === 'running' || j.status === 'pending' || j.status === 'completed' || j.status === 'failed'
      ) : [];
    } catch (e) {
      console.error('Failed to fetch query review jobs:', e);
    }
  }

  async function pollQueryReviewJobs() {
    if (!selectedProjectId) return;
    try {
      const jobs = await queryReviewJobsApi.list(selectedProjectId);
      queryReviewJobs = Array.isArray(jobs) ? jobs.filter(j =>
        j.status === 'running' || j.status === 'pending' || j.status === 'completed' || j.status === 'failed'
      ) : [];
      const stillRunning = queryReviewJobs.some(j => j.status === 'running' || j.status === 'pending');
      if (stillRunning) {
        scheduleQueryJobPoll();
      } else {
        queryJobPollTimer = null;
      }
    } catch (e) {
      console.error('Failed to poll query review jobs:', e);
    }
  }

  function scheduleQueryJobPoll() {
    clearTimeout(queryJobPollTimer);
    queryJobPollTimer = setTimeout(pollQueryReviewJobs, 4000);
  }

  function stopQueryJobPoll() {
    clearTimeout(queryJobPollTimer);
    queryJobPollTimer = null;
  }

  function handleQueryReviewJobStarted(job) {
    // Add or update job in list
    const idx = queryReviewJobs.findIndex(j => j.id === job.id);
    if (idx >= 0) {
      queryReviewJobs = queryReviewJobs.map((j, i) => i === idx ? job : j);
    } else {
      queryReviewJobs = [...queryReviewJobs, job];
    }
    scheduleQueryJobPoll();
  }

  function handleQueryReviewJobHandled(jobId) {
    // Remove job from visible list after user has reviewed/handled it
    queryReviewJobs = queryReviewJobs.filter(j => j.id !== jobId);
    if (activeQueryReviewJobId === jobId) {
      activeQueryReviewJobId = null;
    }
  }

  function closeQueryReviewModal() {
    activeQueryReviewJobId = null;
  }

  function reviewQueryJob(job) {
    activeQueryReviewJobId = job.id;
    view = 'sources';
    writeUrlState({ view: 'sources' }, 'push');
  }

  function handlePopState() {
    if (window.__scoutPromptSuggesting) {
      if (!confirm('AI is generating prompt suggestions. Leave anyway?')) {
        writeUrlState({
          view,
          project: selectedProjectId,
          reportSource,
          report: activeReportId,
          greport: activeGoogleReportId,
          platform: filterPlatform,
          status: filterStatus,
          dm: filterDm,
          score: filterScore,
          max_age: filterMaxAge,
          page: postsPage,
          limit: postsPageLimit,
          post: selectedPost?.id || null,
        }, 'push');
        return;
      }
    }
    const urlState = readUrlState();
    view = urlState.view;
    settingsTab = urlState.tab;
    // Normalize legacy tab=queries URLs (queries now live under Sources)
    if (view === 'settings' && settingsTab === 'queries') {
      settingsTab = 'general';
    }
    reportSource = urlState.reportSource;
    activeReportId = urlState.report;
    activeGoogleReportId = urlState.greport;
    filterPlatform = urlState.platform;
    filterStatus = urlState.status;
    filterDm = urlState.dm;
    filterScore = urlState.score;
    filterMaxAge = urlState.max_age;
    postsPage = urlState.page;
    postsPageLimit = urlState.limit;

    if (urlState.project && urlState.project !== selectedProjectId) {
      stopPoll();
      stopQueryJobPoll();
      stopFilterJobPoll();
      clearAllFilterJobDismissTimers();
      filterJobs = [];
      selectedProjectId = urlState.project;
      selectedPost = null;
      loadPosts().then(() => {
        if (urlState.post) {
          const found = postsList.find(p => p.id === urlState.post);
          if (found) selectedPost = found;
        }
      });
      fetchInitialRunState();
      loadEnabledQueryCount();
      loadGoogleReportPresence();
      fetchQueryReviewJobs();
      fetchFilterJobs();
    } else {
      loadPosts().then(() => {
        if (urlState.post) {
          const found = postsList.find(p => p.id === urlState.post);
          if (found) selectedPost = found;
          else selectedPost = null;
        } else {
          selectedPost = null;
        }
      });
    }
  }

  onDestroy(() => {
    window.removeEventListener('popstate', handlePopState);
    stopPoll();
    stopQueryJobPoll();
    stopFilterJobPoll();
    clearAllFilterJobDismissTimers();
    clearInterval(tickTimer);
  });

  // Filters
  let filterPlatform = $state('all'); // 'all' | 'reddit' | 'bluesky'
  let filterStatus = $state('new');   // 'new' | 'drafted' | 'commented' | 'all'
  let filterDm = $state(false);
  let filterScore = $state('');        // '' | 'lt3' | '3' | '5' | '7' | '9'
  let filterMaxAge = $state('');       // '' | '1' | '3' | '7' | '30'

  // Bulk selection state
  let selectedIds = $state(new Set());
  let bulkMode = $derived(selectedIds.size > 0);
  let bulkStatusBreakdown = $derived(Object.entries(
    postsList.filter(p => selectedIds.has(p.id)).reduce((acc, p) => {
      acc[p.status] = (acc[p.status] || 0) + 1;
      return acc;
    }, {})
  ));

  const RAIL_COLLAPSED_STORAGE_KEY = 'scout:rail-collapsed';

  function loadRailCollapsedPreference() {
    if (typeof window === 'undefined') return false;
    return window.localStorage.getItem(RAIL_COLLAPSED_STORAGE_KEY) === '1';
  }

  function setRailCollapsedPreference(nextCollapsed) {
    railCollapsed = nextCollapsed;
    if (typeof window !== 'undefined') {
      window.localStorage.setItem(RAIL_COLLAPSED_STORAGE_KEY, nextCollapsed ? '1' : '0');
    }
  }

  function toggleRailCollapsed() {
    setRailCollapsedPreference(!railCollapsed);
  }

  let railCollapsed = $state(loadRailCollapsedPreference());

  // --- Derived ---
  let currentProject = $derived(projectList.find(p => p.id === selectedProjectId) || null);
  let showInsights = $derived(currentProject && currentProject.mode === 'research');
  let projectMode = $derived(currentProject?.mode || 'marketing');

  // --- Data loading ---
  async function loadProjects() {
    loadingProjects = true;
    try {
      projectList = await projectsApi.list();
    } catch (e) {
      console.error('Failed to load projects:', e);
    } finally {
      loadingProjects = false;
    }
  }

  async function loadPosts() {
    if (!selectedProjectId) {
      postsList = [];
      postsPagination = {
        page: postsPage,
        limit: postsPageLimit,
        total: 0,
        totalPages: 1,
        hasPreviousPage: false,
        hasNextPage: false,
      };
      return;
    }
    loadingPosts = true;
    try {
      const params = {};
      if (filterPlatform !== 'all') params.platform = filterPlatform;
      if (filterStatus !== 'all') params.status = filterStatus;
      if (filterDm) params.dm = 'true';
      if (filterScore === 'lt3') {
        params.max_score = '3';
      } else if (filterScore) {
        params.min_score = filterScore;
      }
      if (filterMaxAge) {
        params.max_age = filterMaxAge;
      }
      params.page = String(postsPage);
      params.limit = String(postsPageLimit);
      const response = await postsApi.listPage(selectedProjectId, params);
      postsList = response.items;
      postsPagination = response.pagination;
      // Sync selectedPost with updated data if still selected
      if (selectedPost) {
        const refreshed = postsList.find(p => p.id === selectedPost.id);
        if (refreshed) selectedPost = refreshed;
        else {
          selectedPost = null;
          writeUrlState({ post: null });
        }
      }
    } catch (e) {
      console.error('Failed to load posts:', e);
    } finally {
      loadingPosts = false;
    }
  }

  // --- Project selection ---
  async function selectProject(id) {
    stopPoll();
    stopQueryJobPoll();
    stopFilterJobPoll();
    clearAllFilterJobDismissTimers();
    filterJobs = [];
    failedRuns = [];
    completedRuns = [];
    selectedProjectId = id;
    selectedPost = null;
    selectedIds = new Set();
    postsPage = 1;
    writeUrlState({ project: id, view: 'posts', post: null, tab: 'general', reportSource: 'social', report: null, greport: null, page: postsPage, limit: postsPageLimit }, 'push');
    view = 'posts';
    activeReportId = null;
    activeGoogleReportId = null;
    reportSource = 'social';
    await loadPosts();
    await fetchInitialRunState();
    await loadEnabledQueryCount();
    await fetchQueryReviewJobs();
    await fetchFilterJobs();
  }

  async function loadEnabledQueryCount() {
    if (!selectedProjectId) {
      enabledQueryCount = null;
      return;
    }
    try {
      const list = await queriesApi.list(selectedProjectId);
      enabledQueryCount = list.filter(q => q.enabled).length;
    } catch {
      enabledQueryCount = null;
    }
  }

  async function loadGoogleReportPresence() {
    if (!selectedProjectId) {
      googleReportCount = 0;
      return;
    }
    const pid = selectedProjectId;
    try {
      const reports = await googleApi.reports(pid);
      if (selectedProjectId === pid) {
        googleReportCount = Array.isArray(reports) ? reports.length : 0;
      }
    } catch {
      if (selectedProjectId === pid) {
        googleReportCount = 0;
      }
    }
  }

  async function handleProjectSelect(id) {
    await selectProject(id);
  }

  async function handleProjectCreate(detail) {
    const { id, name, description, mode } = detail;
    const created = await projectsApi.create({ id, name, mode, ...(description ? { description } : {}) });
    await loadProjects();

    if (onboardingShowsExploration) {
      const starterProjectItem = onboardingStatus?.items?.find(i => i.id === 'starter_project');
      if (starterProjectItem && starterProjectItem.state === 'pending') {
        try {
          await onboardingApi.starterProject({
            projectId: created.id,
            projectName: created.name,
            query: 'starter query',
            platform: 'reddit',
            description: created.description || '',
          });
          await checkOnboarding();
          checklistOpen = true;
        } catch (e) {
          console.error('Failed to link project to onboarding context:', e);
        }
      }
    }

    await selectProject(created.id);
  }

  function openNewProjectModal() {
    showNewProjectModal = true;
  }

  // --- Project settings handlers ---
  async function handleProjectUpdated(updated) {
    projectList = projectList.map(p => p.id === updated.id ? { ...p, ...updated } : p);
  }

  async function handleQueriesChanged(detail = {}) {
    if (detail?.projectId !== selectedProjectId) return;
    await loadEnabledQueryCount();
  }

  function selectReportSource(source) {
    if (source === 'google' && googleLocked) {
      showGoogleLockedNotice = true;
      return;
    }
    reportSource = source;
    activeReportId = null;
    activeGoogleReportId = null;
    writeUrlState({ reportSource: source, report: null, greport: null });
  }

  function viewExistingGoogleReports() {
    showGoogleLockedNotice = false;
    reportSource = 'google';
    activeGoogleReportId = null;
    writeUrlState({ reportSource: 'google', report: null, greport: null });
  }

  function navigateTo(nextView) {
    if (window.__scoutPromptSuggesting) {
      if (!confirm('AI is generating prompt suggestions. Leave anyway?')) return;
    }
    view = nextView;
    if (nextView === 'posts') {
      writeUrlState({ view: 'posts', tab: 'general' }, 'push');
      return;
    }
    if (nextView === 'reports') {
      activeReportId = null;
      activeGoogleReportId = null;
      writeUrlState({ view: 'reports', report: null, greport: null }, 'push');
      return;
    }
    if (nextView === 'settings') {
      writeUrlState({ view: 'settings', tab: settingsTab }, 'push');
      return;
    }
    writeUrlState({ view: nextView }, 'push');
  }

  async function handleProjectDeleted(detail) {
    const { id } = detail;
    projectList = projectList.filter(p => p.id !== id);
    if (selectedProjectId === id) {
      selectedProjectId = null;
      selectedPost = null;
      postsList = [];
      postsPage = 1;
      view = 'posts';
      clearUrlState(['project', 'view', 'reportSource', 'report', 'greport', 'post', 'tab', 'platform', 'status', 'dm', 'page', 'limit']);
      if (projectList.length > 0) {
        await selectProject(projectList[0].id);
      }
    }
  }

  async function handleProjectCloned(detail) {
    await loadProjects();
    await selectProject(detail.id);
  }

  // --- Filter handlers ---
  async function applyFilters() {
    selectedPost = null;
    selectedIds = new Set();
    postsPage = 1;
    writeUrlState({ platform: filterPlatform, status: filterStatus, dm: filterDm, score: filterScore, max_age: filterMaxAge, post: null, page: postsPage, limit: postsPageLimit });
    await loadPosts();
  }

  async function changePostsPage(nextPage) {
    if (nextPage < 1 || nextPage === postsPage || nextPage > postsPagination.totalPages) return;
    postsPage = nextPage;
    selectedIds = new Set();
    selectedPost = null;
    writeUrlState({ page: postsPage, limit: postsPageLimit, post: null });
    await loadPosts();
  }

  async function handlePageLimitChange() {
    postsPage = 1;
    selectedIds = new Set();
    selectedPost = null;
    writeUrlState({ page: postsPage, limit: postsPageLimit, post: null });
    await loadPosts();
  }

  // --- Post event handlers ---

  // After a status change removes a post from the current filtered list,
  // auto-select the next post at the same position (or the last one).
  function selectNextPost(previousIndex) {
    if (postsList.length === 0) {
      selectedPost = null;
      writeUrlState({ post: null });
      return;
    }
    const nextIndex = Math.min(previousIndex, postsList.length - 1);
    selectedPost = postsList[nextIndex];
    generateError = null;
    postReplyResult = null;
    writeUrlState({ post: selectedPost.id });
  }

  // DetailPanel dispatches { id, platform, status }
  async function handleStatusChange(detail) {
    const { id, status } = detail;
    const currentIndex = postsList.findIndex(p => p.id === id);
    try {
      await postsApi.update(selectedProjectId, id, { status });
      await loadPosts();
      // Always advance to the next post when skipping from the detail panel
      if (selectedPost && selectedPost.id === id && status === 'skipped') {
        selectNextPost(currentIndex);
      // If the post left the current filtered list for any other reason, also advance
      } else if (selectedPost && selectedPost.id === id && !postsList.find(p => p.id === id)) {
        selectNextPost(currentIndex);
      }
    } catch (err) {
      console.error('Failed to update status:', err);
    }
  }

  // DetailPanel dispatches { id, platform, draft_comment }
  async function handleDraftChange(detail) {
    const { id, draft_comment } = detail;
    try {
      await postsApi.update(selectedProjectId, id, { draft_comment });
      if (selectedPost && selectedPost.id === id) {
        selectedPost = { ...selectedPost, draft_comment };
      }
    } catch (err) {
      console.error('Failed to save draft:', err);
    }
  }

  // DetailPanel dispatches { id, platform }
  async function handleGenerateDraft(detail) {
    const { id } = detail;
    generating = true;
    generateError = null;
    try {
      const updated = await postsApi.generateDraft(selectedProjectId, id);
      if (selectedPost && selectedPost.id === id) {
        selectedPost = { ...selectedPost, ...updated };
      }
      await loadPosts();
    } catch (err) {
      console.error('Failed to generate draft:', err);
      generateError = err.message || 'Generation failed';
    } finally {
      generating = false;
    }
  }

  // DetailPanel dispatches { id, platform, text }
  async function handlePostReply(detail) {
    const { id, text } = detail;
    const currentIndex = postsList.findIndex(p => p.id === id);
    postingReply = true;
    postReplyResult = null;
    try {
      await postsApi.postReply(selectedProjectId, id, text);
      postReplyResult = { type: 'success', message: 'Reply posted successfully!' };
      await loadPosts();
      if (selectedPost && selectedPost.id === id && !postsList.find(p => p.id === id)) {
        selectNextPost(currentIndex);
      }
    } catch (err) {
      console.error('Failed to post reply:', err);
      postReplyResult = { type: 'error', message: err.message || 'Failed to post reply' };
    } finally {
      postingReply = false;
    }
  }

  async function handleSelectAngle(detail) {
    const { reportId, clusterIndex } = detail;
    try {
      const updated = await projectsApi.selectAngle(selectedProjectId, {
        report_id: reportId,
        cluster_index: clusterIndex,
      });
      projectList = projectList.map(p => p.id === updated.id ? { ...p, ...updated } : p);
    } catch (err) {
      console.error('Failed to select angle:', err);
    }
  }

  // PostCard dispatches select event with post as detail
  function handlePostSelect(post) {
    selectedPost = post;
    generateError = null;
    postReplyResult = null;
    writeUrlState({ post: selectedPost.id });
  }

  async function handleNavigateToPostFromManual(post) {
    view = 'posts';
    filterStatus = 'all';
    generateError = null;
    postReplyResult = null;
    writeUrlState({ view: 'posts', post: post.id, status: 'all', tab: 'general' }, 'push');

    let fullPost = null;
    try {
      fullPost = await postsApi.get(selectedProjectId, post.id);
      selectedPost = fullPost;
    } catch {
      selectedPost = post;
    }

    await loadPosts();
    if (!selectedPost || selectedPost.id !== post.id) {
      selectedPost = fullPost || post;
    }
  }

  // --- Bulk selection handlers ---
  function handleBulkToggle(detail) {
    const postId = detail.id;
    if (selectedIds.has(postId)) {
      selectedIds.delete(postId);
    } else {
      selectedIds.add(postId);
    }
    selectedIds = new Set(selectedIds); // trigger reactivity
  }

  function handleSelectAll() {
    if (selectedIds.size === postsList.length) {
      selectedIds = new Set();
    } else {
      selectedIds = new Set(postsList.map(p => p.id));
    }
  }

  function handleBulkCancel() {
    selectedIds = new Set();
  }

  async function handleBulkStatusChange(status) {
    const currentIndex = selectedPost ? postsList.findIndex(p => p.id === selectedPost.id) : -1;
    try {
      await postsApi.bulkUpdate(selectedProjectId, {
        ids: [...selectedIds],
        status,
      });
      selectedIds = new Set();
      await loadPosts();
      // If selected post was bulk-updated and left the filtered list, advance
      if (selectedPost && currentIndex >= 0 && !postsList.find(p => p.id === selectedPost.id)) {
        selectNextPost(currentIndex);
      }
    } catch (err) {
      console.error('Failed to bulk update:', err);
    }
  }

  async function handleRefilterSelected() {
    if (!selectedProjectId || selectedIds.size === 0) return;
    const targets = [...selectedIds].map(id => ({ finding_type: 'post', id }));
    try {
      const job = await filtersApi.createJob(selectedProjectId, {
        requested_scope: 'selected_visible_posts',
        targets,
      });
      selectedIds = new Set();
      // Add job to list and schedule polling
      filterJobs = [job, ...filterJobs];
      scheduleFilterJobPoll();
    } catch (e) {
      console.error('Failed to create re-filter job:', e);
    }
  }

  // --- Mount ---
  let appInitialized = $state(false);

  async function initializeApp() {
    if (appInitialized || typeof window === 'undefined') return;
    appInitialized = true;
    dismissedFilterJobIds = readDismissedFilterJobIds();
    window.addEventListener('popstate', handlePopState);
    await checkOnboarding();
    if (onboardingError || onboardingRequiresSetup) return;
    await continueAppInit();
  }

  async function handleOnboardingComplete() {
    await checkOnboarding();
    if (!appInitialized || onboardingRequiresSetup || onboardingError) return;
    await continueAppInit();
    if (onboardingShowsExploration) {
      checklistOpen = true;
    }
  }

  async function continueAppInit() {
    await Promise.all([loadCapabilities(), loadProjects()]);

    // Initialize browser telemetry client.
    try {
      const telemetryStatus = await telemetryApi.status();
      initTelemetry(telemetryStatus);
    } catch {
      // Telemetry is non-critical; ignore failures.
    }

    const urlState = readUrlState();

    const validViews = ['posts', 'settings', 'sources', 'reports', 'models', 'filtered', 'privacy', 'manual'];
    const validReportSources = ['social', 'google'];
    const validPlatforms = ['all', 'reddit', 'bluesky'];
    const validStatuses = [...POST_STATUSES, 'all'];
    const validMaxAge = ['', '1', '3', '7', '30'];
    if (!validViews.includes(urlState.view)) urlState.view = 'posts';
    if (!validReportSources.includes(urlState.reportSource)) urlState.reportSource = 'social';
    if (!validPlatforms.includes(urlState.platform)) urlState.platform = 'all';
    if (!validStatuses.includes(urlState.status)) urlState.status = 'new';
    if (!validMaxAge.includes(urlState.max_age)) urlState.max_age = '';

    const validTabs = ['general', 'queries', 'prompts', 'schedules', 'advanced'];
    settingsTab = validTabs.includes(urlState.tab) ? urlState.tab : 'general';
    // Normalize legacy tab=queries URLs (queries now live under Sources)
    if (view === 'settings' && settingsTab === 'queries') {
      settingsTab = 'general';
    }

    const targetProject = urlState.project && projectList.find(p => p.id === urlState.project)
      ? urlState.project
      : projectList[0]?.id || null;

    if (targetProject) {
      filterPlatform = urlState.platform;
      filterStatus = urlState.status;
      filterDm = urlState.dm;
      filterScore = urlState.score;
      filterMaxAge = urlState.max_age;
      postsPage = urlState.page;
      postsPageLimit = urlState.limit;
      view = urlState.view;

      selectedProjectId = targetProject;
      writeUrlState({ project: targetProject, view: urlState.view }, 'replace');
      await loadEnabledQueryCount();
      reportSource = urlState.reportSource;
      activeReportId = urlState.report;
      activeGoogleReportId = urlState.greport;

      await loadPosts();
      await fetchInitialRunState();
      await fetchQueryReviewJobs();
      await fetchFilterJobs();

      if (urlState.post) {
        const found = postsList.find(p => p.id === urlState.post);
        if (found) selectedPost = found;
      }
    } else {
      view = urlState.view === 'models' ? 'models' : 'posts';
    }
  }

  onMount(initializeApp);

  $effect(() => {
    if (!appInitialized && typeof window !== 'undefined') {
      initializeApp();
    }
  });

  $effect(() => {
    if (showGoogleLockedNotice) {
      loadGoogleReportPresence();
    }
  });
</script>

<div class="app">
  {#if !onboardingChecked}
    <div class="setup-loading">Loading setup status...</div>
  {:else if onboardingError}
    <div class="setup-error" data-testid="onboarding-status-error">
      <p>{onboardingError}</p>
      <button onclick={checkOnboarding}>Retry setup check</button>
    </div>
  {:else if onboardingRequiresSetup}
    <OnboardingWizard status={onboardingStatus} onStatusReload={handleOnboardingComplete} />
  {:else}
  <ActiveRunBanner runs={activeRuns} {failedRuns} {completedRuns} onDismissFailed={dismissFailedRun} onDismissCompleted={dismissCompletedRun} onCancel={cancelRun} />
  <TopbarJobBanner runningJobs={[...runningQueryJobs, ...normalizedRunningFilterJobs]} completedJobs={[...completedQueryJobs, ...normalizedCompletedFilterJobs]} failedJobs={[...failedQueryJobs, ...normalizedFailedFilterJobs]} onReview={reviewQueryJob} onDismissCompleted={dismissCompletedFilterJob} onDismissFailed={dismissFailedFilterJob} />

  {#if onboardingShowsExploration}
    <ExplorationChecklist
      open={checklistOpen}
      status={onboardingStatus}
      selectedProjectId={selectedProjectId}
      onStatusReload={checkOnboarding}
      onProjectReady={async (projectId) => { await loadProjects(); await selectProject(projectId); }}
      onNavigate={(nextView) => navigateTo(nextView)}
      onClose={() => { checklistOpen = false; }}
      googleLocked={googleLocked}
      onGoogleLocked={() => { showGoogleLockedNotice = true; }}
    />
  {/if}

  <AppShell collapsed={railCollapsed}>
    {#snippet rail()}
      <WorkspaceRail
        projects={projectList}
        selectedProjectId={selectedProjectId}
        {view}
        collapsed={railCollapsed}
        onToggleCollapse={toggleRailCollapsed}
        onSelectProject={handleProjectSelect}
        onRequestCreateProject={openNewProjectModal}
        onNavigate={(nextView) => navigateTo(nextView)}
        {onboardingStatus}
        onToggleChecklist={() => { checklistOpen = !checklistOpen; }}
      />
    {/snippet}

    {#snippet topbar()}
      <TopContextBar
        {view}
        projectName={currentProject?.name}
      >
        {#if selectedProjectId}
          <ScoutRunButton
            projectId={selectedProjectId}
            externalRunning={activeRuns.length > 0}
            {lastRunLabel}
            {enabledQueryCount}
            capabilities={capabilitySnapshot}
            onScoutComplete={handleScoutTriggered}
          />
        {/if}
        <button
          class="icon-btn"
          disabled={!selectedProjectId || loadingPosts}
          onclick={loadPosts}
          title="Refresh"
        >
          &#8635;
        </button>
      </TopContextBar>
    {/snippet}
    {#if !selectedProjectId && view !== 'models'}
      <div class="empty-screen">
        <p class="empty-title">No project selected</p>
        <p class="empty-sub">Create a new project or select an existing one to get started.</p>
        {#if onboardingShowsExploration && onboardingStatus?.items?.find(i => i.id === 'starter_project')?.state === 'pending'}
          <button class="empty-create-btn" onclick={() => { checklistOpen = true; }}>Create your starter project</button>
        {:else}
          <button class="empty-create-btn" onclick={openNewProjectModal}>+ New Project</button>
        {/if}
      </div>
    {:else if view === 'models'}
      <div class="full-width-view">
        <ModelConfig />
      </div>
    {:else if view === 'manual'}
      <div class="full-width-view">
        <ManualTasker projectId={selectedProjectId} onNavigateToPost={handleNavigateToPostFromManual} />
      </div>
    {:else if view === 'posts'}
      <!-- Filter bar / Bulk action bar -->
      {#if bulkMode}
        <div class="filter-bar bulk-bar">
          <div class="bulk-left">
            <span class="bulk-count">{selectedIds.size} post{selectedIds.size === 1 ? '' : 's'} selected</span>
            <div class="bulk-breakdown">
              {#each bulkStatusBreakdown as [status, count]}
                <span class="bulk-status-tag">{count} {status}</span>
              {/each}
            </div>
            <div class="bulk-actions">
              {#if projectMode === 'marketing'}
                <button class="action-btn comment" onclick={() => handleBulkStatusChange('commented')}>
                  Mark as Commented
                </button>
                <button class="action-btn skip" onclick={() => handleBulkStatusChange('skipped')}>
                  Skip
                </button>
                <button class="action-btn exclude" onclick={() => handleBulkStatusChange('excluded')}>
                  Exclude
                </button>
              {:else}
                <button class="action-btn reviewed" onclick={() => handleBulkStatusChange('reviewed')}>
                  Mark as Reviewed
                </button>
                <button class="action-btn star" onclick={() => handleBulkStatusChange('starred')}>
                  Star
                </button>
                <button class="action-btn exclude" onclick={() => handleBulkStatusChange('excluded')}>
                  Exclude
                </button>
              {/if}
              <button class="action-btn refilter" onclick={handleRefilterSelected}>
                {REFILTER_SELECTED_LABEL}
              </button>
            </div>
          </div>
          <div class="bulk-right">
            <label class="select-all-toggle">
              <input
                type="checkbox"
                checked={selectedIds.size === postsList.length}
                onchange={handleSelectAll}
              />
              Select All ({postsList.length})
            </label>
            <button class="bulk-cancel-btn" onclick={handleBulkCancel}>Cancel</button>
          </div>
        </div>
      {:else}
        <div class="filter-bar">
          <div class="filter-group">
            <span class="filter-label">Platform</span>
            <select class="filter-select" bind:value={filterPlatform} onchange={applyFilters}>
              <option value="all">All</option>
              <option value="reddit">Reddit</option>
              <option value="bluesky">Bluesky</option>
            </select>
          </div>
          <div class="filter-group">
            <span class="filter-label">Status</span>
            {#if projectMode === 'research'}
              <select class="filter-select" bind:value={filterStatus} onchange={applyFilters}>
                <option value="new">New</option>
                <option value="starred">Starred</option>
                <option value="reviewed">Reviewed</option>
                <option value="excluded">Excluded</option>
                <option value="all">All</option>
              </select>
            {:else}
              <select class="filter-select" bind:value={filterStatus} onchange={applyFilters}>
                <option value="new">New</option>
                <option value="drafted">Drafted</option>
                <option value="commented">Commented</option>
                <option value="skipped">Skipped</option>
                <option value="all">All</option>
              </select>
            {/if}
          </div>
          <div class="filter-group">
            <span class="filter-label">Score</span>
            <select class="filter-select" bind:value={filterScore} onchange={applyFilters}>
              <option value="">Any</option>
              <option value="lt3">3-</option>
              <option value="3">3+</option>
              <option value="5">5+</option>
              <option value="7">7+</option>
              <option value="9">9+</option>
            </select>
          </div>
          <div class="filter-group">
            <span class="filter-label">When</span>
            <select class="filter-select" bind:value={filterMaxAge} onchange={applyFilters}>
              <option value="">Any time</option>
              <option value="1">Today</option>
              <option value="3">Last 3 days</option>
              <option value="7">Last 7 days</option>
              <option value="30">Last 30 days</option>
            </select>
          </div>
          {#if projectMode === 'marketing'}
            <div class="filter-group">
              <label class="dm-toggle">
                <input type="checkbox" bind:checked={filterDm} onchange={applyFilters} />
                DMs only
              </label>
            </div>
          {/if}
        </div>
      {/if}

      <!-- Posts layout -->
      <div class="posts-layout">
        <!-- Post list sidebar -->
        <div class="post-list">
          <div class="post-list-scroll">
          {#if loadingPosts}
            <div class="loading">Loading posts...</div>
          {:else if postsList.length === 0}
            <div class="empty-list">
              <div class="empty-message">No posts found. Try running ThreadLens or adjusting filters.</div>
              {#if onboardingShowsExploration && showTourCallout}
                <div class="tour-callout-wrapper">
                  <TourCallout
                    title="Scout is user-triggered"
                    body="The Scout button above starts a new search. ThreadLens never starts external scouting automatically — you control when network-heavy work runs."
                    onDismiss={() => { showTourCallout = false; checklistOpen = true; }}
                  />
                </div>
              {/if}
            </div>
          {:else}
            {#each postsList as post (post.id)}
              <FindingCard
                {post}
                selected={!!(selectedPost && selectedPost.id === post.id)}
                {projectMode}
                bulkMode={bulkMode}
                checked={selectedIds.has(post.id)}
                onSelect={handlePostSelect}
                onBulkToggle={handleBulkToggle}
              />
            {/each}
          {/if}
          </div>
          {#if !loadingPosts && postsList.length > 0}
            <div class="pagination-bar">
              <div class="pagination-summary">
                Page {postsPagination.page} of {postsPagination.totalPages} · {postsPagination.total} posts
              </div>
              <div class="pagination-actions">
                <button
                  class="pagination-btn"
                  disabled={!postsPagination.hasPreviousPage || loadingPosts}
                  onclick={() => changePostsPage(postsPagination.page - 1)}
                  aria-label="Previous page"
                  title="Previous page"
                >‹</button>
                <button
                  class="pagination-btn"
                  disabled={!postsPagination.hasNextPage || loadingPosts}
                  onclick={() => changePostsPage(postsPagination.page + 1)}
                  aria-label="Next page"
                  title="Next page"
                >›</button>
              </div>
              <div class="pagination-per-page">
                <span class="pagination-per-page-label">Per page</span>
                <select class="filter-select" bind:value={postsPageLimit} onchange={handlePageLimitChange}>
                  <option value={20}>20</option>
                  <option value={50}>50</option>
                  <option value={100}>100</option>
                </select>
              </div>
            </div>
          {/if}
        </div>

        <!-- Detail panel -->
        <div class="detail-panel">
          <DetailPanel
            post={selectedPost}
            {generating}
            {generateError}
            {postingReply}
            {postReplyResult}
            {projectMode}
            {selectedProjectId}
            onGenerateDraft={handleGenerateDraft}
            onStatusChange={handleStatusChange}
            onDraftChange={handleDraftChange}
            onPostReply={handlePostReply}
          />
        </div>
      </div>

    {:else if view === 'sources'}
      <div class="full-width-view">
        {#key `${selectedProjectId}:${view}`}
          <QueryEditor
            projectId={selectedProjectId}
            reviewJob={activeQueryReviewJob}
            onQueriesChanged={handleQueriesChanged}
            onQueryReviewJobStarted={handleQueryReviewJobStarted}
            onQueryReviewJobHandled={handleQueryReviewJobHandled}
            onQueryReviewModalClosed={closeQueryReviewModal}
          />
        {/key}
      </div>
    {:else if view === 'settings'}
      <div class="full-width-view">
        {#key `${selectedProjectId}:${view}`}
          <ProjectSettings
            projectId={selectedProjectId}
            project={currentProject}
            initialTab={settingsTab}
            onProjectUpdated={handleProjectUpdated}
            onProjectDeleted={handleProjectDeleted}
            onProjectCloned={handleProjectCloned}
            onTabChange={(tab) => { settingsTab = tab; writeUrlState({ tab }); }}
          />
        {/key}
      </div>

    {:else if view === 'reports'}
      <div class="full-width-view">
        <div class="report-source-tabs">
          <button
            class="report-source-btn"
            class:active={reportSource === 'social'}
            onclick={() => selectReportSource('social')}
          >
            Social
          </button>
          <button
            class="report-source-btn"
            class:active={reportSource === 'google'}
            class:locked={googleLocked}
            onclick={() => selectReportSource('google')}
            title={googleLocked ? 'Google Search requires a PARALLEL_API_KEY to be configured on your Scout server' : 'Google Search reports'}
          >
            {#if googleLocked}🔒 {/if}Google
          </button>
        </div>

        {#if reportSource === 'google'}
          {#if activeGoogleReportId}
            <GoogleReportView
              projectId={selectedProjectId}
              reportId={activeGoogleReportId}
              onBack={() => { activeGoogleReportId = null; writeUrlState({ greport: null }); }}
            />
          {:else}
            <GoogleReportsTab
              projectId={selectedProjectId}
              googleLocked={googleLocked}
              onViewReport={({ reportId }) => { activeGoogleReportId = reportId; writeUrlState({ greport: reportId }); }}
            />
          {/if}
        {:else}
          {#if activeReportId}
            <ReportView
              projectId={selectedProjectId}
              reportId={activeReportId}
              onBack={() => { activeReportId = null; writeUrlState({ report: null }); }}
              onSelectAngle={handleSelectAngle}
            />
          {:else}
            <ReportsTab
              projectId={selectedProjectId}
              onViewReport={({ reportId }) => { activeReportId = reportId; writeUrlState({ report: reportId }); }}
            />
          {/if}
        {/if}
      </div>
    {:else if view === 'filtered'}
      <div class="full-width-view">
        <FilteredFindingsTab projectId={selectedProjectId} api={filtersApi} onJobCreated={fetchFilterJobs} onRestored={loadPosts} />
      </div>
    {:else if view === 'privacy'}
      <div class="full-width-view">
        <PrivacySettings />
      </div>
    {/if}
  </AppShell>
  <TelemetryConsentToast />
  {/if}
</div>

<NewProjectModal
  open={showNewProjectModal}
  onClose={() => showNewProjectModal = false}
  onCreate={handleProjectCreate}
/>

<GoogleLockedNotice
  open={showGoogleLockedNotice}
  showViewExistingReports={googleReportCount > 0}
  onClose={() => { showGoogleLockedNotice = false; }}
  onViewExistingReports={viewExistingGoogleReports}
/>

<style>
  :global(*) {
    box-sizing: border-box;
    margin: 0;
    padding: 0;
  }

  :global(body) {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    background: #0f0f13;
    color: #e2e2e8;
    min-height: 100vh;
  }

  :global(::selection) {
    background: #7c6af5;
    color: #ffffff;
  }

  /* Scrollbar styling for webkit browsers */
  :global(*::-webkit-scrollbar) {
    width: 8px;
    height: 8px;
  }

  :global(*::-webkit-scrollbar-track) {
    background: transparent;
  }

  :global(*::-webkit-scrollbar-thumb) {
    background: #3a3a4a;
    border-radius: 4px;
    border: 2px solid transparent;
    background-clip: content-box;
  }

  :global(*::-webkit-scrollbar-thumb:hover) {
    background: #4a4a5a;
    background-clip: content-box;
  }

  /* Scrollbar styling for Firefox */
  :global(*) {
    scrollbar-width: thin;
    scrollbar-color: #3a3a4a transparent;
  }

  :global(*:hover) {
    scrollbar-color: #4a4a5a transparent;
  }

  .app {
    height: 100vh;
    overflow: hidden;
  }

  /* Main */
  .icon-btn {
    width: 34px;
    height: 34px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--color-surface, #23233a);
    border: 1px solid var(--color-border, #2a2a3a);
    border-radius: 6px;
    color: var(--color-text, #e2e2e8);
    font-size: 16px;
    cursor: pointer;
    transition: background 0.15s;
  }

  .icon-btn:hover:not(:disabled) {
    background: var(--color-surface-hover, #2a2a45);
  }

  .icon-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .empty-screen {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 8px;
  }

  .empty-title {
    font-size: 18px;
    color: #e2e2e8;
  }

  .empty-sub {
    font-size: 14px;
    color: #666;
  }

  .empty-create-btn {
    margin-top: 8px;
    padding: 10px 24px;
    background: #7c6af5;
    border: none;
    border-radius: 6px;
    color: #fff;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: opacity 0.15s;
  }

  .empty-create-btn:hover {
    opacity: 0.88;
  }

  /* Filter bar */
  .filter-bar {
    display: flex;
    align-items: center;
    gap: 20px;
    padding: 10px 20px;
    background: #0f0f13;
    border-bottom: 1px solid #2a2a3a;
    flex-shrink: 0;
  }

  .filter-group {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .filter-label {
    font-size: 12px;
    color: #666;
    white-space: nowrap;
  }

  .filter-select {
    padding: 4px 8px;
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 4px;
    color: #e2e2e8;
    font-size: 13px;
    cursor: pointer;
    outline: none;
    transition: border-color 0.15s;
  }

  .filter-select:hover,
  .filter-select:focus {
    border-color: #7c6af5;
  }

  .dm-toggle {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 13px;
    color: #888;
    cursor: pointer;
  }

  .dm-toggle input {
    cursor: pointer;
    accent-color: #7c6af5;
  }

  .bulk-bar {
    justify-content: space-between;
  }

  .bulk-left {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .bulk-count {
    font-size: 13px;
    font-weight: 600;
    color: #d0d0e8;
    white-space: nowrap;
  }

  .bulk-breakdown {
    display: flex;
    gap: 6px;
  }

  .bulk-status-tag {
    font-size: 11px;
    font-weight: 600;
    padding: 2px 8px;
    border-radius: 4px;
    color: #9090a8;
    background: #2a2a3a;
    text-transform: capitalize;
    white-space: nowrap;
  }

  .bulk-actions {
    display: flex;
    gap: 6px;
  }

  .bulk-actions .action-btn {
    padding: 4px 12px;
    font-size: 12px;
    font-weight: 500;
    border-radius: 5px;
    cursor: pointer;
    transition: all 0.15s;
  }

  .bulk-actions .action-btn.reviewed {
    background: #23233a;
    color: #61afef;
    border: 1px solid #61afef50;
  }

  .bulk-actions .action-btn.reviewed:hover {
    background: #61afef25;
  }

  .bulk-actions .action-btn.star {
    background: #23233a;
    color: #e5c07b;
    border: 1px solid #e5c07b50;
  }

  .bulk-actions .action-btn.star:hover {
    background: #e5c07b25;
  }

  .bulk-actions .action-btn.exclude {
    background: #23233a;
    color: #e06c75;
    border: 1px solid #e06c7550;
  }

  .bulk-actions .action-btn.exclude:hover {
    background: #e06c7525;
  }

  .bulk-actions .action-btn.comment {
    background: #98c37918;
    color: #98c379;
    border: 1px solid #98c37935;
  }

  .bulk-actions .action-btn.comment:hover {
    background: #98c37930;
  }

  .bulk-actions .action-btn.skip {
    background: #2a2a3a;
    color: #6b6b80;
    border: 1px solid #3a3a50;
  }

  .bulk-actions .action-btn.skip:hover {
    background: #3a3a50;
    color: #9090a8;
  }

  .bulk-right {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .bulk-actions .action-btn.refilter {
    background: #23233a;
    color: #c0a0f0;
    border: 1px solid #7c6af550;
  }

  .bulk-actions .action-btn.refilter:hover {
    background: #7c6af520;
  }

  .bulk-cancel-btn {
    padding: 4px 12px;
    font-size: 12px;
    color: #9090a8;
    background: #2a2a3a;
    border: 1px solid #3a3a50;
    border-radius: 5px;
    cursor: pointer;
    transition: background 0.15s, color 0.15s;
  }

  .bulk-cancel-btn:hover {
    background: #3a3a50;
    color: #e2e2e8;
  }

  .select-all-toggle {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 13px;
    color: #7c6af5;
    cursor: pointer;
    user-select: none;
  }

  .select-all-toggle input[type="checkbox"] {
    accent-color: #7c6af5;
  }

  /* Posts layout */
  .posts-layout {
    flex: 1;
    display: flex;
    overflow: hidden;
  }

  .post-list {
    width: 340px;
    flex-shrink: 0;
    border-right: 1px solid #2a2a3a;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .post-list-scroll {
    flex: 1;
    overflow-y: auto;
    padding: 8px;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .pagination-bar {
    display: flex;
    flex-direction: row;
    align-items: center;
    gap: 8px;
    padding: 8px;
    border-top: 1px solid #2a2a3a;
    background: #0f0f13;
    flex-shrink: 0;
  }

  .pagination-per-page {
    display: flex;
    align-items: center;
    gap: 4px;
    margin-left: auto;
    flex-shrink: 0;
  }

  .pagination-per-page-label {
    font-size: 11px;
    color: #555;
    white-space: nowrap;
  }

  .pagination-summary {
    font-size: 10px;
    color: #8f8fa5;
    white-space: nowrap;
    flex-shrink: 0;
  }

  .pagination-actions {
    display: flex;
    gap: 4px;
    flex: 1;
    justify-content: center;
    min-width: 0;
  }

  .pagination-btn {
    width: 28px;
    height: 28px;
    padding: 0;
    border-radius: 6px;
    border: 1px solid #2a2a3a;
    background: #1a1a24;
    color: #e2e2e8;
    font-size: 16px;
    line-height: 1;
    cursor: pointer;
    transition: background 0.15s, border-color 0.15s;
  }

  .pagination-btn:hover:not(:disabled) {
    background: #23233a;
    border-color: #7c6af5;
  }

  .pagination-btn:disabled {
    opacity: 0.45;
    cursor: not-allowed;
  }

  .loading {
    padding: 20px;
    text-align: center;
    color: #8f8fa5;
    font-size: 14px;
  }

  .empty-list {
    padding: 20px;
    text-align: center;
    color: #8f8fa5;
    font-size: 14px;
    display: flex;
    flex-direction: column;
    gap: 24px;
    align-items: center;
  }
  
  .empty-message {
    margin-top: 20px;
    line-height: 1.5;
  }
  
  .tour-callout-wrapper {
    text-align: left;
    width: 100%;
  }

  .detail-panel {
    flex: 1;
    overflow-y: auto;
    background: var(--color-canvas);
  }

  /* Full-width views */
  .full-width-view {
    flex: 1;
    overflow-y: auto;
    padding: var(--space-24);
    background: var(--color-canvas);
  }

  /* Target explicit surface classes only — avoids styling arbitrary nested <section>
     elements inside child components. Add .card or .panel to new surfaces as needed. */
  .full-width-view :global(.card),
  .full-width-view :global(.panel) {
    background: var(--color-surface);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg, 8px);
    margin-bottom: var(--space-16);
  }

  .report-source-tabs {
    display: flex;
    gap: 8px;
    margin-bottom: 12px;
  }

  .report-source-btn {
    padding: 6px 12px;
    border: 1px solid #2a2a3a;
    border-radius: 6px;
    background: #1a1a24;
    color: #888;
    font-size: 13px;
    cursor: pointer;
  }

  .report-source-btn.active {
    color: #e2e2e8;
    border-color: #7c6af5;
    background: #23233a;
  }

  .report-source-btn.locked {
    color: #666;
    border-color: #2a2a3a;
    cursor: pointer;
  }

  .report-source-btn.locked:hover {
    border-color: #e5a550;
    color: #e5a550;
  }
</style>
