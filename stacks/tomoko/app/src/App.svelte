<script lang="ts">
  import Tree from './lib/Tree.svelte';
  import CommandBar from './lib/CommandBar.svelte';
  import { fetchTree, type TreeNode } from './lib/api';

  let roots = $state<TreeNode[]>([]);
  let error = $state<string | null>(null);
  let loading = $state(true);
  let notice = $state<string | null>(null);

  $effect(() => {
    fetchTree()
      .then((t) => (roots = t.roots))
      .catch((e: Error) => (error = e.message))
      .finally(() => (loading = false));
  });

  function onCommand(input: string) {
    // Step 5 wires this to POST /api/command (retrieval-only intent parsing).
    notice = `Command parsing lands in the next phase — received: “${input}”`;
    setTimeout(() => (notice = null), 4000);
  }
</script>

<main class="mx-auto max-w-2xl px-4 pt-3 pb-28">
  <header class="flex items-center py-3">
    <img src="/tomoko_logo.svg" alt="Tomoko" class="glow-drop h-12 w-auto" />
  </header>

  {#if loading}
    <p class="text-fg-muted">Loading…</p>
  {:else if error}
    <p class="text-danger">Failed to reach Sophon: {error}</p>
  {:else if roots.length === 0}
    <p class="text-fg-muted">
      No projects or contexts yet. Create your first one from the command bar
      once command parsing is wired, or seed via the API.
    </p>
  {:else}
    <Tree nodes={roots} />
  {/if}

  {#if notice}
    <div
      class="glass shadow-glow-sm fixed bottom-24 left-1/2 max-w-[90vw] -translate-x-1/2 rounded-lg px-4 py-2 text-sm"
    >
      {notice}
    </div>
  {/if}
</main>

<CommandBar onsubmit={onCommand} />
