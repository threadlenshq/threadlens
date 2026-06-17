import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

export default defineConfig({
  site: 'https://docs.threadlens.dev',
  integrations: [
    starlight({
      title: 'ThreadLens Docs',
      description: 'Self-host ThreadLens and understand its open-core research intelligence workflow.',
      favicon: '/favicon.svg',
      customCss: ['./src/styles/native.css'],
      editLink: {
        baseUrl: 'https://github.com/threadlenshq/threadlens/edit/main/open-core/docs/',
      },
      social: [
        { icon: 'rocket', label: 'ThreadLens', href: 'https://threadlens.dev' },
        { icon: 'github', label: 'GitHub', href: 'https://github.com/threadlenshq/threadlens' },
      ],
      sidebar: [
        { label: 'Docs Home', link: '/' },
        {
          label: 'Start Here',
          items: [
            { slug: 'start-here/overview' },
            { slug: 'start-here/quick-start' },
            { slug: 'start-here/configuration-basics' },
            { slug: 'start-here/first-project' },
            { slug: 'start-here/first-value-in-15-minutes' },
          ],
        },
        {
          label: 'User Guide',
          items: [
            { slug: 'user-guide/projects-and-modes' },
            { slug: 'user-guide/scouting-sources' },
            { slug: 'user-guide/scoring-filtering-and-statuses' },
            { slug: 'user-guide/dm-targets-and-profile-scoring' },
            { slug: 'user-guide/reports' },
            { slug: 'user-guide/schedules' },
            { slug: 'user-guide/model-provider-configuration' },
          ],
        },
        {
          label: 'Reference',
          items: [
            { slug: 'reference/licensing' },
            { slug: 'reference/environment-variables' },
            { slug: 'reference/credential-setup' },
            { slug: 'reference/local-ai-bridge' },
            { slug: 'reference/ports-and-local-urls' },
            { slug: 'reference/docker-commands-and-profiles' },
            { slug: 'reference/api-shape-overview' },
            { slug: 'reference/data-storage-and-backups' },
            { slug: 'reference/self-host-troubleshooting' },
            { slug: 'reference/telemetry' },
          ],
        },
        {
          label: 'Architecture',
          items: [
            { slug: 'architecture/workspace-layout' },
            { slug: 'architecture/go-api' },
            { slug: 'architecture/web-app' },
            { slug: 'architecture/pipelines' },
            { slug: 'architecture/ai-providers' },
            { slug: 'architecture/open-core-and-hosted-boundaries' },
          ],
        },
        {
          label: 'Contributing',
          items: [
            { slug: 'contributing/development-setup' },
            { slug: 'contributing/testing' },
            { slug: 'contributing/docs-contributions' },
            { slug: 'contributing/issue-reproductions' },
            { slug: 'contributing/package-boundaries' },
          ],
        },
        {
          label: 'Maintainer Notes',
          items: [
            { slug: 'maintainer-notes/open-core-procedures' },
            { slug: 'maintainer-notes/releases-docker-and-support' },
          ],
        },
      ],
      credits: true,
      head: [
        { tag: 'meta', attrs: { name: 'theme-color', content: '#185a54' } },
      ],
    }),
  ],
});
