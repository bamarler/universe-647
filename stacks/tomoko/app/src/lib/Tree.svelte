<script lang="ts">
  import Tree from './Tree.svelte';
  import type { TreeNode } from './api';

  let { nodes, depth = 0 }: { nodes: TreeNode[]; depth?: number } = $props();
</script>

<ul class="tree" class:nested={depth > 0}>
  {#each nodes as node (node.id)}
    <li>
      <button class="row" type="button">
        <span class="kind" data-kind={node.kind}>
          {node.kind === 'project' ? '▣' : '◈'}
        </span>
        <span class="name">{node.name}</span>
        <span class="counts">
          {#if node.task_count > 0}<span>{node.task_count}t</span>{/if}
          {#if node.note_count > 0}<span>{node.note_count}n</span>{/if}
        </span>
      </button>
      {#if node.children.length > 0}
        <Tree nodes={node.children} depth={depth + 1} />
      {/if}
    </li>
  {/each}
</ul>

<style>
  .tree {
    list-style: none;
    margin: 0;
    padding-left: 0;
  }
  .tree.nested {
    padding-left: 1.1rem;
    border-left: 1px solid var(--border);
    margin-left: 0.55rem;
  }
  .row {
    display: flex;
    align-items: center;
    gap: 0.55rem;
    width: 100%;
    padding: 0.55rem 0.5rem;
    background: none;
    border: none;
    border-radius: 0.45rem;
    color: var(--fg);
    font-size: 0.95rem;
    text-align: left;
    cursor: pointer;
  }
  .row:hover,
  .row:focus-visible {
    background: var(--surface-2);
  }
  .kind {
    color: var(--accent);
    font-size: 0.8rem;
  }
  .kind[data-kind='context'] {
    color: var(--accent-2);
  }
  .name {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .counts {
    display: flex;
    gap: 0.4rem;
    color: var(--fg-muted);
    font-size: 0.75rem;
    font-variant-numeric: tabular-nums;
  }
</style>
