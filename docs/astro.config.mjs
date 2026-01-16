import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import vue from '@astrojs/vue';

export default defineConfig({
  site: 'https://felixgeelhaar.github.io',
  base: '/orbita/docs',
  integrations: [
    starlight({
      title: 'Orbita',
      description: 'CLI-first adaptive productivity operating system',
      logo: {
        src: './src/assets/logo.svg',
        replacesTitle: false,
      },
      social: {
        github: 'https://github.com/felixgeelhaar/orbita',
      },
      editLink: {
        baseUrl: 'https://github.com/felixgeelhaar/orbita/edit/main/docs/',
      },
      customCss: [
        './src/styles/custom.css',
      ],
      sidebar: [
        {
          label: 'Getting Started',
          items: [
            { label: 'Introduction', slug: 'getting-started/introduction' },
            { label: 'Installation', slug: 'getting-started/installation' },
            { label: 'Quick Start', slug: 'getting-started/quickstart' },
            { label: 'Configuration', slug: 'getting-started/configuration' },
          ],
        },
        {
          label: 'Features',
          items: [
            { label: 'Overview', slug: 'features/overview' },
            { label: 'Task Management', slug: 'features/tasks' },
            { label: 'Smart Scheduling', slug: 'features/scheduling' },
            { label: 'Calendar Sync', slug: 'features/calendar' },
            { label: 'Habit Tracking', slug: 'features/habits' },
            { label: '1:1 Meetings', slug: 'features/meetings' },
            { label: 'AI Inbox', slug: 'features/inbox' },
            { label: 'Automations', slug: 'features/automations' },
            { label: 'Time Insights', slug: 'features/insights' },
          ],
        },
        {
          label: 'CLI Reference',
          items: [
            { label: 'Overview', slug: 'cli/overview' },
            { label: 'orbita task', slug: 'cli/task' },
            { label: 'orbita schedule', slug: 'cli/schedule' },
            { label: 'orbita calendar', slug: 'cli/calendar' },
            { label: 'orbita habit', slug: 'cli/habit' },
            { label: 'orbita meeting', slug: 'cli/meeting' },
            { label: 'orbita inbox', slug: 'cli/inbox' },
            { label: 'orbita mcp', slug: 'cli/mcp' },
          ],
        },
        {
          label: 'Guides',
          items: [
            { label: 'Daily Planning', slug: 'guides/daily-planning' },
            { label: 'Weekly Review', slug: 'guides/weekly-review' },
            { label: 'AI Integration', slug: 'guides/ai-integration' },
            { label: 'Conflict Resolution', slug: 'guides/conflict-resolution' },
          ],
        },
        {
          label: 'Developers',
          items: [
            { label: 'Architecture', slug: 'developers/architecture' },
            { label: 'Engine SDK', slug: 'developers/engine-sdk' },
            { label: 'Orbit SDK', slug: 'developers/orbit-sdk' },
            { label: 'MCP Integration', slug: 'developers/mcp' },
            { label: 'Contributing', slug: 'developers/contributing' },
          ],
        },
      ],
      components: {
        // Override components with Vue when needed
      },
    }),
    vue(),
  ],
});
