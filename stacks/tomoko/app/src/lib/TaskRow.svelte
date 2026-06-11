<script lang="ts">
  import { api, type Task } from './api';
  import { dueLabel } from './dates';
  import { router } from './router.svelte';

  let { task, showProject = true }: { task: Task; showProject?: boolean } = $props();

  // Derived from the prop, reassignable for optimistic toggling.
  let done = $derived(task.status === 'done');
  const due = $derived(dueLabel(task.due_at));

  async function toggle(e: Event) {
    e.stopPropagation();
    done = !done;
    try {
      await api.updateTask(task.id, { status: done ? 'done' : 'open' });
    } catch {
      done = !done; // revert on failure
    }
  }
</script>

<div
  class="hover:glass group flex w-full cursor-pointer items-center gap-3 rounded-md px-2 py-2.5
         transition-shadow hover:shadow-glow-sm"
  role="button"
  tabindex="0"
  onclick={() => router.go(`/t/${task.id}`)}
  onkeydown={(e) => e.key === 'Enter' && router.go(`/t/${task.id}`)}
>
  <button
    type="button"
    aria-label={done ? 'reopen task' : 'complete task'}
    class={[
      'grid size-5 shrink-0 cursor-pointer place-items-center rounded-full border text-[0.6rem] transition-colors',
      done
        ? 'border-ember bg-ember text-space'
        : 'border-line group-hover:border-ember text-transparent',
    ]}
    onclick={toggle}
  >
    ✓
  </button>

  <span class={['flex-1 truncate text-[0.95rem]', done && 'text-fg-muted line-through']}>
    {task.title}
  </span>

  {#if task.priority > 0}
    <span class="text-ember-bright text-xs">{'!'.repeat(task.priority)}</span>
  {/if}
  {#if showProject && task.project_name}
    <span class="text-fg-muted hidden text-xs sm:inline">{task.project_name}</span>
  {/if}
  {#if due}
    <span
      class={[
        'rounded-sm px-1.5 py-0.5 text-xs tabular-nums',
        due.tone === 'overdue' && 'bg-danger/15 text-danger',
        due.tone === 'today' && 'bg-ember/15 text-ember-bright',
        due.tone === 'future' && 'text-fg-muted',
      ]}
    >
      {due.text}
    </span>
  {/if}
</div>
