<script lang="ts">
  import { api, type Task } from '../lib/api';
  import TaskRow from '../lib/TaskRow.svelte';

  let { view }: { view: 'today' | 'upcoming' | 'inbox' } = $props();

  let tasks = $state<Task[]>([]);
  let loaded = $state(false);
  let error = $state<string | null>(null);

  const titles = { today: 'Today', upcoming: 'Upcoming', inbox: 'Inbox' };

  $effect(() => {
    loaded = false;
    api
      .view(view)
      .then((r) => {
        tasks = r.tasks;
        loaded = true;
      })
      .catch((e: Error) => (error = e.message));
  });
</script>

<div class="flex flex-col gap-3">
  <h2 class="glow-text px-2 text-lg font-bold">{titles[view]}</h2>
  {#if error}
    <p class="text-danger">{error}</p>
  {:else if !loaded}
    <p class="text-fg-muted">Loading…</p>
  {:else if tasks.length === 0}
    <p class="text-fg-muted px-2 text-sm">
      {view === 'inbox' ? 'Inbox zero.' : 'Nothing here. Clear skies.'}
    </p>
  {:else}
    {#each tasks as t (t.id)}
      <TaskRow task={t} />
    {/each}
  {/if}
</div>
