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

<main>
  <header>
    <img src="/icon.svg" alt="" class="logo" />
    <h1>Tomoko</h1>
  </header>

  {#if loading}
    <p class="muted">Loading…</p>
  {:else if error}
    <p class="error">Failed to reach Sophon: {error}</p>
  {:else if roots.length === 0}
    <p class="muted">
      No projects or contexts yet. Create your first one from the command bar
      once command parsing is wired, or seed via the API.
    </p>
  {:else}
    <Tree nodes={roots} />
  {/if}

  {#if notice}
    <div class="notice">{notice}</div>
  {/if}
</main>

<CommandBar onsubmit={onCommand} />

<style>
  main {
    max-width: 40rem;
    margin: 0 auto;
    padding: 1rem 1rem 6rem;
  }
  header {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    padding: 0.5rem 0 1rem;
  }
  .logo {
    width: 1.75rem;
    height: 1.75rem;
  }
  h1 {
    font-size: 1.1rem;
    font-weight: 600;
    letter-spacing: 0.04em;
    margin: 0;
  }
  .muted {
    color: var(--fg-muted);
  }
  .error {
    color: var(--danger);
  }
  .notice {
    position: fixed;
    bottom: 5rem;
    left: 50%;
    transform: translateX(-50%);
    background: var(--surface-2);
    border: 1px solid var(--border);
    border-radius: 0.5rem;
    padding: 0.5rem 0.9rem;
    font-size: 0.85rem;
    max-width: 90vw;
  }
</style>
