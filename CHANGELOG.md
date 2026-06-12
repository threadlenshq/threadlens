# Changelog

## 0.10.0 - 2026-06-12

Tag: `threadlens-v0.10.0`

### Features

- feat(onboarding): extend provider detection, test connectivity for all providers, hard gate (`e6ad719`)
- feat(web): add progress bar, smart first-scout labels, post-scout guidance to checklist (`68708b1`)
- feat(web): add query seeding panel with editable AI suggestions to exploration checklist (`02992d2`)
- feat(onboarding): add Description to starter project, populate SeededQueryCount (`14cf7e0`)
- feat(onboarding): immediate env activation via os.Setenv in Save() (`82b4e49`)
- feat(web): add Test Connection button to AI provider step in wizard (`429461d`)
- feat(onboarding): add POST /api/onboarding/test-ai endpoint (`08bbcef`)

### Fixes

- fix(onboarding): wire exploration checklist into full user flow (`712f982`)
- fix(onboarding): increase CLI connectivity timeout to 60s for copilot (`3b8f90d`)
- fix(onboarding): restore testCLIProvider and increase connectivity timeout to 30s (`3fbbcfc`)
- fix(web): restore Test Connection button for non-secret providers (CLI tools) (`0688451`)
- fix(web): include status filter when dm-only is active (`f234440`)
- fix(ai): strip Copilot skill noise from generated draft text (`d456671`)

### Maintenance

- refactor(api): collapse GetProjectWithStats into one query, drop dead var (`e87912e`)
- refactor(api): dedup not-found mapping, markdown-fence stripping, retry backoff (`1acfb65`)
- refactor(api): hoist more hot-path regexes and dedup scan helpers (`3e13f4f`)
- refactor(api): hoist hot-path regexes, batch dm_targets, drop hand-rolled toString (`0b3be05`)
- refactor(onboarding): simplify connectivity, capabilities, and seeding flow (`57bd27d`)

## 0.9.1 - 2026-06-09

Tag: `threadlens-v0.9.1`

### Maintenance

- Version alignment only for the unified ThreadLens release.

## 0.9.0 - 2026-06-09

Tag: `threadlens-v0.9.0`

### Features

- feat(bootstrap): add opencode to bridge runtimes array in ai-bridge.json (`c5f7dec`)
- feat(bridge): register OpencodeRuntime in scout-ai-bridge daemon (`58f4a03`)
- feat(ai): extend isBridgeCompatible for opencode and opencode-go provider tags (`0d197ad`)
- feat(ai): wire OpencodeProvider into service with fallback chain and bridge translation (`b757258`)
- feat(ai): add 16 opencode and opencode-go catalog entries (`16f7541`)
- feat(ai): add OpencodeProvider direct in-process wrapper with NDJSON parsing (`aa28842`)
- feat(bridge): add OpencodeRuntime with NDJSON parser and prefix translation (`4508564`)

### Fixes

- fix(ai): cap scorer batch execution with 3-minute context timeout (`63547cb`)

### Documentation

- docs(open-core): document opencode provider from PR #12 (`f87340c`)

### Maintenance

- chore: remove legacy Express API and all references (`444de18`)
- test(ai): add bridge provider tests for opencode compatibility and tag translation (`61fd96a`)
- test(ai): update fallback order test and add opencode-go routing tests (`9202875`)
- test(bootstrap): assert opencode in ai-bridge.json runtimes array (`75b6b29`)
- test(ai): add OpencodeProvider unit tests for name, availability, NDJSON parsing, prefix translation (`6ef5227`)
- test(bridge): add OpencodeRuntime unit tests for detect, generate, NDJSON parsing, prefix translation (`830d451`)

## 0.8.0 - 2026-06-08

Tag: `threadlens-v0.8.0`

### Features

- feat: fetch bluesky replies for dm targets (`cd13e27`)
- feat: support dm target post filtering (`163ea3f`)
- feat: apply dm candidate filter to ranking (`7efe28e`)
- feat: add deterministic dm candidate filter (`de407a0`)

### Fixes

- fix(web): include projectId in DM draft API URLs (`e3ed67d`)
- fix(pipeline): authenticate bluesky getPostThread to avoid 401 on restricted posts (`caaf8f2`)
- fix(ai-bridge): restart broken daemons at boot and retry transient failures (`16bad83`)
- fix: use safe handler assertion pattern and add second reply check in bluesky reply test (`2390b7c`)
- fix(test): restore paid-promotion spam filter and remove brittle source-scan guardrail (`cee1b1f`)
- fix: guard against future-dated ProfileCreatedAt in dm candidate filter (`2aaf87e`)

### Maintenance

- test: validate dm target generation workflow (`d59843a`)
- test: cover dm target generation preservation rules (`c3e73c5`)
- test: cover deterministic dm candidate filter (`99fa521`)

## 0.7.0 - 2026-06-07

Tag: `threadlens-v0.7.0`

### Features

- feat(pipeline): surface scoring progress in scout step (`9e6340b`)
- feat: run dm target generation after social storage (`50182ab`)
- feat(pipeline): implement FetchBlueskyReplies for Bluesky thread reply fetching (`65557f5`)
- feat: parse dm posts filter (`66c0145`)
- feat: add HasDMTargets filter, CountDMTargets, InsertDMTargets to post repository (`7bf8d2b`)
- feat: add deterministic dm target generator (`7d29a01`)
- feat(pipeline): implement deterministic DM target generator (`8517d20`)
- feat(domain): add DMTargetInsert struct for repository insertion (`fcc1085`)
- feat(detail-panel): auto-advance to next post after skip (`0baddc4`)
- feat: add dismiss to completed/failed filter job banners (`7db1ccf`)
- feat: wire filtered findings workflow (`3cbeb7c`)
- feat: add filtered findings owner tab (`6da65d4`)
- feat: add filter recovery modal (`0023485`)
- feat: add filtering frontend api labels (`478cad8`)
- feat: expose owner filtering endpoints (`6af31a3`)
- feat: filter google findings before reports (`5367ccd`)
- feat: retain filtered social candidates (`b87f0a8`)
- feat: hide filtered google results by default (`5f6bdbf`)
- feat: add filtering repository helpers (trust records, filtered findings, recovery, filter jobs) (`a9164d1`)
- feat: hide filtered social posts by default (`645c987`)
- feat: add conservative filter classifier (`cc853df`)
- feat: persist filter state schema (`c8526e7`)
- feat(domain): add shared filtering contract types and constants (`8b59bfa`)

### Fixes

- fix(pipeline): distinguish pipeline timeout from user cancel (`b39c9cd`)
- fix(pipeline): isolate scorer retries from pipeline deadline (`5baafca`)
- fix(web): suppress status filter when dm=true (`b1330ba`)
- fix(pipeline): mark failed/cancelled runs even when context is dead (`1250d37`)
- fix(filter-jobs): show filter banners as ephemeral state — only on transition, auto-dismiss after 8s (`d61dabc`)
- fix(detail-panel): migrate from createEventDispatcher to Svelte 5 callback props (`93ad3d9`)
- fix(ui): enable bulk select checkbox and actions for all project modes (`522d11a`)

### Documentation

- docs: explain filtering visibility model (`c0e4a78`)

### Maintenance

- chore(pipeline): extend scout run timeout to 30m (`4778755`)
- refactor(reddit): use shared fetch helper in FetchRedditContext (`077ada9`)
- test: cover dm-only posts api param (`c049486`)
- test: cover bluesky reply candidate fetcher (`f287272`)
- test: cover dm-only post filter (`cc157cc`)
- test(pipeline): fix readability issues in DMTargetGenerator tests (`156de1e`)
- test(pipeline): fix DMTargetGenerator tests for spec alignment (`d2f5ec0`)
- test(pipeline): add failing DMTargetGenerator unit tests (`c1ab6f6`)
- test(domain): add post filter metadata and google model field coverage tests (`a351d2c`)
- refactor: remove stale apps/web source, consolidate to open-core (`3784801`)

## 0.6.0 - 2026-05-30

Tag: `threadlens-v0.6.0`

### Features

- feat(showcase): add Vite bootstrap files for showcase app (`fa26a12`)
- feat: add ui:dev scripts to launch showcase app from root and open-core (`f400e86`)
- feat(showcase): add App.svelte with manifest validation and sidebar/detail layout (`5b35fff`)
- feat(showcase): add static preview snippets for all v1 scenario groups (Task 7) (`fad5f11`)
- feat(showcase): add ScenarioSidebar, ScenarioDetail, and PreviewCard components (Task 6) (`ce03862`)
- feat(showcase): add scenario manifest and preview registry (Task 5) (`be141e9`)
- feat(showcase): add reusable styles, utility shims, and Svelte mount entry (`5a62f61`)

### Fixes

- fix(docker): correct go module paths in dev stage (`22daf0e`)
- fix(reddit): solve JS challenge to bypass 403 on anonymous API access (`4ee4a05`)
- fix: use $state for validationError in showcase App.svelte; correct showcase port in README (`7cb6fdf`)
- fix(showcase): guard App.svelte against non-array scenarios and non-Error throws (`f845ca1`)

### Documentation

- docs: add showcase design spec and implementation plan (`dcebdb1`)
- docs: add contributor showcase notes to root and open-core READMEs (`6da7c96`)

### Maintenance

- chore: remove unused ad hoc component preview files (`75aa335`)
- Add showcase manifest validation helper and unit tests (`9f06e99`)
- Add static mock data for deterministic showcase previews (`27ba11b`)
- Remove unused StatusChip and FindingToolbar components; simplify ComponentPreview to InsightPane only (`b093668`)

## 0.5.0 - 2026-05-29

Tag: `threadlens-v0.5.0`

### Features

- feat(app): wire Google lock state through App.svelte (task 9) (`bed917c`)
- feat: update Google tab copy and onboarding for locked state (`7b0df8f`)
- feat: show Google as locked in ScoutRunButton, open locked modal on interaction (`9f37d9a`)
- feat: add Google gating capability helpers and locked notice modal (`043cdf4`)
- feat(entitlements): deny CapabilityScoutRunGoogle when PARALLEL_API_KEY is absent (`54522c9`)

### Fixes

- fix(app): compact sidebar filters and pagination (`13952d1`)
- fix(app): remove dead skipLockCheck option and guard async project race in loadGoogleReportPresence (`296695e`)
- fix(docs): add missing pages to sidebar nav (`6d4972d`)
- fix(docs): correct broken anchor fragment links to credential-setup headings (`b902e21`)
- fix(docs): remove literal anchor syntax (`a157330`)

### Maintenance

- test(frontend): add google-gating tests for locked Google UX (`8ae24ec`)
- test: add Google gating safety tests and trim PARALLEL_API_KEY before use (`2e52583`)
- test(runtime): add Google capability tests gated on PARALLEL_API_KEY (`7f8a7e7`)
- refactor(entitlements): extract googleScoutConfigured helper and make Google capability explicit (`47a969f`)
- test(entitlements): split unset/whitespace cases, add local message code constant (`b6ec6d6`)
- test(entitlements): add Google capability environment-aware tests (`3098102`)

## 0.4.0 - 2026-05-28

Tag: `threadlens-v0.4.0`

### Features

- feat(App): add query-review job state, polling, banner, and Settings wiring (`deb1639`)
- feat(web): add QueryJobReviewModal component for query review jobs (`fdc8be5`)
- feat: add queryReviewJobs API client export to open-core web api.js (`1a1a925`)
- feat: mount query review job routes in app router (`887ce7b`)
- feat: add query review job HTTP handlers (Task 7) (`0aa0ff7`)
- feat(repository): add query review job persistence helpers (`3399794`)
- feat(db): add query_review_jobs table and indexes to SQLite migration (`44241ea`)

### Fixes

- fix(query-review): add missing background job migrations (`3b9c87f`)
- fix(ai): set explicit config path in TestAutoLaunchWithHelper to avoid host file interference (`4f1237b`)
- fix(web): correct backend contract in QueryJobReviewModal (kind, resolution, refine shape) (`a0598e3`)
- fix(repository): tighten CreateQueryReviewJob signature to use domain.QueryReviewKind and correct param order (`c6dc510`)
- fix(threadlens): improve Reddit query quality and refinement (`b3395eb`)
- fix(ai): set explicit config path in TestAutoLaunchWithHelper to avoid host file interference (`3d974dc`)

### Maintenance

- Add frontend regression tests for query review background flow (`384b135`)
- refactor(QueryEditor): replace blocking suggest/refine with background job flow (`b387df5`)
- test(handlers): add backend contract tests for query review job API (Task 9) (`358d44f`)
- startup: mark stale query review jobs failed on boot (Task 6) (`63c614c`)
- test(db): add query_review_jobs to wiring table assertions (`82457e3`)
- Add query_review_jobs table and indexes to Postgres migration (`1e9a561`)
- Add QueryReviewJob domain type with kind/status/resolution constants (`255ae31`)

## 0.3.0 - 2026-05-27

Tag: `threadlens-v0.3.0`

### Features

- feat(web): add API timestamp parser helper (`af0504b`)
- feat: auto-detect scheduler timezone with optional SCOUT_TIMEZONE override (`337ff7c`)
- feat(web): migrate new project flow to modal with empty-state CTA (#4) (`e4ad227`)

### Fixes

- fix(web): use API timestamp parser in run UI (`59a53bd`)

### Documentation

- docs: add credential setup guide with official provider links (`8a08529`)

## 0.2.6 - 2026-05-25

Tag: `threadlens-v0.2.6`

### Maintenance

- Version alignment only for the unified ThreadLens release.

## 0.2.5 - 2026-05-25

Tag: `threadlens-v0.2.5`

### Maintenance

- Version alignment only for the unified ThreadLens release.

## 0.2.4 - 2026-05-24

Tag: `threadlens-v0.2.4`

### Maintenance

- chore(release): version bump alongside create-threadlens-app 0.2.4 (image tag fix).

## 0.2.3 - 2026-05-23

Tag: `threadlens-v0.2.3`

### Documentation

- docs: update release skill and docs for unified threadlens-vVERSION release flow (`f0f15b3`)

## 0.2.1 - 2026-05-21

Tag: `threadlens-v0.2.1`

### Fixes

- fix(threadlens): refocus self-host activation copy (`10a92a4`)
- fix(open-core): align self-host smoke check with activation docs (`62d17bd`)

## 0.2.0 - 2026-05-20

Tag: `threadlens-v0.2.0`

### Features

- feat(release): add smart-release CLI and automation system (#3) (`4ee95d3`)
- feat: wire threadlens installer scripts (`5104e0a`)
- feat: update enabled-query warning to minimalist first-value copy (`d783fff`)
- feat: show narrow first-query guidance when no queries exist (Task 6.2) (`fd21264`)
- feat: add activationProgressLabel helper to onboardingState (`35abedf`)
- feat: update OnboardingWizard welcome copy to ThreadLens self-host activation messaging (`7281593`)
- feat(bridge): add advanced-user helper commands (`0ef895e`)
- feat(onboarding): modernize workspace tour UI (`ac7364a`)
- feat(WorkspaceRail): stretch rail-nav to fill full rail-inner height (`d170f26`)
- feat(WorkspaceRail): move Models nav item to bottom of sidebar (`0649214`)
- feat(App): persist rail collapsed state to localStorage (`ba154ec`)
- feat(WorkspaceRail): collapse with icons, tooltips, and nav item icons (`ec94f8f`)
- feat(ProjectSelector): collapse to initial badge; fly dropdown right when slim (`cb390e5`)
- feat(App): add railCollapsed state and pass collapse props to shell (`db27b2b`)
- feat(AppShell): add collapsible rail prop and CSS variable-driven width (`bfe8a9c`)
- feat: polish ModelConfig into a capability dashboard with Surface cards (`c98dce4`)
- feat(query-editor): group queries by platform in Surface cards (`ad27197`)
- feat: add polished empty states and loading skeletons to ReportsTab (`60cdf77`)
- feat(shell): add insight slot and responsive flex layout to AppShell (`2395036`)
- feat: add FindingToolbar for bulk inbox actions (`e0238d6`)
- feat: add InsightPane component for wide desktop (`916d0c6`)
- feat: add Surface, EmptyState, and LoadingSkeleton UI primitives (`5dd917e`)
- feat(ui): add StatusChip and FindingCard inbox primitives with dark token styling (`cad58f2`)
- feat: add AppShell, WorkspaceRail, TopContextBar layout primitives (`1a70867`)
- feat: convert ExplorationChecklist to slide-in right drawer (`281594e`)
- feat: configfile envfile helpers with test coverage (`ced956b`)
- feat: ThreadLens frontend onboarding wizard and exploration checklist (Tasks 14-22) (`a2a1223`)
- feat: add ThreadLens onboarding state API (Tasks 8-13) (`259341b`)
- feat(onboarding): expand handler routes and test coverage for full API contract (`37ccc80`)
- feat(onboarding): implement CreateStarterProject with idempotent project/query creation (`0113c98`)
- feat(onboarding): add step validation and RestartRequired to Task 6 transitions (`5f1cb97`)
- feat(onboarding): Task 5 – extend service with v1 progress model and GetStatus (`3addbef`)
- feat(onboarding): add state model, types, and helper functions (Task 3) (`e3ec617`)
- feat(onboarding): add StateKey constant for ThreadLens v1 state (`d5a557c`)
- feat(docker): mount writable .env into /data/.env and enable onboarding docker mode (`1def46c`)
- feat(onboarding): wire OnboardingService into Go app bootstrap and router (`761851e`)
- feat(onboarding): add HTTP handlers, ServiceIface, and ErrDisabled sentinel (Task 9) (`2b1224d`)
- feat(onboarding): implement Service with IsComplete, GetStatus, Save, Reset (`3c43f4f`)
- feat(onboarding): add LoadConfig and Config type for onboarding env config (`09230b5`)
- feat(settings): add generic app-settings repository (`b06767c`)
- feat(configfile): add env-file helper with atomic write, masking, and injection rejection (`c89faf5`)
- feat(models): mark host bridge as optional-local transport in catalog status (`3c2aad0`)
- feat(ai): add local-only bridge discovery policy gated by SCOUT_AI_BRIDGE_MODE (`ce391ab`)
- feat(ai): add runtime-aware bridge generation and health checks (Task 3) (`6ba12a2`)
- feat(queries): gracefully handle missing AI provider in refine endpoint (`2571cd2`)
- feat(bridge): add package scripts for bridge (`e9da7bf`)
- feat(bridge): wire docker compose host bridge access (`87ac663`)
- feat(bridge): run bootstrap during docker dev (`8bc6d0d`)
- feat(bridge): add bootstrap script (`41fe1f6`)
- feat(bridge): send provider through bridge client (`23a7c06`)
- feat(bridge): add scout-ai-bridge daemon command (`766b257`)
- feat(bridge): add HTTP daemon server (`adb0950`)
- feat(bridge): add CLI runtime implementations (`d8b53db`)
- feat(bridge): add Registry with cached detection and Generate dispatch (`96a9bff`)
- feat(bridge): add ValidateBindAddress and LocalhostURLForBind (Task 3) (`a21f818`)
- feat(bridge): add bridge package types for CLI daemon API (`db3a303`)
- feat: expose bridge status in model catalog (`8562770`)
- feat: prefer host bridge for ai fallback (`9ede42c`)
- feat(ai): add bridge discovery config loader with tests (`0a2c2c9`)
- feat(web): add capability-aware UI — Tasks 35-41 (`d871c67`)
- feat(services): inject entitlement checks into ProjectService (Task 26) (`b5b9d8b`)
- feat(ai): add usage metering to GenerateForTask (Task 31) (`54a714d`)
- feat(config): add RuntimeMode field loaded from THREADLENS_RUNTIME_MODE env var (`eddf8d2`)
- feat(handlers): add MountRuntimeRoutes for capability discovery and prompt-pack listing (Task 18) (`ef5b92f`)
- feat(services): add RuntimeService for capability snapshot and template listing (Task 17) (`652a893`)
- feat(modules): add compiled module registry seam (Tasks 14-16) (`ada7134`)
- feat(templates): add prompt-pack catalog seam with local empty catalog (`bd8a625`)
- feat(usage): add usage metering seam with NoopMeter and MemoryMeter (`c8dd1c8`)
- feat(tenant): add request-scoped tenant context seam (`f93c81f`)
- feat(entitlements): create entitlements package with local resolver and capability seam (`f0a53ec`)
- feat: add saas database migrations (`0fa6ba3`)
- feat: add saas database config boundary (`b78793e`)
- feat: add core seed markers (`7098f76`)
- feat(api): wire Go API to shared open-core/db module (Task 6) (`5ad5fcd`)
- feat: add shared database open helpers (`71ae110`)
- feat: add core database migrations (`6798622`)
- feat: restore spec-exact Task 2 config contract with open-core root detection (`7504a72`)
- feat: add shared database config (`15536f8`)
- feat: define open-core docker dev and prod profiles (`a8746ee`)
- feat: add open-core web docker image (`0ba4c5f`)
- feat(docker): rebuild Dockerfile.go-api from open-core root with web assets (`1955811`)
- feat: make open-core docker scripts canonical (`bd1e998`)
- feat: add live-link Docker dev mode for adjacent open-core checkout (Task 7) (`2f05805`)
- feat: add open-core subtree sync script and documentation (Task 5) (`107ac23`)
- feat: add demo seed cli command (`dfbf7fe`)
- feat: add open-core setup wizard and self-hosted deployment docs (`81a75aa`)
- feat: add pnpm docker shortcuts (docker:dev, docker:prod, docker:down) (`011d7b8`)
- feat: scaffold Scout project with Svelte 4 + Vite (`99b634c`)

### Fixes

- fix(docs): satisfy open-core release surface checks (`1641b2c`)
- fix(onboarding): harden starter CTA with trim/guard and strengthen test with interaction assertions (`5a517ab`)
- fix(onboarding): point wizard test at open-core component; add show/hide and completion event (`52277b7`)
- fix(smoke): address Day 2 review quality issues (`b471370`)
- fix(ui): correct tooltip positioning in collapsed rail navigation (`d8f0e13`)
- fix(ProjectSelector): replace chevron text with SVG icon and improve dropdown width styling (`4fa92d6`)
- fix(App): improve empty list layout and TourCallout styling (`26844f4`)
- fix(AppShell): restore flex layout via CSS after Tailwind class regression (`da51c88`)
- fix: gate EmptyState on hasLoadedCatalog and remove global CSS leaks in ModelConfig (`b6375fd`)
- fix(AppShell): make rail responsive on mobile, improve insight pane semantics (`f776d7f`)
- fix: add type fallback and readable clampCount helper to LoadingSkeleton (`eb9d5be`)
- fix: use typeof guard before Number() to truly prevent Symbol coercion throws in LoadingSkeleton (`f9d0a64`)
- fix: use parseFloat instead of Number() to guard against Symbol coercion throws in LoadingSkeleton (`25685a6`)
- fix: clamp LoadingSkeleton count to safe integer to prevent Array() throws (`8fafe62`)
- fix: restore UI primitives to approved plan spec (revert quality drift) (`5db06a7`)
- fix: address code-quality issues in Surface, EmptyState, LoadingSkeleton UI primitives (`1a3d5b7`)
- fix(web): narrow full-width-view section selector and merge detail-panel rules (`ad004a8`)
- fix: update empty-state copy to reference workspace rail (`a6338c7`)
- fix: correct Sources nav item to navigate to 'sources', add type=button to nav items (`06394a4`)
- fix(onboarding): show full setup review in step 4 (`d6de29a`)
- fix(onboarding): allow re-saving completed steps (`e5359af`)
- fix(onboarding): show resolved database path in setup step (`52f1796`)
- fix(web): rename Anthropic option and add Claude CLI provider in onboarding (`0fca3c0`)
- fix(web): clarify onboarding AI provider options (`229c374`)
- fix: correct Go module paths in Docker build (`24697c0`)
- fix(onboarding): enforce strict linear flow in SaveRequiredStep and Save (`3ddc2a5`)
- fix(onboarding): tighten Task 6 SaveRequiredStep and Save transitions per spec (`9ffdfbe`)
- fix(onboarding): Task 5 code-quality fixes in service.go (`5e69816`)
- fix(onboarding): align Task 5 service to approved plan contract (`2919818`)
- fix(onboarding): align Task 3 state model exactly to approved plan spec (`bdca769`)
- fix(onboarding): update config_test.go comments and add StateKey to fields check (`7687a80`)
- fix(onboarding): add config validation in NewService, trim whitespace values (`63f6ed0`)
- fix(configfile): prevent duplicate managed section marker on repeated UpdateFile calls (`958180a`)
- fix: register CLI runtimes in bridge daemon and fix bootstrap script (`94eefb0`)
- fix(ai): treat BridgeState.Runtimes as advisory; live health is authoritative (`c89ed55`)
- fix(ai): forward catalog provider tag to bridge in invokeModelWithBridge (`00fe08a`)
- fix(ai-test): use real *BridgeProvider in SDK/Gemini no-bridge routing tests (`45c0fe6`)
- fix(landing): use shared logo favicon assets (`7628afd`)
- fix(docker): source .env before compose so PARALLEL_API_KEY passes to containers (`e4ef526`)
- fix(docs): replace docker compose profile restart with pnpm docker:down/docker:dev commands (`f8919f2`)
- fix(ui): remove runtime banner from app shell (`ed44500`)
- fix(docs): remove redundant h1 from index page, Starlight auto-generates from title (`2ec75fd`)
- fix(docs): replace invalid title placeholders in Starlight frontmatter (`9d76665`)
- fix(docs): hoist @astrojs/* to resolve internal-helpers ESM conflict (`acdeb17`)
- fix(actions): publish open-core to master branch (`f71d86f`)
- fix(actions): use full refname when pushing open-core split (`fbcd31d`)
- fix(actions): prevent checkout from persisting bot credentials (`fb272e3`)
- fix: repair failing test suite - docker scripts, port conflict, db test exclusion (`2ca2e9a`)
- fix(web): address review findings for capability UI (`31b3e0e`)
- fix(docker): mount db package in dev stage for go run (`2c4ffbb`)
- fix(docker): update Go to 1.25 and include db module for dependency resolution (`5f996ae`)
- fix(task-9): delegate core migrations to Go module; SaaS runner is additive-only (`970c271`)
- fix(db/config): harden SaaS dialect/URL validation and remove broken scripts (`dcd7f71`)
- fix(db): replace driver-identity detection with SQL probe queries in dialectFromDB (`fa34348`)
- fix(db): replace race-prone WHERE NOT EXISTS with atomic INSERT OR IGNORE / ON CONFLICT; use driver-name-based dialect detection (`28405b5`)
- fix(db): make EnsureCoreSeedMarker compatible with both SQLite and Postgres (`a8c5aea`)
- fix: restore EnsureCoreSeedMarker to approved contract (no dialect param) (`b565b3d`)
- fix(db): strengthen wire_test coverage and fence off legacy sqlite.go (`10df73e`)
- fix(db): safe SQLite DSN path encoding and expand pragma/postgres tests (`c1e7b03`)
- fix: remove DEFAULT 'core' from schema_migrations.scope DDL for SQLite and Postgres (`08522b2`)
- fix(db): align migrate.go and postgres SQL with approved spec (`ebccd60`)
- fix(db): revert go.mod/go.sum module-direct promotion from cleanup commit (`16c83f7`)
- fix(db): restore blank driver imports and tighten idempotency assertion in migrate_test (`f7b1ddc`)
- fix(db): tighten LoadConfigFromEnv and ResolveDefaultSQLitePath to match approved contract (`73e34a4`)
- fix: expose ResolveDefaultSQLitePath() public API with cwd injection and fallback (`ed2a3b6`)
- fix: resolve open-core docker dev/prod runtime issues (`17e9ba4`)
- fix(workspace): remove speculative open-core/apps/web and open-core/packages/* globs (`68b1dd3`)
- fix(task-2): add turbo devDependency to open-core workspace root (`7aff2d6`)
- fix: refine google reports UX state and frontend test coverage (`6377692`)
- fix: auto-kill stale server on dev start and clean up on exit (`8129f0e`)

### Documentation

- docs(threadlens): fix public repo setup paths (`5f88a01`)
- docs(threadlens): remove open-core branding from public docs (`cb151d5`)
- docs(threadlens): point self-host quick start to installer (`b577381`)
- docs: add local VitePress entrypoint (`b97cde6`)
- docs: consolidate open-core sync guidance (`8ebbaac`)
- docs: expand issue-reproductions Include list with smoke test and wizard steps (Day 9.3) (`8525803`)
- docs: add Self-Host Troubleshooting link to quick-start stop section (`6134035`)
- docs: add self-host troubleshooting reference page (Day 9.1) (`f73a9e3`)
- docs: add first-value guide and activation wording (`568e15e`)
- docs: add Fastest first-value path section to quick-start (Day 8.3) (`c32b6c5`)
- docs(open-core): link first-value-in-15-minutes guide from Docker quick start (`78b6b33`)
- docs(open-core): add onboarding guidance paragraph after web service table row (`5ac2fbe`)
- docs(open-core): add self-host smoke-check step to quick start (`07ec105`)
- docs: add local AI bridge guide and update provider configuration (`537981c`)
- docs: update redesign spec and phase plans (`9919150`)
- docs: document ThreadLens onboarding flow (`8c828e2`)
- docs(open-core): add onboarding env var notes to reference and config-basics docs (`2f77d61`)
- docs: document onboarding env variables in .env.example (`80210b6`)
- docs: clarify bridge optionality for self-hosting (`760cd83`)
- docs: document hybrid AI runtime; bridge is optional local-only transport (`d111a09`)
- docs: describe daemon bootstrap flow (`051d78e`)
- docs: add bridge env example entries (`8548766`)
- docs: describe host bridge fallback path (`a463fd5`)
- docs: clarify quick start hybrid provider wording (`766e455`)
- docs: clarify AI provider fallback readiness (`c7e3cd1`)
- docs: clarify AI provider fallback readiness (`bf3224f`)
- docs: clarify AI provider fallback readiness (`9fdb3fa`)
- docs: clarify hybrid provider policy (`02c0dce`)
- docs: clarify hybrid AI provider guidance (`f091297`)
- docs: connect Docker reference to first-run path (`689fc96`)
- docs: add environment variable capability context (`b568243`)
- docs: add environment variable capability context (`af050cf`)
- docs: clarify AI provider fallback readiness (`f85e63b`)
- docs: document report prerequisites (`84c21d1`)
- docs: clarify scouting source readiness (`c280ece`)
- docs: clarify scouting source readiness (`a4fc883`)
- docs: align AI provider readiness guidance (`49e355c`)
- docs: clarify UI click-path and slug/display-name wording in first-project walkthrough (`bdbb982`)
- docs: expand first project walkthrough (`f1f7b3e`)
- docs: add Docker restart instruction after .env edits in configuration-basics (`86c0316`)
- docs: explain first-run provider setup (`1f46f77`)
- docs: make quick start Docker-first (`3bd7819`)
- docs: preview first ThreadLens session (`569797f`)
- docs: clarify first-run homepage path (`006554e`)
- docs(contributing): add open-core contribution terms (`aca1393`)
- docs(licensing): adopt BUSL for open-core (`58bd7c4`)
- docs(open-core): normalize title in frontmatter, remove redundant H1 (`e08ea69`)
- docs: link READMEs to canonical docs.threadlens.dev pages (`ce1502d`)
- docs: add Cloudflare Pages wrangler config and docs README (`1aa2f43`)
- docs: seed Contributing and Maintainer Notes sections for open-core (`ed0bc2c`)
- docs: seed Architecture section with six open-core pages (`f897710`)
- docs: seed Reference section with env vars, ports, Docker, API shape, and storage pages (`3fe8ba7`)
- docs: seed User Guide section with six ThreadLens topic pages (`bce2ea1`)
- docs: seed Start Here migration pages for ThreadLens open-core (`f41aad5`)
- docs: create ThreadLens docs homepage and remove .gitkeep marker (`9e2d747`)
- docs: add Astro Starlight config, content collection, native CSS brand styling, and favicon (`a4f70ff`)
- docs(threadlens): update shared web copy to ThreadLens (`86a7913`)
- docs(threadlens): update web metadata branding (`78fcab6`)
- docs(threadlens): rebrand open-core docker docs (`545a295`)
- docs(threadlens): rewrite open-core README for ThreadLens (`48055c1`)
- docs: document open-core docker commands (`c78438b`)
- docs: freeze open-core/SaaS boundary in README, monorepo docs, and package.json scripts (`ff0a0fb`)

### Maintenance

- Day 6 code-quality: deterministic flush helper, less-coupled selectors, extract fallback constants (`fc64e2a`)
- test: add regression coverage for Day 2 smoke-check fixes (`810622e`)
- add self-host smoke dry-run shell test (Task 2.4) (`1ceee23`)
- refactor(bridge): gate bridge env vars on runtime availability (`0cfe808`)
- Graceful notice when AI providers unavailable in query suggest (`439df1b`)
- Harden AI JSON parsing against CLI prefix noise in suggest and refine flows (`883e2a0`)
- Fix bridge health runtimes parsing to support object format (`244add8`)
- Fix bridge CLI runtime detection: use non-empty probe prompt and improve auth error normalization (`e3ded5e`)
- style: apply dark token baseline to full-width views (`02666c0`)
- refactor(App): extract navigateTo helper, make sources a real view state (`cf78a6f`)
- chore(web): remove unused ProjectSelector import from App.svelte (`67b49fc`)
- refactor(web): replace header/main with AppShell layout in App.svelte (`7f00aac`)
- Add design tokens CSS and import globally in main.js (`aad88fe`)
- refactor(api): dedupe handler helpers and drop redundant work in onboarding (`39d6f8a`)
- refactor(web): parallelize app init and drop unused capability error state (`82ff2b8`)
- chore: remove dead Express Docker infrastructure (`5ef9ffb`)
- chore: cut over dev scripts and docker config to Go backend (`aa232f5`)
- test(onboarding): rewrite starter_test.go to match approved Task 8 contract (`5def70f`)
- test(onboarding): add starter project orchestration tests (Task 8) (`ee18c52`)
- onboarding: fix exploration reopen/completion-timestamp edge cases (Task 7 QA) (`1440b90`)
- onboarding: fix Task 7 spec gaps in UpdateExploration and Reset (`6404d99`)
- onboarding: align Task 7 exploration update and reset modes with spec (`adf0426`)
- test(onboarding): align Task 4 service tests to approved plan spec (`c9029c9`)
- test(onboarding): rewrite service tests for v1 phase-based model (`f09f943`)
- test(onboarding): strengthen state_test.go map lookup and JSON positive assertion (`32a8d99`)
- test(onboarding): fix state_test.go to match approved plan symbols (`a81e0e1`)
- test(onboarding): add state model tests for Task 2 (`b977136`)
- test(onboarding): strengthen handler tests per review feedback (`5b5e8dd`)
- test(onboarding): add failing HTTP handler tests for onboarding endpoints (`c0e3cc6`)
- test(onboarding): add failing service tests before implementation (`e50ef9c`)
- test(onboarding): add failing config loader tests (Task 4) (`d62c01f`)
- task 7: add explicit local mode and camelCase config to AI bridge bootstrap (`c78217a`)
- sanitize aggregate failures in GenerateForTask to omit raw provider errors (`dedc4e0`)
- refactor(ai): make bridge optional transport field on Service, not a direct provider (`51688f1`)
- test(ai): add service tests locking fallback order and bridge-as-transport behavior (`3eda72a`)
- test(bridge): add bootstrap script tests (red) (`a656954`)
- test(bridge): add server tests for auth and limits (`afb681a`)
- test(bridge): add CLIRuntime tests (Task 6, red phase) (`1f236f4`)
- test(bridge): add registry_test.go with cache and dispatch tests (red) (`1194b0b`)
- test(bridge): add bind address validation tests (red) (`d96d149`)
- chore: add landing:dev script for local landing page development (`3e61f56`)
- chore: register apps/landing workspace package in pnpm-lock.yaml (`5d96566`)
- ci(docs): add GitHub Actions pipeline to deploy to Cloudflare Pages (`04c7899`)
- Register docs scripts in open-core and root workspaces (`461a80e`)
- Add @threadlens/docs workspace package manifest, gitignore, and tsconfig (`d010fa0`)
- chore(ci): add open-core subtree publish workflow (`9d0aac3`)
- test(handlers): wire entitlement resolver in handler tests (Task 33) (`62179b1`)
- wire RuntimeMode, entitlements, modules, templates, usage into app and routes (`9e344ad`)
- task(30): route /api/models/catalog through svc.Catalog() instead of ai package globals (`f1baa5c`)
- task 29: add entitlement-aware Catalog method to ModelService (`9cf6c15`)
- task 28: inject entitlement checks into ReportService.StartReport (`ab1e56a`)
- task 27: inject entitlement checks into ScoutService.StartRun (`6ff33c1`)
- test(ai): add usage metering coverage for GenerateForTask (Task 32) (`89cbae8`)
- test(config): add runtime mode coverage to config_test.go (Task 22) (`3847590`)
- test(handlers): add runtime capabilities and prompt-pack handler tests (Task 19) (`1cdb1ac`)
- test(entitlements): add unit tests for local resolver and denial behavior (`e00361c`)
- chore(deps): add turbo for build orchestration (`4f554d5`)
- refactor(env): consolidate env templates with shared + SaaS overlay model (`d2b2922`)
- chore(db): remove local gitignore in favor of open-core root (`110d609`)
- chore(db): ignore compiled migrate binary in open-root (`f4629fa`)
- chore(db): ignore compiled migrate binary (`4f8e128`)
- test(db): deduplicate expected table list and trim go.mod (`e56e14a`)
- test: rewrite core migration contract to match approved spec (`1ee81e3`)
- test: add pgx stdlib import for postgres migration test (`c23a3b4`)
- test: define core migration contract (`ee7afd0`)
- refactor(db): replace mutable cwdResolver seam with getwd function injection (`f120dcb`)
- refactor(db): use safe cwd-injection seam instead of os.Chdir in tests (`3cc9ef0`)
- test: harden db config contract — remove os.Chdir, use fixture helper, drop premature driver deps (`e95b995`)
- test: define shared database config contract (`25e08e6`)
- chore: keep open-core staging aligned with public source (`b4b3615`)
- chore: stage open-core web workspace packages (`a9af17c`)
- refine contract test: use YAML parsing for workspace and Compose assertions (`430138e`)
- task(8): add workspace filters and Turbo passthrough tasks for open-core/Go builds (`f4755f1`)
- task 6: move shared Docker base to open-core, reduce private to thin overlays (`94f6253`)
- park Express backend: move apps/api → apps/legacy-express (`0213a6f`)
- task(3): delete apps/api-go and update all references to open-core/apps/api (`127deb7`)
- task(3): move Go backend from apps/api-go to open-core/apps/api (`aca9333`)
- chore(open-core): add .gitignore for standalone repo (`4948cc7`)
- task(2): scaffold open-core skeleton with workspace config and directory stubs (`cac8da3`)
- task(1): route root 'server' script through open-core passthrough wrapper (`7638a70`)
- test(api-go): add full api parity suite (`823c733`)
- chore(api-go): initialize go backend module (`8ec1e23`)
- chore(web): migrate frontend tooling and docker web service to bun (`d16a633`)
- chore: upgrade frontend to Svelte 5 toolchain (`628b0fe`)
- chore: standardize workspace on pnpm (`71f59be`)
- chore: convert repo to npm workspaces (`794d079`)
- refactor(google): extract signals module, improve pain theme extraction, and harden dev workflow (`61980f9`)
- chore: add launchctl scripts, logs gitignore, and dev server port (`d04a634`)

## threadlens (open-core)

Release entries are maintained automatically by the release command. Run `pnpm run release -- --track open-core` from the monorepo root to cut a new public GitHub release and append an entry here.
