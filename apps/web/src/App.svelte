<script>
  import { onMount, onDestroy } from 'svelte';
  import ProjectSettings from './components/ProjectSettings.svelte';
  import ScoutRunButton from './components/ScoutRunButton.svelte';
  import ActiveRunBanner from './components/ActiveRunBanner.svelte';
  import PostCard from './components/PostCard.svelte';
  import DetailPanel from './components/DetailPanel.svelte';
  import { projects as projectsApi, posts as postsApi, scout as scoutApi, queries as queriesApi, runtime as runtimeApi, onboarding as onboardingApi } from './lib/api.js';
  import { readUrlState, writeUrlState, clearUrlState } from './lib/url.js';
  import OnboardingWizard from './components/OnboardingWizard.svelte';
  import ExplorationChecklist from './components/onboarding/ExplorationChecklist.svelte';
  import TourCallout from './components/onboarding/TourCallout.svelte';
  import { normalizeOnboardingStatus, shouldShowRequiredWizard, shouldShowExploration } from './lib/onboardingState.js';
  import AppShell from './components/layout/AppShell.svelte';
  import WorkspaceRail from './components/layout/WorkspaceRail.svelte';
  import TopContextBar from './components/layout/TopContextBar.svelte';
  import ReportsTab from './components/ReportsTab.svelte';
  import ReportView from './components/ReportView.svelte';
  import GoogleReportsTab from './components/GoogleReportsTab.svelte';
  import GoogleReportView from './components/GoogleReportView.svelte';
  import ModelConfig from './components/ModelConfig.svelte';
  import { POST_STATUSES } from '@scout/shared';

  // --- Onboarding gate ---
  let onboardingStatus = $state(null);
  let onboardingChecked = $state(false);
  let onboardingError = $state('');
  let showTourCallout = $state(true);
  let checklistOpen = $state(false);
  let onboardingRequiresSetup = $derived(shouldShowRequiredWizard(onboardingStatus));
  let onboardingShowsExploration = $derived(shouldShowExploration(onboardingStatus));

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
  let view = $state('posts'); // 'posts' | 'settings' | 'reports' | 'models'
  let activeReportId = $state(null);
  let activeGoogleReportId = $state(null);
  let reportSource = $state('social'); // 'social' | 'google'
  let capabilitySnapshot = $state(null);

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
  let hasGoogleQueries = $state(false);

  // Run-awareness state
  let activeRuns = $state([]);
  let activeRunIds = $state(new Set());
  let failedRuns = $state([]);
  let completedRuns = $state([]);
  let lastCompletedRun = $state(null);
  let pollTimer = $state(null);
  let timeTick = $state(0);
  let tickTimer = setInterval(() => { timeTick++; }, 60000);

  function relativeTime(dateStr) {
    if (!dateStr) return '';
    const seconds = Math.floor((Date.now() - new Date(dateStr + 'Z').getTime()) / 1000);
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

  function handlePopState() {
    const urlState = readUrlState();
    view = urlState.view;
    reportSource = urlState.reportSource;
    activeReportId = urlState.report;
    activeGoogleReportId = urlState.greport;
    filterPlatform = urlState.platform;
    filterStatus = urlState.status;
    filterDm = urlState.dm;
    filterScore = urlState.score;
    postsPage = urlState.page;
    postsPageLimit = urlState.limit;

    if (urlState.project && urlState.project !== selectedProjectId) {
      stopPoll();
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
    clearInterval(tickTimer);
  });

  // Filters
  let filterPlatform = $state('all'); // 'all' | 'reddit' | 'bluesky'
  let filterStatus = $state('new');   // 'new' | 'drafted' | 'commented' | 'all'
  let filterDm = $state(false);
  let filterScore = $state('');        // '' | '3' | '5' | '7' | '9'

  // Bulk selection state
  let selectedIds = $state(new Set());
  let bulkMode = $derived(selectedIds.size > 0);
  let bulkStatusBreakdown = $derived(Object.entries(
    postsList.filter(p => selectedIds.has(p.id)).reduce((acc, p) => {
      acc[p.status] = (acc[p.status] || 0) + 1;
      return acc;
    }, {})
  ));

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
  }

  async function loadEnabledQueryCount() {
    if (!selectedProjectId) {
      enabledQueryCount = null;
      hasGoogleQueries = false;
      return;
    }
    try {
      const list = await queriesApi.list(selectedProjectId);
      enabledQueryCount = list.filter(q => q.enabled).length;
      hasGoogleQueries = list.some(q => q.enabled && q.platform === 'google');
    } catch {
      enabledQueryCount = null;
      hasGoogleQueries = false;
    }
  }

  async function handleProjectSelect(id) {
    await selectProject(id);
  }

  async function handleProjectCreate(detail) {
    const { id, name, mode } = detail;
    try {
      const created = await projectsApi.create({ id, name, mode });
      await loadProjects();
      await selectProject(created.id);
    } catch (e) {
      console.error('Failed to create project:', e);
    }
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
    if (source === 'google' && !hasGoogleQueries) return;
    reportSource = source;
    activeReportId = null;
    activeGoogleReportId = null;
    writeUrlState({ reportSource: source, report: null, greport: null });
  }

  $effect(() => {
    if (!hasGoogleQueries && reportSource === 'google') {
      reportSource = 'social';
      activeGoogleReportId = null;
      writeUrlState({ reportSource: 'social' });
    }
  });

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
    writeUrlState({ platform: filterPlatform, status: filterStatus, dm: filterDm, score: filterScore, post: null, page: postsPage, limit: postsPageLimit });
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
      // If the post left the current filtered list, advance to the next one
      if (selectedPost && selectedPost.id === id && !postsList.find(p => p.id === id)) {
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

  // --- Mount ---
  let appInitialized = $state(false);

  async function initializeApp() {
    if (appInitialized || typeof window === 'undefined') return;
    appInitialized = true;
    window.addEventListener('popstate', handlePopState);
    await checkOnboarding();
    if (onboardingError || onboardingRequiresSetup) return;
    await continueAppInit();
  }

  async function handleOnboardingComplete() {
    await checkOnboarding();
    if (!appInitialized || onboardingRequiresSetup || onboardingError) return;
    await continueAppInit();
  }

  async function continueAppInit() {
    await Promise.all([loadCapabilities(), loadProjects()]);
    const urlState = readUrlState();

    const validViews = ['posts', 'settings', 'reports', 'models'];
    const validReportSources = ['social', 'google'];
    const validPlatforms = ['all', 'reddit', 'bluesky'];
    const validStatuses = [...POST_STATUSES, 'all'];
    if (!validViews.includes(urlState.view)) urlState.view = 'posts';
    if (!validReportSources.includes(urlState.reportSource)) urlState.reportSource = 'social';
    if (!validPlatforms.includes(urlState.platform)) urlState.platform = 'all';
    if (!validStatuses.includes(urlState.status)) urlState.status = 'new';

    const targetProject = urlState.project && projectList.find(p => p.id === urlState.project)
      ? urlState.project
      : projectList[0]?.id || null;

    if (targetProject) {
      const proj = projectList.find(p => p.id === targetProject);
      if (urlState.view === 'reports' && proj?.mode !== 'research') {
        urlState.view = 'posts';
      }

      filterPlatform = urlState.platform;
      filterStatus = urlState.status;
      filterDm = urlState.dm;
      filterScore = urlState.score;
      postsPage = urlState.page;
      postsPageLimit = urlState.limit;
      view = urlState.view;

      selectedProjectId = targetProject;
      writeUrlState({ project: targetProject }, 'replace');
      await loadEnabledQueryCount();
      reportSource = urlState.reportSource;
      activeReportId = urlState.report;
      activeGoogleReportId = urlState.greport;

      await loadPosts();
      await fetchInitialRunState();

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

  {#if onboardingShowsExploration}
    <ExplorationChecklist
      open={checklistOpen}
      status={onboardingStatus}
      selectedProjectId={selectedProjectId}
      onStatusReload={checkOnboarding}
      onProjectReady={async (projectId) => { await loadProjects(); await selectProject(projectId); }}
      onNavigate={(nextView) => { view = nextView; writeUrlState({ view: nextView }, 'push'); }}
      onClose={() => { checklistOpen = false; }}
    />
  {/if}

  <AppShell>
    {#snippet rail()}
      <WorkspaceRail
        projects={projectList}
        selectedProjectId={selectedProjectId}
        {view}
        onSelectProject={handleProjectSelect}
        onCreateProject={handleProjectCreate}
        onNavigate={(nextView) => {
          if (nextView === 'sources') {
            view = 'settings';
            writeUrlState({ view: 'settings' }, 'push');
            return;
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
          writeUrlState({ view: nextView }, 'push');
        }}
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
        <p class="empty-sub">Use the workspace rail to select or create a project.</p>
      </div>
    {:else if view === 'models'}
      <div class="full-width-view">
        <ModelConfig />
      </div>
    {:else if view === 'posts'}
      <!-- Filter bar / Bulk action bar -->
      {#if bulkMode && projectMode === 'research'}
        <div class="filter-bar bulk-bar">
          <div class="bulk-left">
            <span class="bulk-count">{selectedIds.size} post{selectedIds.size === 1 ? '' : 's'} selected</span>
            <div class="bulk-breakdown">
              {#each bulkStatusBreakdown as [status, count]}
                <span class="bulk-status-tag">{count} {status}</span>
              {/each}
            </div>
            <div class="bulk-actions">
              <button class="action-btn reviewed" onclick={() => handleBulkStatusChange('reviewed')}>
                Mark as Reviewed
              </button>
              <button class="action-btn star" onclick={() => handleBulkStatusChange('starred')}>
                Star
              </button>
              <button class="action-btn exclude" onclick={() => handleBulkStatusChange('excluded')}>
                Exclude
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
            <span class="filter-label">Platform:</span>
            {#each ['all', 'reddit', 'bluesky'] as p}
              <button
                class="filter-btn"
                class:active={filterPlatform === p}
                onclick={() => { filterPlatform = p; applyFilters(); }}
              >
                {p === 'all' ? 'All' : p.charAt(0).toUpperCase() + p.slice(1)}
              </button>
            {/each}
          </div>
          <div class="filter-group">
            <span class="filter-label">Status:</span>
            {#if projectMode === 'research'}
              {#each ['new', 'starred', 'reviewed', 'excluded', 'all'] as s}
                <button
                  class="filter-btn"
                  class:active={filterStatus === s}
                  onclick={() => { filterStatus = s; applyFilters(); }}
                >
                  {s.charAt(0).toUpperCase() + s.slice(1)}
                </button>
              {/each}
            {:else}
              {#each ['new', 'drafted', 'commented', 'all'] as s}
                <button
                  class="filter-btn"
                  class:active={filterStatus === s}
                  onclick={() => { filterStatus = s; applyFilters(); }}
                >
                  {s.charAt(0).toUpperCase() + s.slice(1)}
                </button>
              {/each}
            {/if}
          </div>
          <div class="filter-group">
            <span class="filter-label">Score:</span>
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
            <span class="filter-label">Per page:</span>
            <select class="filter-select" bind:value={postsPageLimit} onchange={handlePageLimitChange}>
              <option value={20}>20</option>
              <option value={50}>50</option>
              <option value={100}>100</option>
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
          {#if loadingPosts}
            <div class="loading">Loading posts...</div>
          {:else if postsList.length === 0}
            <div class="empty-list">
              No posts found. Try running ThreadLens or adjusting filters.
              {#if onboardingShowsExploration && showTourCallout}
                <TourCallout
                  title="Scout is user-triggered"
                  body="The Scout button above starts a new search. ThreadLens never starts external scouting automatically — you control when network-heavy work runs."
                  onDismiss={() => { showTourCallout = false; checklistOpen = true; }}
                />
              {/if}
            </div>
          {:else}
            {#each postsList as post (post.id)}
              <PostCard
                {post}
                selected={!!(selectedPost && selectedPost.id === post.id)}
                {projectMode}
                bulkMode={bulkMode}
                checked={selectedIds.has(post.id)}
                onSelect={handlePostSelect}
                onBulkToggle={handleBulkToggle}
              />
            {/each}
            <div class="pagination-bar">
              <div class="pagination-summary">
                Page {postsPagination.page} of {postsPagination.totalPages} · {postsPagination.total} posts
              </div>
              <div class="pagination-actions">
                <button
                  class="pagination-btn"
                  disabled={!postsPagination.hasPreviousPage || loadingPosts}
                  onclick={() => changePostsPage(postsPagination.page - 1)}
                >Previous</button>
                <button
                  class="pagination-btn"
                  disabled={!postsPagination.hasNextPage || loadingPosts}
                  onclick={() => changePostsPage(postsPagination.page + 1)}
                >Next</button>
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
            onGenerateDraft={handleGenerateDraft}
            onStatusChange={handleStatusChange}
            onDraftChange={handleDraftChange}
            onPostReply={handlePostReply}
          />
        </div>
      </div>

    {:else if view === 'settings'}
      <div class="full-width-view">
        <ProjectSettings
          projectId={selectedProjectId}
          project={currentProject}
          initialTab={readUrlState().tab}
          onProjectUpdated={handleProjectUpdated}
          onQueriesChanged={handleQueriesChanged}
          onProjectDeleted={handleProjectDeleted}
          onProjectCloned={handleProjectCloned}
          onTabChange={(tab) => writeUrlState({ tab })}
        />
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
          {#if hasGoogleQueries}
            <button
              class="report-source-btn"
              class:active={reportSource === 'google'}
              onclick={() => selectReportSource('google')}
            >
              Google
            </button>
          {/if}
        </div>

        {#if reportSource === 'google' && hasGoogleQueries}
          {#if activeGoogleReportId}
            <GoogleReportView
              projectId={selectedProjectId}
              reportId={activeGoogleReportId}
              onBack={() => { activeGoogleReportId = null; writeUrlState({ greport: null }); }}
            />
          {:else}
            <GoogleReportsTab
              projectId={selectedProjectId}
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
    {/if}
  </AppShell>
  {/if}
</div>

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
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .filter-btn {
    padding: 4px 10px;
    background: none;
    border: 1px solid #2a2a3a;
    border-radius: 4px;
    color: #888;
    font-size: 13px;
    cursor: pointer;
    transition: all 0.15s;
  }

  .filter-btn:hover {
    border-color: #7c6af5;
    color: #e2e2e8;
  }

  .filter-btn.active {
    background: #2a2a45;
    border-color: #7c6af5;
    color: #e2e2e8;
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

  .bulk-right {
    display: flex;
    align-items: center;
    gap: 12px;
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
    overflow-y: auto;
    padding: 8px;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .pagination-bar {
    margin-top: auto;
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 12px 8px 4px;
    border-top: 1px solid #2a2a3a;
    background: linear-gradient(180deg, rgba(15, 15, 19, 0) 0%, rgba(15, 15, 19, 1) 24%);
    position: sticky;
    bottom: 0;
  }

  .pagination-summary {
    font-size: 12px;
    color: #8f8fa5;
    text-align: center;
  }

  .pagination-actions {
    display: flex;
    gap: 8px;
  }

  .pagination-btn {
    flex: 1;
    padding: 8px 10px;
    border-radius: 6px;
    border: 1px solid #2a2a3a;
    background: #1a1a24;
    color: #e2e2e8;
    font-size: 13px;
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

  .loading,
  .empty-list {
    padding: 20px;
    text-align: center;
    color: #666;
    font-size: 14px;
  }

  .detail-panel {
    flex: 1;
    overflow-y: auto;
  }

  /* Full-width views */
  .full-width-view {
    flex: 1;
    overflow-y: auto;
    padding: 24px;
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
</style>
