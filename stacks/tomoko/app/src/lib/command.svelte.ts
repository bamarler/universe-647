// Command state: the invisible-AI flow. Input goes to /api/command; what
// comes back is either a draft (rendered as an editable card — the user owns
// every field) or hits (navigable links). Nothing else exists.

import { api, type CommandDraft, type SearchHit } from './api';
import { router } from './router.svelte';

class CommandState {
  busy = $state(false);
  error = $state<string | null>(null);
  draft = $state<CommandDraft | null>(null);
  hits = $state<SearchHit[]>([]);
  lastQuery = $state('');

  // navigate: mobile routes to /results; the desktop palette renders hits
  // in place instead.
  async run(input: string, opts: { navigate?: boolean } = {}) {
    const { navigate = true } = opts;
    this.busy = true;
    this.error = null;
    try {
      const res = await api.command(input);
      if (res.action === 'draft' && res.draft) {
        this.draft = res.draft;
      } else {
        this.hits = res.hits ?? [];
        this.lastQuery = input;
        if (navigate) router.go('/results');
      }
    } catch (e) {
      this.error = e instanceof Error ? e.message : String(e);
    } finally {
      this.busy = false;
    }
  }

  dismissDraft() {
    this.draft = null;
  }

  // Saving a draft materializes candidates: ID-less tags are created first
  // (a new project candidate becomes a real project), then the item.
  async saveDraft(): Promise<string | null> {
    const d = this.draft;
    if (!d) return null;
    this.busy = true;
    this.error = null;
    try {
      let projectId = d.project_id;
      if (!projectId && d.project_name) {
        const p = await api.createTag({ name: d.project_name, kind: 'project' });
        projectId = p.id;
      }
      const tagIds: number[] = [];
      for (const t of d.tags) {
        if (t.id) {
          tagIds.push(t.id);
        } else if (t.name.trim()) {
          const created = await api.createTag({ name: t.name.trim(), kind: 'tag' });
          tagIds.push(created.id);
        }
      }

      let path: string;
      if (d.type === 'note') {
        const n = await api.createNote({ title: d.title, body_md: d.body_md, tag_ids: tagIds });
        path = `/n/${n.id}`;
      } else if (d.type === 'tag') {
        const t = await api.createTag({
          name: d.title,
          kind: d.kind || 'tag',
          description: d.body_md,
        });
        path = `/p/${t.id}`;
      } else {
        const t = await api.createTask({
          title: d.title,
          body_md: d.body_md,
          project_id: projectId,
          tag_ids: tagIds,
          due_at: d.due_at,
          defer_at: d.defer_at,
          priority: d.priority,
        });
        path = `/t/${t.id}`;
      }
      this.draft = null;
      return path;
    } catch (e) {
      this.error = e instanceof Error ? e.message : String(e);
      return null;
    } finally {
      this.busy = false;
    }
  }
}

export const command = new CommandState();
