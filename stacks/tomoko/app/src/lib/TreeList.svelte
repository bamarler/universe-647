<script lang="ts">
  import TreeList from './TreeList.svelte';
  import { router } from './router.svelte';
  import type { TreeNode } from './api';

  let { nodes, depth = 0 }: { nodes: TreeNode[]; depth?: number } = $props();
</script>

<ul class={['m-0 list-none p-0', depth > 0 && 'border-line/70 ml-2.5 border-l pl-4']}>
  {#each nodes as node (node.id)}
    <li>
      <button
        type="button"
        class="hover:glass focus-visible:glass flex w-full cursor-pointer items-center
               gap-2.5 rounded-md px-2 py-2.5 text-left text-[0.95rem]
               transition-shadow hover:shadow-glow-sm focus-visible:outline-none"
        onclick={() => router.go(`/p/${node.id}`)}
      >
        <span
          class={['text-xs', node.kind === 'project' ? 'text-ember' : 'text-ember-bright']}
        >
          {node.kind === 'project' ? '▣' : '◈'}
        </span>
        <span class="flex-1 truncate">{node.name}</span>
        <span class="text-fg-muted flex gap-1.5 text-xs tabular-nums">
          {#if node.task_count > 0}<span>{node.task_count}t</span>{/if}
          {#if node.note_count > 0}<span>{node.note_count}n</span>{/if}
        </span>
      </button>
      {#if node.children.length > 0}
        <TreeList nodes={node.children} depth={depth + 1} />
      {/if}
    </li>
  {/each}
</ul>
