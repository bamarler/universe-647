<script lang="ts">
  import { api, type TreeNode } from '../lib/api';
  import { router } from '../lib/router.svelte';
  import TreeList from '../lib/TreeList.svelte';

  let roots = $state<TreeNode[]>([]);
  let error = $state<string | null>(null);
  let loading = $state(true);

  $effect(() => {
    api
      .tree()
      .then((t) => (roots = t.roots))
      .catch((e: Error) => (error = e.message))
      .finally(() => (loading = false));
  });

  const views = [
    { path: '/view/today', label: 'Today', glyph: '☀' },
    { path: '/view/upcoming', label: 'Upcoming', glyph: '↗' },
    { path: '/view/inbox', label: 'Inbox', glyph: '⊡' },
  ];
</script>

<!-- Mobile home: smart views + the tree. Desktop redirects to /view/today. -->
<div class="flex flex-col gap-5">
  <div class="grid grid-cols-3 gap-2">
    {#each views as v (v.path)}
      <button
        type="button"
        class="glass hover:shadow-glow-sm flex cursor-pointer flex-col items-center gap-1
               rounded-xl py-3 transition-shadow"
        onclick={() => router.go(v.path)}
      >
        <span class="text-ember text-lg">{v.glyph}</span>
        <span class="text-sm">{v.label}</span>
      </button>
    {/each}
  </div>

  {#if loading}
    <p class="text-fg-muted">Loading…</p>
  {:else if error}
    <p class="text-danger">Failed to reach Sophon: {error}</p>
  {:else if roots.length === 0}
    <p class="text-fg-muted px-1 text-sm">
      No projects or contexts yet — try “new project called universe-647” below.
    </p>
  {:else}
    <TreeList nodes={roots} />
  {/if}
</div>
