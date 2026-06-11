// Typed client for the sophon API (shapes mirror api/openapi.yaml).

export interface TagRef {
  id: number;
  name: string;
}

export interface Tag {
  id: number;
  name: string;
  kind: 'project' | 'context' | 'tag';
  description?: string;
  parent_id?: number;
  archived_at?: string;
}

export interface Task {
  id: number;
  title: string;
  body_md?: string;
  status: 'open' | 'done';
  priority: number;
  due_at?: string;
  defer_at?: string;
  completed_at?: string;
  project_id?: number;
  project_name?: string;
  tags: TagRef[];
  created_at: string;
  updated_at: string;
}

export interface Note {
  id: number;
  title: string;
  body_md?: string;
  tags: TagRef[];
  created_at: string;
  updated_at: string;
}

export interface TreeNode {
  id: number;
  name: string;
  kind: 'project' | 'context';
  task_count: number;
  note_count: number;
  children: TreeNode[];
}

export interface Folder {
  tag: Tag;
  children: Tag[];
  tasks: Task[];
  notes: Note[];
}

export interface SearchHit {
  source_type: 'task' | 'note' | 'tag';
  source_id: number;
  title: string;
  snippet?: string;
  project?: string;
  score: number;
}

export interface DraftTag {
  id?: number;
  name: string;
}

export interface CommandDraft {
  type: 'task' | 'note' | 'tag';
  title: string;
  body_md?: string;
  kind?: string;
  project_id?: number;
  project_name?: string;
  tags: DraftTag[];
  due_at?: string;
  defer_at?: string;
  priority: number;
}

// TaskFields is the editable surface shared by the task page and draft cards.
export interface TaskFields {
  title: string;
  body_md: string;
  project_id?: number;
  project_name?: string;
  tags: DraftTag[];
  due_at?: string;
  defer_at?: string;
  priority: number;
  status?: 'open' | 'done';
}

export interface CommandResponse {
  action: 'draft' | 'results';
  draft?: CommandDraft;
  hits?: SearchHit[];
}

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
  }
}

async function req<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(path, {
    method,
    headers: { Accept: 'application/json', 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  if (!res.ok) {
    let detail = `${res.status} ${res.statusText}`;
    try {
      const e = (await res.json()) as { detail?: string; title?: string };
      detail = e.detail ?? e.title ?? detail;
    } catch {
      /* keep statusText */
    }
    throw new ApiError(res.status, detail);
  }
  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}

export const api = {
  tree: () => req<{ roots: TreeNode[] }>('GET', '/api/tree'),
  folder: (id: number) => req<Folder>('GET', `/api/folders/${id}`),
  view: (view: string) => req<{ tasks: Task[] }>('GET', `/api/views/${view}`),

  tags: (kind?: string) =>
    req<{ tags: Tag[] }>('GET', kind ? `/api/tags?kind=${kind}` : '/api/tags'),
  createTag: (b: { name: string; kind: string; description?: string; parent_id?: number }) =>
    req<Tag>('POST', '/api/tags', b),

  task: (id: number) => req<Task>('GET', `/api/tasks/${id}`),
  createTask: (b: {
    title: string;
    body_md?: string;
    project_id?: number;
    tag_ids?: number[];
    due_at?: string;
    defer_at?: string;
    priority?: number;
  }) => req<Task>('POST', '/api/tasks', b),
  updateTask: (id: number, b: Record<string, unknown>) =>
    req<Task>('PATCH', `/api/tasks/${id}`, b),
  deleteTask: (id: number) => req<void>('DELETE', `/api/tasks/${id}`),

  note: (id: number) => req<Note>('GET', `/api/notes/${id}`),
  createNote: (b: { title: string; body_md?: string; tag_ids?: number[] }) =>
    req<Note>('POST', '/api/notes', b),
  updateNote: (id: number, b: Record<string, unknown>) =>
    req<Note>('PATCH', `/api/notes/${id}`, b),
  deleteNote: (id: number) => req<void>('DELETE', `/api/notes/${id}`),

  command: (input: string) => req<CommandResponse>('POST', '/api/command', { input }),
  search: (b: Record<string, unknown>) =>
    req<{ hits: SearchHit[] }>('POST', '/api/search', b),
};

export function hitPath(h: SearchHit): string {
  if (h.source_type === 'task') return `/t/${h.source_id}`;
  if (h.source_type === 'note') return `/n/${h.source_id}`;
  return `/p/${h.source_id}`;
}
