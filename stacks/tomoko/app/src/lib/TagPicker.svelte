<script lang="ts">
  import { api, type DraftTag, type Tag } from './api';

  let {
    selected = $bindable(),
  }: {
    selected: DraftTag[];
  } = $props();

  let all = $state<Tag[]>([]);
  let input = $state('');

  $effect(() => {
    api.tags().then((r) => (all = r.tags));
  });

  const suggestions = $derived(
    input.trim()
      ? all
          .filter(
            (t) =>
              t.name.toLowerCase().includes(input.trim().toLowerCase()) &&
              !selected.some((s) => s.name === t.name),
          )
          .slice(0, 6)
      : [],
  );

  function add(t: DraftTag) {
    if (!selected.some((s) => s.name === t.name)) {
      selected = [...selected, t];
    }
    input = '';
  }

  function onKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' && input.trim()) {
      e.preventDefault();
      const existing = all.find((t) => t.name.toLowerCase() === input.trim().toLowerCase());
      add(existing ? { id: existing.id, name: existing.name } : { name: input.trim() });
    } else if (e.key === 'Backspace' && !input && selected.length > 0) {
      selected = selected.slice(0, -1);
    }
  }
</script>

<div class="glass flex flex-wrap items-center gap-1.5 rounded-lg px-2 py-1.5">
  {#each selected as t (t.name)}
    <span
      class={[
        'flex items-center gap-1 rounded-sm px-1.5 py-0.5 text-xs',
        t.id ? 'bg-ember/15 text-ember-bright' : 'bg-surface-2 text-fg-muted border-line border border-dashed',
      ]}
      title={t.id ? undefined : 'new tag — created on save'}
    >
      #{t.name}
      <button
        type="button"
        class="cursor-pointer opacity-60 hover:opacity-100"
        onclick={() => (selected = selected.filter((s) => s.name !== t.name))}
      >
        ×
      </button>
    </span>
  {/each}
  <input
    type="text"
    bind:value={input}
    onkeydown={onKeydown}
    placeholder={selected.length === 0 ? 'add tags…' : ''}
    class="placeholder:text-fg-muted min-w-20 flex-1 bg-transparent py-0.5 text-sm outline-none"
  />
</div>
{#if suggestions.length > 0}
  <div class="glass mt-1 rounded-lg p-1">
    {#each suggestions as s (s.id)}
      <button
        type="button"
        class="hover:bg-surface-2 block w-full cursor-pointer rounded-sm px-2 py-1 text-left text-sm"
        onclick={() => add({ id: s.id, name: s.name })}
      >
        #{s.name} <span class="text-fg-muted text-xs">{s.kind}</span>
      </button>
    {/each}
  </div>
{/if}
