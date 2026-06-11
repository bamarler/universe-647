<script lang="ts">
  import { api, type TaskFields } from '../lib/api';
  import { router } from '../lib/router.svelte';
  import TaskEditor from '../lib/TaskEditor.svelte';

  let { id }: { id: number } = $props();

  let fields = $state<TaskFields | null>(null);
  let status = $state<'open' | 'done'>('open');
  let error = $state<string | null>(null);
  let saving = $state(false);

  $effect(() => {
    fields = null;
    api
      .task(id)
      .then((t) => {
        status = t.status;
        fields = {
          title: t.title,
          body_md: t.body_md ?? '',
          project_id: t.project_id,
          project_name: t.project_name,
          tags: t.tags.map((x) => ({ id: x.id, name: x.name })),
          due_at: t.due_at,
          defer_at: t.defer_at,
          priority: t.priority,
        };
      })
      .catch((e: Error) => (error = e.message));
  });

  async function save() {
    if (!fields) return;
    saving = true;
    error = null;
    try {
      // Materialize any new tags first, then patch with the full tag set.
      const tagIds: number[] = [];
      for (const t of fields.tags) {
        if (t.id) tagIds.push(t.id);
        else if (t.name.trim()) {
          const created = await api.createTag({ name: t.name.trim(), kind: 'tag' });
          tagIds.push(created.id);
        }
      }
      await api.updateTask(id, {
        title: fields.title,
        body_md: fields.body_md,
        status,
        priority: fields.priority,
        tag_ids: tagIds,
        ...(fields.due_at ? { due_at: fields.due_at } : { clear_due_at: true }),
        ...(fields.defer_at ? { defer_at: fields.defer_at } : { clear_defer_at: true }),
        ...(fields.project_id ? { project_id: fields.project_id } : { clear_project: true }),
      });
      router.back();
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      saving = false;
    }
  }

  async function remove() {
    if (!confirm('Delete this task?')) return;
    await api.deleteTask(id);
    router.back();
  }
</script>

{#if error}
  <p class="text-danger">{error}</p>
{/if}
{#if !fields}
  <p class="text-fg-muted">Loading…</p>
{:else}
  <div class="flex flex-col gap-4">
    <div class="flex items-center justify-between">
      <button
        type="button"
        class={[
          'cursor-pointer rounded-lg px-3 py-1.5 text-sm transition-colors',
          status === 'done' ? 'bg-ember text-space' : 'glass hover:border-ember/60',
        ]}
        onclick={() => (status = status === 'done' ? 'open' : 'done')}
      >
        {status === 'done' ? '✓ done' : 'mark done'}
      </button>
      <button
        type="button"
        class="text-fg-muted hover:text-danger cursor-pointer text-sm"
        onclick={remove}
      >
        delete
      </button>
    </div>

    <TaskEditor bind:fields />

    <div class="flex gap-2">
      <button
        type="button"
        class="bg-ember text-space hover:bg-ember-bright shadow-glow-sm flex-1 cursor-pointer
               rounded-lg py-2.5 font-bold transition-colors disabled:opacity-50"
        disabled={saving || !fields.title.trim()}
        onclick={save}
      >
        {saving ? 'Saving…' : 'Save'}
      </button>
      <button
        type="button"
        class="glass cursor-pointer rounded-lg px-4 py-2.5"
        onclick={() => router.back()}
      >
        Cancel
      </button>
    </div>
  </div>
{/if}
