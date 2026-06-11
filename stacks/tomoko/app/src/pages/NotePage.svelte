<script lang="ts">
  import { api, type DraftTag } from '../lib/api';
  import { router } from '../lib/router.svelte';
  import TagPicker from '../lib/TagPicker.svelte';

  let { id }: { id: number } = $props();

  let title = $state('');
  let body = $state('');
  let tags = $state<DraftTag[]>([]);
  let loaded = $state(false);
  let error = $state<string | null>(null);
  let saving = $state(false);

  $effect(() => {
    loaded = false;
    api
      .note(id)
      .then((n) => {
        title = n.title;
        body = n.body_md ?? '';
        tags = n.tags.map((t) => ({ id: t.id, name: t.name }));
        loaded = true;
      })
      .catch((e: Error) => (error = e.message));
  });

  async function save() {
    saving = true;
    error = null;
    try {
      const tagIds: number[] = [];
      for (const t of tags) {
        if (t.id) tagIds.push(t.id);
        else if (t.name.trim()) {
          const created = await api.createTag({ name: t.name.trim(), kind: 'tag' });
          tagIds.push(created.id);
        }
      }
      await api.updateNote(id, { title, body_md: body, tag_ids: tagIds });
      router.back();
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      saving = false;
    }
  }

  async function remove() {
    if (!confirm('Delete this note?')) return;
    await api.deleteNote(id);
    router.back();
  }
</script>

{#if error}
  <p class="text-danger">{error}</p>
{/if}
{#if !loaded}
  <p class="text-fg-muted">Loading…</p>
{:else}
  <div class="flex flex-col gap-3">
    <input
      type="text"
      bind:value={title}
      class="glass focus:border-ember/60 w-full rounded-lg px-3 py-2.5 text-base outline-none"
    />
    <textarea
      bind:value={body}
      rows={14}
      placeholder="Markdown…"
      class="glass placeholder:text-fg-muted focus:border-ember/60 w-full resize-y rounded-lg
             px-3 py-2.5 text-sm outline-none"
    ></textarea>
    <TagPicker bind:selected={tags} />
    <div class="flex items-center gap-2">
      <button
        type="button"
        class="bg-ember text-space hover:bg-ember-bright shadow-glow-sm flex-1 cursor-pointer
               rounded-lg py-2.5 font-bold transition-colors disabled:opacity-50"
        disabled={saving || !title.trim()}
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
      <button
        type="button"
        class="text-fg-muted hover:text-danger cursor-pointer px-2 text-sm"
        onclick={remove}
      >
        delete
      </button>
    </div>
  </div>
{/if}
