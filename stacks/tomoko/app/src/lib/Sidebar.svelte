<script lang="ts">
  import { api, type TreeNode } from './api';
  import { router } from './router.svelte';
  import TreeList from './TreeList.svelte';

  // Desktop-only persistent navigation. Mobile gets the drill-in HomePage.
  let { onOmnibar }: { onOmnibar: () => void } = $props();

  let roots = $state<TreeNode[]>([]);

  // Refetch when navigation happens (counts change as tasks are created).
  $effect(() => {
    void router.path;
    api.tree().then((t) => (roots = t.roots));
  });

  const views = [
    { path: '/view/today', label: 'Today', glyph: '☀' },
    { path: '/view/upcoming', label: 'Upcoming', glyph: '↗' },
    { path: '/view/inbox', label: 'Inbox', glyph: '⊡' },
  ];
</script>

<aside class="border-line/50 flex h-dvh w-64 shrink-0 flex-col gap-4 border-r p-4">
  <button type="button" class="cursor-pointer" onclick={() => router.go('/view/today')}>
    <img src="/tomoko_logo.svg" alt="Tomoko" class="glow-drop h-10 w-auto" />
  </button>

  <nav class="flex flex-col gap-0.5">
    {#each views as v (v.path)}
      <button
        type="button"
        class={[
          'flex cursor-pointer items-center gap-2.5 rounded-md px-2 py-1.5 text-left text-sm',
          router.path === v.path ? 'glass shadow-glow-sm' : 'hover:glass',
        ]}
        onclick={() => router.go(v.path)}
      >
        <span class="text-ember">{v.glyph}</span>
        {v.label}
      </button>
    {/each}
  </nav>

  <div class="border-line/50 -mx-1 border-t"></div>

  <div class="min-h-0 flex-1 overflow-y-auto">
    {#if roots.length === 0}
      <p class="text-fg-muted px-2 text-xs">No folders yet.</p>
    {:else}
      <TreeList nodes={roots} />
    {/if}
  </div>

  <button
    type="button"
    class="glass hover:shadow-glow-sm text-fg-muted cursor-pointer rounded-lg px-3 py-2
           text-left text-sm transition-shadow"
    onclick={onOmnibar}
  >
    Ask, create, or find… <kbd class="text-ember float-right">⌘K</kbd>
  </button>
</aside>
