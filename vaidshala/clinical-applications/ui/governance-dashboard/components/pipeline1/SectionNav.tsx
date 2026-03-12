'use client';

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { ChevronRight, ChevronDown, FileText } from 'lucide-react';
import { pipeline1Api } from '@/lib/pipeline1-api';
import { cn } from '@/lib/utils';
import type { TreeNode } from '@/types/pipeline1';

interface SectionNavProps {
  jobId: string;
  selectedSectionId: string | null;
  onSelectSection: (sectionId: string) => void;
}

export function SectionNav({ jobId, selectedSectionId, onSelectSection }: SectionNavProps) {
  const { data: tree, isLoading } = useQuery({
    queryKey: ['pipeline1-tree', jobId],
    queryFn: () => pipeline1Api.context.getTree(jobId),
  });

  if (isLoading) {
    return (
      <div className="p-4 space-y-2">
        {[...Array(6)].map((_, i) => (
          <div key={i} className="h-6 bg-gray-200 rounded animate-pulse" style={{ width: `${70 + Math.random() * 30}%` }} />
        ))}
      </div>
    );
  }

  if (!tree?.treeJson?.length) {
    return (
      <div className="p-4 text-sm text-gray-500">No sections found</div>
    );
  }

  return (
    <nav className="p-2 overflow-y-auto">
      <p className="px-2 py-1 text-xs font-semibold text-gray-400 uppercase tracking-wider">
        Sections
      </p>
      {tree.treeJson.map((node) => (
        <TreeItem
          key={node.id}
          node={node}
          depth={0}
          selectedId={selectedSectionId}
          onSelect={onSelectSection}
        />
      ))}
    </nav>
  );
}

interface TreeItemProps {
  node: TreeNode;
  depth: number;
  selectedId: string | null;
  onSelect: (id: string) => void;
}

function TreeItem({ node, depth, selectedId, onSelect }: TreeItemProps) {
  const [expanded, setExpanded] = useState(depth < 2);
  const hasChildren = node.children && node.children.length > 0;
  const isSelected = selectedId === node.id;

  return (
    <div>
      <button
        onClick={() => {
          onSelect(node.id);
          if (hasChildren) setExpanded(!expanded);
        }}
        className={cn(
          'w-full flex items-center px-2 py-1.5 text-sm rounded-md transition-colors text-left',
          isSelected ? 'bg-blue-50 text-blue-700 font-medium' : 'text-gray-700 hover:bg-gray-100'
        )}
        style={{ paddingLeft: `${depth * 16 + 8}px` }}
      >
        {hasChildren ? (
          expanded ? (
            <ChevronDown className="h-3.5 w-3.5 mr-1.5 flex-shrink-0" />
          ) : (
            <ChevronRight className="h-3.5 w-3.5 mr-1.5 flex-shrink-0" />
          )
        ) : (
          <FileText className="h-3.5 w-3.5 mr-1.5 flex-shrink-0 text-gray-400" />
        )}
        <span className="truncate">{node.heading}</span>
        {node.pageNumber != null && (
          <span className="ml-auto text-xs text-gray-400 flex-shrink-0">p.{node.pageNumber}</span>
        )}
      </button>
      {hasChildren && expanded && (
        <div>
          {node.children!.map((child) => (
            <TreeItem
              key={child.id}
              node={child}
              depth={depth + 1}
              selectedId={selectedId}
              onSelect={onSelect}
            />
          ))}
        </div>
      )}
    </div>
  );
}
