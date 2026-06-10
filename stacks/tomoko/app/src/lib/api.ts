// Minimal hand-rolled client for the scaffold. Step 5 replaces this with a
// client generated from sophon's committed api/openapi.yaml.

export interface TreeNode {
  id: number;
  name: string;
  kind: 'project' | 'context';
  task_count: number;
  note_count: number;
  children: TreeNode[];
}

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
  }
}

async function get<T>(path: string): Promise<T> {
  const res = await fetch(path, { headers: { Accept: 'application/json' } });
  if (!res.ok) {
    throw new ApiError(res.status, `${res.status} ${res.statusText}`);
  }
  return res.json() as Promise<T>;
}

export function fetchTree(): Promise<{ roots: TreeNode[] }> {
  return get('/api/tree');
}
