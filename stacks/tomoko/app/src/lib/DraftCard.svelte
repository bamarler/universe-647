<script lang="ts">
  import { command } from './command.svelte';
  import { router } from './router.svelte';
  import TaskEditor from './TaskEditor.svelte';
  import type { TaskFields } from './api';

  // The materialized card: the AI drafted it, the user owns every field.
  let { oncomplete }: { oncomplete?: () => void } = $props();

  const draft = $derived(command.draft);

  // Bridge draft <-> TaskEditor fields (shared shape).
  let fields = $state<TaskFields>({ title: '', body_md: '', tags: [], priority: 0 });
  $effect(() => {
    const d = command.draft;
    if (d) {
      fields = {
        title: d.title,
        body_md: d.body_md ?? '',
        project_id: d.project_id,
        project_name: d.project_name,
        tags: d.tags,
        due_at: d.due_at,
        defer_at: d.defer_at,
        priority: d.priority,
      };
    }
  });

  async function save() {
    const d = command.draft;
    if (!d) return;
    // Write edits back into the draft, then materialize.
    d.title = fields.title;
    d.body_md = fields.body_md;
    d.project_id = fields.project_id;
    d.project_name = fields.project_name;
    d.tags = fields.tags;
    d.due_at = fields.due_at;
    d.defer_at = fields.defer_at;
    d.priority = fields.priority;
    const path = await command.saveDraft();
    if (path) {
      oncomplete?.();
      router.go(path);
    }
  }
</script>

{#if draft}
  <div class="glass shadow-glow flex flex-col gap-3 rounded-xl p-4">
    <div class="flex items-center justify-between">
      <span class="text-ember-bright text-xs tracking-wider uppercase">
        new {draft.type}{draft.type === 'tag' && draft.kind ? ` (${draft.kind})` : ''}
      </span>
      <button
        type="button"
        class="text-fg-muted hover:text-fg cursor-pointer text-sm"
        onclick={() => command.dismissDraft()}
      >
        ✕
      </button>
    </div>

    {#if draft.type === 'task'}
      <TaskEditor bind:fields compact />
    {:else}
      <input
        type="text"
        bind:value={fields.title}
        class="glass focus:border-ember/60 w-full rounded-lg px-3 py-2.5 outline-none"
      />
      <textarea
        bind:value={fields.body_md}
        rows={3}
        placeholder={draft.type === 'note' ? 'Markdown…' : 'Description…'}
        class="glass placeholder:text-fg-muted focus:border-ember/60 w-full resize-y rounded-lg
               px-3 py-2.5 text-sm outline-none"
      ></textarea>
    {/if}

    {#if command.error}
      <p class="text-danger text-sm">{command.error}</p>
    {/if}

    <button
      type="button"
      class="bg-ember text-space hover:bg-ember-bright shadow-glow-sm cursor-pointer rounded-lg
             py-2.5 font-bold transition-colors disabled:opacity-50"
      disabled={command.busy || !fields.title.trim()}
      onclick={save}
    >
      {command.busy ? 'Saving…' : 'Save'}
    </button>
  </div>
{/if}
