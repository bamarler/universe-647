<script lang="ts">
  import { api, type Tag, type TaskFields } from './api';
  import { toLocalInput, fromLocalInput } from './dates';
  import TagPicker from './TagPicker.svelte';

  let {
    fields = $bindable(),
    compact = false,
  }: {
    fields: TaskFields;
    compact?: boolean;
  } = $props();

  let projects = $state<Tag[]>([]);
  let due = $state(toLocalInput(fields.due_at));
  let defer = $state(toLocalInput(fields.defer_at));

  $effect(() => {
    api.tags('project').then((r) => (projects = r.tags));
  });
  $effect(() => {
    fields.due_at = fromLocalInput(due);
  });
  $effect(() => {
    fields.defer_at = fromLocalInput(defer);
  });
</script>

<div class="flex flex-col gap-3">
  <input
    type="text"
    bind:value={fields.title}
    placeholder="Title"
    class="glass placeholder:text-fg-muted focus:border-ember/60 w-full rounded-lg px-3 py-2.5
           text-base outline-none"
  />

  <textarea
    bind:value={fields.body_md}
    placeholder="Notes (markdown)…"
    rows={compact ? 2 : 6}
    class="glass placeholder:text-fg-muted focus:border-ember/60 w-full resize-y rounded-lg
           px-3 py-2.5 text-sm outline-none"
  ></textarea>

  <div class="grid grid-cols-2 gap-3">
    <label class="flex flex-col gap-1">
      <span class="text-fg-muted text-xs">Due</span>
      <input
        type="datetime-local"
        bind:value={due}
        class="glass focus:border-ember/60 rounded-lg px-2 py-1.5 text-sm outline-none"
      />
    </label>
    <label class="flex flex-col gap-1">
      <span class="text-fg-muted text-xs">Defer until</span>
      <input
        type="datetime-local"
        bind:value={defer}
        class="glass focus:border-ember/60 rounded-lg px-2 py-1.5 text-sm outline-none"
      />
    </label>
  </div>

  <div class="grid grid-cols-2 gap-3">
    <label class="flex flex-col gap-1">
      <span class="text-fg-muted text-xs">
        Project
        {#if !fields.project_id && fields.project_name}
          <span class="text-ember-bright">(new: {fields.project_name})</span>
        {/if}
      </span>
      <select
        bind:value={fields.project_id}
        class="glass focus:border-ember/60 rounded-lg px-2 py-1.5 text-sm outline-none"
      >
        <option value={undefined}>
          {fields.project_name && !fields.project_id
            ? `+ ${fields.project_name}`
            : '(none)'}
        </option>
        {#each projects as p (p.id)}
          <option value={p.id}>{p.name}</option>
        {/each}
      </select>
    </label>
    <label class="flex flex-col gap-1">
      <span class="text-fg-muted text-xs">Priority</span>
      <select
        bind:value={fields.priority}
        class="glass focus:border-ember/60 rounded-lg px-2 py-1.5 text-sm outline-none"
      >
        <option value={0}>none</option>
        <option value={1}>! low</option>
        <option value={2}>!! medium</option>
        <option value={3}>!!! high</option>
      </select>
    </label>
  </div>

  <div class="flex flex-col gap-1">
    <span class="text-fg-muted text-xs">Tags</span>
    <TagPicker bind:selected={fields.tags} />
  </div>
</div>
