<script lang="ts">
  import { api, type Folder, type Task } from '../lib/api';
  import { isDeferred } from '../lib/dates';
  import { router } from '../lib/router.svelte';
  import TaskRow from '../lib/TaskRow.svelte';

  let { id }: { id: number } = $props();

  let folder = $state<Folder | null>(null);
  let error = $state<string | null>(null);
  let showDeferred = $state(false);

  $effect(() => {
    folder = null;
    api
      .folder(id)
      .then((f) => (folder = f))
      .catch((e: Error) => (error = e.message));
  });

  // Grouping: overdue / today / upcoming / someday — deferred hidden (tickler)
  function groups(tasks: Task[]) {
    const now = new Date();
    const endOfToday = new Date(now.getFullYear(), now.getMonth(), now.getDate(), 23, 59, 59);
    const visible = tasks.filter((t) => showDeferred || !isDeferred(t.defer_at));
    const overdue: Task[] = [], today: Task[] = [], upcoming: Task[] = [], someday: Task[] = [];
    for (const t of visible) {
      if (!t.due_at) someday.push(t);
      else if (new Date(t.due_at) < now && new Date(t.due_at) < endOfToday) overdue.push(t);
      else if (new Date(t.due_at) <= endOfToday) today.push(t);
      else upcoming.push(t);
    }
    return [
      ['Overdue', overdue],
      ['Today', today],
      ['Upcoming', upcoming],
      ['Someday', someday],
    ] as const;
  }

  const deferredCount = $derived(
    folder ? folder.tasks.filter((t) => isDeferred(t.defer_at)).length : 0,
  );
</script>

{#if error}
  <p class="text-danger">{error}</p>
{:else if !folder}
  <p class="text-fg-muted">Loading…</p>
{:else}
  <div class="flex flex-col gap-5">
    <header class="flex items-baseline gap-2.5">
      <span class={folder.tag.kind === 'project' ? 'text-ember' : 'text-ember-bright'}>
        {folder.tag.kind === 'project' ? '▣' : '◈'}
      </span>
      <h2 class="glow-text text-lg font-bold">{folder.tag.name}</h2>
      <span class="text-fg-muted text-xs">{folder.tag.kind}</span>
    </header>

    {#if folder.tag.description}
      <p class="text-fg-muted -mt-3 text-sm">{folder.tag.description}</p>
    {/if}

    {#if folder.children.length > 0}
      <section class="flex flex-col gap-1">
        {#each folder.children as c (c.id)}
          <button
            type="button"
            class="hover:glass flex w-full cursor-pointer items-center gap-2.5 rounded-md px-2
                   py-2 text-left transition-shadow hover:shadow-glow-sm"
            onclick={() => router.go(`/p/${c.id}`)}
          >
            <span class={['text-xs', c.kind === 'project' ? 'text-ember' : 'text-ember-bright']}>
              {c.kind === 'project' ? '▣' : '◈'}
            </span>
            <span>{c.name}</span>
          </button>
        {/each}
      </section>
    {/if}

    {#each groups(folder.tasks) as [label, tasks] (label)}
      {#if tasks.length > 0}
        <section>
          <h3 class="text-fg-muted mb-1 px-2 text-xs tracking-wider uppercase">{label}</h3>
          {#each tasks as t (t.id)}
            <TaskRow task={t} showProject={false} />
          {/each}
        </section>
      {/if}
    {/each}

    {#if deferredCount > 0}
      <button
        type="button"
        class="text-fg-muted hover:text-fg cursor-pointer px-2 text-left text-xs"
        onclick={() => (showDeferred = !showDeferred)}
      >
        {showDeferred ? 'hide' : 'show'} {deferredCount} deferred
      </button>
    {/if}

    {#if folder.notes.length > 0}
      <section>
        <h3 class="text-fg-muted mb-1 px-2 text-xs tracking-wider uppercase">Notes</h3>
        {#each folder.notes as n (n.id)}
          <button
            type="button"
            class="hover:glass flex w-full cursor-pointer items-center gap-2.5 rounded-md px-2
                   py-2 text-left transition-shadow hover:shadow-glow-sm"
            onclick={() => router.go(`/n/${n.id}`)}
          >
            <span class="text-ember text-sm">≣</span>
            <span class="truncate">{n.title}</span>
          </button>
        {/each}
      </section>
    {/if}

    {#if folder.tasks.length === 0 && folder.notes.length === 0 && folder.children.length === 0}
      <p class="text-fg-muted px-2 text-sm">Empty — capture something from the command bar.</p>
    {/if}
  </div>
{/if}
